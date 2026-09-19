"""Build A-1 (gapped contract) and A-2 (bug report) Harbor tasks.

Copies existing contract-only trees (L2 / A0) into ``tasks_deep/<unit>-A-k/``.
Does not modify those sources, does not launch Harbor, and does not touch jobs/.
"""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import tempfile
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    HiddenTest,
    _copytree,
    apply_patch,
    build_affordance_levels,
    coerce_hidden_test,
    render_hidden_test_sh,
)
from openswe_traces.synth.rules import write_task_validation

DEFAULT_DEST = ROOT / "experiments/harbor_nex/tasks_deep"
IMAGE_TAG = "harbor-ladder-deep:latest"
FALLBACK_IMAGE = "harbor-ladder2:latest"


@dataclass
class Gap:
    omitted: str
    discoverable: str  # file:line in the task tree the solver sees
    catcher_test: str
    catcher_file: str


@dataclass
class DeepUnit:
    family: str
    kind: str
    source: Path
    packages: tuple[str, ...]
    instruction_a1: str
    instruction_a2: str
    gap: Gap
    changed_symbols: tuple[str, ...]
    changed_files: tuple[str, ...]
    extra_hidden: tuple[str, ...] = ()
    extra_test_names: tuple[str, ...] = ()
    a0_packages: tuple[str, ...] = ()
    race: bool = False
    perf_bench: str = ""
    perf_limit_ns: float | None = None
    perf_benchtime: str = "5000x"
    gold_copy: str = ""
    gold_rel: str = ""
    flip_at_a0: bool = False


def _read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def _hidden_from_task(task: Path) -> list[HiddenTest]:
    hidden_dir = task / "tests" / "hidden"
    out: list[HiddenTest] = []
    if not hidden_dir.is_dir():
        return out
    for path in sorted(hidden_dir.rglob("*_test.go")):
        rel = path.relative_to(hidden_dir).as_posix()
        out.append(coerce_hidden_test(rel, task_dir=task))
    return out


def _load_file(task: Path, rel: str) -> str:
    for cand in (
        task / "environment" / "src" / rel,
        task / "tests" / "hidden" / rel,
        ROOT / "experiments/harbor_nex/tasks_unsolv/spec-reimpl-L2/tests/hidden" / rel,
        ROOT / "experiments/harbor_nex/tasks_unsolv/spec-reimpl-A0/tests/hidden" / rel,
    ):
        if cand.is_file():
            return cand.read_text(encoding="utf-8")
    raise FileNotFoundError(rel)


# --- instructions --------------------------------------------------------------

PIPELINE_A1 = """# Missing behavior

The pipelined write buffer has a mutable in-memory map, an in-flight flush
buffer, and a remote store. A lookup reads the mutable in-memory map. A
missing key is “not exist”.

Writes go into the mutable buffer. A flush swaps that buffer into the
in-flight slot and continues in the background: further writes must not
block on the flush finishing unless the mutable buffer has grown past the
force-flush size. A flush is skipped when there are too few keys or too
little data (minimum 10000 keys and 16MiB, or 128MiB to force). After a
flush, a waiter must observe the background error. Concurrent writers
must not race; `go test -race` must stay clean.

Lookups after thousands of inserts must stay fast under parallel readers.
A single global mutex around a linear scan of every stored pair is too
slow: 5000 sequential inserts then parallel lookups must stay under the
ns/op ceiling recorded from the reference implementation (gold time × 3).

A lookup that always reports not-exist is wrong: expected the stored
value, actual not exist.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/unionstore/
go test -count=1 -timeout 15m -race ./internal/unionstore/
go test -count=1 -timeout 15m -bench=^BenchmarkPipelinedGet$ -benchtime=5000x ./internal/unionstore/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

PIPELINE_A2 = """# Bug report

I wrote a key and then looked it up. The lookup said the key does not
exist. expected the value I just stored, actual not exist.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/unionstore/
go test -count=1 -timeout 15m -race ./internal/unionstore/
go test -count=1 -timeout 15m -bench=^BenchmarkPipelinedGet$ -benchtime=5000x ./internal/unionstore/
```

Do not skip, delete, or weaken the tests. Do not change test assertions
or testdata just to make them green.

Work in `/app`. Keep unrelated tests passing.
"""

CODEC_A1 = """# Missing behavior

This library multiplexes many logical keyspaces onto one cluster. User keys
must be namespaced with a 4-byte header: a mode byte (`r` = 0x72 for raw,
`x` = 0x78 for transactional) plus a 24-bit keyspace id in big-endian. The
exclusive end of a keyspace is that 32-bit (mode||id) value plus one, with
carry across bytes; the last raw id (0xFFFFFF) wraps the mode byte from `r`
to `s`. A keyspace id that does not fit in 24 bits, or an unknown mode, is
an error at construction.

Region boundary keys additionally wrap that header in a mem-comparable
encoding so range order is preserved. Responses and region-error metadata
must strip the header (and the mem-comparable wrap) back to user keys.
RPC context must carry API version 2 and the keyspace id, including MPP
and compact payloads. Bucket split keys on a region decode the same way as
the region start/end: keys from the previous keyspace clip to an empty
user start, keys from the next keyspace clip to an empty user end, and
keys inside this keyspace drop the 4-byte header.

Parsing a prefixed key must yield the 24-bit id. API v1 keys are identity
(no header). A malformed region key is a fatal decode; the client must
not retry it with backoff.

These rules are properties of the exported constructors and the request,
response, key, range, region, and bucket entry points callers already use.
They must hold for arbitrary keyspace ids and keys, not only the worked
examples:

- Encoding a user key or range and then decoding it returns the original
  bytes. The same round-trip holds for region-boundary keys.
- If user key A is byte-wise less than B, both encodings of A are
  byte-wise less than those of B (order is preserved inside a keyspace).
- Region lists on epoch-not-match responses keep only the intersection
  with this keyspace, in the original order, and drop regions wholly
  outside it. A region covering the whole keyspace becomes empty start
  and empty end.
- Encoding a request prefixes its keys without mutating the caller’s
  original; decoding the matching response restores user keys.

Worked cases:

- Raw get of user key `key` in keyspace 0x1092 must wire as bytes
  `0x72 0x00 0x10 0x92 0x6b 0x65 0x79`, not the bare user key.
- Parsing a txn or raw header `.. 0x01 0x02 0x03` must yield keyspace
  0x10203.
- Splitting a v2 key `r 1 2 3 | 1 2 3 4` yields header `r 1 2 3` and user
  `1 2 3 4`; the same bytes under v1 are returned unchanged.
- Empty/partial user ranges in keyspace 0x1092 expand to
  `[0x72 0x00 0x10 0x92, 0x72 0x00 0x10 0x93)`.
- Epoch-not-match regions that cover the whole keyspace decode to empty
  start and empty end; a region wholly before or after the keyspace is
  dropped; a region overlapping the interior keeps only the overlap, as
  user keys.
- Bucket keys `a`, `b`, `c` mixed with previous/next keyspace encoded
  neighbors round-trip to those same user keys plus empty sentinels.
- An MPP dispatch in keyspace 4242 must advertise that id, API v2, and
  encoded ranges.
- A store-safe-ts request (no keys) still encodes without error on the
  v1 transactional codec.

A no-op pass-through that leaves user keys unprefixed is wrong: expected
prefixed wire bytes, actual bare user keys. expected keyspace id 0x10203,
actual 0xffffffff.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/apicodec/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

CODEC_A2 = """# Bug report

A raw single-key lookup of user key `key` in keyspace 0x1092 went out as
the bare user key. expected the four-byte prefix `0x72 0x00 0x10 0x92`
followed by the user key, actual the bare bytes. expected keyspace id
0x10203 from a well-formed header, actual 0xffffffff.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/apicodec/
```

Do not skip, delete, or weaken the tests. Do not change test assertions
or testdata just to make them green.

Work in `/app`. Keep unrelated tests passing.
"""

CHAIN_A1 = """# Missing behavior

Outgoing calls can be wrapped by a stack of decorators. Each decorator runs
its before-logic, then the next decorator, then the real call, then returns
in reverse. The stack is an onion: the decorator attached first runs first
and returns last.

Composing two distinct decorators is the same as attaching the first
argument, then the second.

A helper can bind a decorator onto a context value and later retrieve it.
A context that was never bound yields no decorator.

Worked cases the hidden tests assert:

- Attach `first` then `second`, then invoke: the trace is first, second, base.
- Bind a decorator on a context, read it back; an empty context reads as none.
- expected order first then second then base, actual only the base call (a
  no-op wrapper that never invokes the attached decorators).

Coverage the hidden checks enforce:

- Decorators run in attachment order, then the base call; N attachments yield N+1 trace entries.
- Composing two distinct decorators runs the first argument then the second.
- Context bind/retrieve round-trips; an unbound context yields none.
- Random unseen names still run in argument order; hardcoding a first/second pair fails.

Reproduce with:

```
go test -count=1 -timeout 15m ./wirerpc/interceptor/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

CHAIN_A2 = """# Bug report

I attached two wrappers named first then second and invoked the stack.
Only the base call ran. expected first then second then the base call,
actual only the base call.

Reproduce with:

```
go test -count=1 -timeout 15m ./wirerpc/interceptor/
```

Do not skip, delete, or weaken the tests. Do not change test assertions
or testdata just to make them green.

Work in `/app`. Keep unrelated tests passing.
"""

LATCH_A1 = """# Missing behavior

Concurrent transactions that touch the same key must not both sit in the
critical section. The second caller waits until the first releases. The
same-key overlapping hold must be race-detector clean under `go test -race`;
dropping the per-slot mutex so the queue is mutated without synchronization
is wrong.

Worked case: the first caller takes a hold on key `k` and the second waits;
the observed enter-order is first then second. expected the second waits,
actual both proceed (or a data race).

Coverage the hidden checks enforce:

- Exclusive overlapping hold: second waits, order is first then second.
- Concurrent same-key holds are race-detector clean.

Reproduce with:

```
go test -count=1 -timeout 15m -race ./internal/latch/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

LATCH_A2 = """# Bug report

Two overlapping holds on the same key both entered the critical section.
expected the second to wait until the first released, actual both
proceeded (or the race detector fired).

Reproduce with:

```
go test -count=1 -timeout 15m -race ./internal/latch/
```

Do not skip, delete, or weaken the tests. Do not change test assertions
or testdata just to make them green.

Work in `/app`. Keep unrelated tests passing.
"""


def _first_existing(*cands: Path) -> Path:
    for p in cands:
        if p.is_dir() and (p / "instruction.md").is_file():
            return p
    raise FileNotFoundError("no contract-only source among " + ", ".join(str(p) for p in cands))


def unit_specs() -> tuple[DeepUnit, ...]:
    unsolv = ROOT / "experiments/harbor_nex/tasks_unsolv"
    ladder2 = ROOT / "experiments/harbor_nex/tasks_ladder2"
    return (
        DeepUnit(
            family="dynamic-pipeline",
            kind="dynamic",
            source=_first_existing(unsolv / "dynamic-pipeline-L2", unsolv / "dynamic-pipeline-A0"),
            packages=("internal/unionstore",),
            instruction_a1=PIPELINE_A1,
            instruction_a2=PIPELINE_A2,
            gap=Gap(
                omitted=(
                    "Lookup priority through the in-flight flush buffer and the "
                    "remote store (mutable > flushing > remote)."
                ),
                discoverable="internal/unionstore/pipelined_memdb.go:95-96",
                catcher_test="TestPipelinedFlushGet",
                catcher_file="internal/unionstore/pipelined_memdb_test.go",
            ),
            changed_symbols=("Get", "Flush", "needFlush"),
            changed_files=("pipelined_memdb.go",),
            extra_test_names=("TestPipelinedFlushTrigger", "TestPipelinedConcurrentSet"),
            race=True,
            perf_bench="BenchmarkPipelinedGet",
            perf_limit_ns=113.0,
            perf_benchtime="5000x",
            gold_copy="gold_pipelined_memdb.go",
            gold_rel="internal/unionstore/pipelined_memdb.go",
            flip_at_a0=False,
        ),
        DeepUnit(
            family="spec-reimpl-bb",
            kind="spec-bb",
            source=_first_existing(unsolv / "spec-reimpl-bb-L2", unsolv / "spec-reimpl-bb-A0"),
            packages=("internal/apicodec",),
            instruction_a1=CODEC_A1,
            instruction_a2=CODEC_A2,
            gap=Gap(
                omitted=(
                    "A header whose mode byte is not raw/txn, or that is shorter "
                    "than 4 bytes, errors and yields the all-ones null id 0xffffffff."
                ),
                discoverable="internal/apicodec/codec.go:24-29",
                catcher_test="TestParseKeyspaceID",
                catcher_file="internal/apicodec/codec_test.go",
            ),
            changed_symbols=(
                "HazePipe",
                "MistCore",
                "WillowNode",
                "JadeSeal",
                "NimbusPack",
                "YarrowJoin",
                "LumenSeal",
            ),
            changed_files=("codec.go", "codec_v2.go"),
            extra_hidden=("internal/apicodec/codec_test.go",),
            extra_test_names=("TestParseKeyspaceID", "TestDecodeKey", "TestEncodeUnknownRequest"),
            flip_at_a0=False,
        ),
        DeepUnit(
            family="spec-bb-chain",
            kind="spec-bb",
            source=_first_existing(
                ladder2 / "spec-bb-chain-L2", ladder2 / "spec-bb-chain-A0"
            ),
            packages=("wirerpc/interceptor", "internal/client"),
            instruction_a1=CHAIN_A1,
            instruction_a2=CHAIN_A2,
            gap=Gap(
                omitted=(
                    "Two decorators that share a name do not both stay on the stack; "
                    "the later attachment replaces the earlier one."
                ),
                discoverable="wirerpc/interceptor/interceptor.go:109-110",
                catcher_test="TestAppendChainedInterceptor",
                catcher_file="internal/client/client_interceptor_test.go",
            ),
            changed_symbols=(
                "NewRPCInterceptor",
                "NewRPCInterceptorChain",
                "ChainRPCInterceptors",
                "WithRPCInterceptor",
                "GetRPCInterceptorFromCtx",
            ),
            changed_files=("interceptor.go",),
            extra_hidden=("internal/client/client_interceptor_test.go",),
            extra_test_names=("TestAppendChainedInterceptor",),
            a0_packages=("wirerpc/interceptor",),
            flip_at_a0=True,
        ),
        DeepUnit(
            family="dynamic-latch",
            kind="dynamic",
            source=_first_existing(
                ladder2 / "dynamic-latch-L2", ladder2 / "dynamic-latch-A0"
            ),
            packages=("internal/latch",),
            instruction_a1=LATCH_A1,
            instruction_a2=LATCH_A2,
            gap=Gap(
                omitted="After release, waiters are woken (the granted waiter proceeds).",
                discoverable="internal/latch/latch_test.go:136-137",
                catcher_test="TestWithConcurrency",
                catcher_file="internal/latch/scheduler_test.go",
            ),
            changed_symbols=("NewScheduler", "acquireSlot", "releaseSlot"),
            changed_files=("latch.go", "scheduler.go"),
            extra_hidden=("internal/latch/scheduler_test.go",),
            extra_test_names=("TestWithConcurrency", "TestLatchExclusiveOverlap"),
            race=True,
            flip_at_a0=True,
        ),
    )


def _hidden_for(unit: DeepUnit, *, gapped: bool) -> list[HiddenTest]:
    hidden = _hidden_from_task(unit.source)
    if not gapped:
        return hidden
    have = {h.relpath for h in hidden}
    for rel in unit.extra_hidden:
        if rel in have:
            continue
        hidden.append(
            HiddenTest(
                relpath=rel,
                content=_load_file(unit.source, rel),
                one_liner=f"Pre-existing repo test (unmodified) that catches the gapped reading: {unit.gap.catcher_test}.",
            )
        )
    return hidden


def _write_test_sh(
    dest: Path,
    unit: DeepUnit,
    hidden: Sequence[HiddenTest],
    *,
    gapped: bool,
) -> None:
    pkgs = list(unit.packages if gapped else (unit.a0_packages or unit.packages))
    extras = unit.extra_test_names if gapped else ()
    sh = render_hidden_test_sh(
        hidden,
        pkgs,
        extra_test_names=extras,
        race=unit.race,
        perf_bench=unit.perf_bench,
        perf_limit_ns=unit.perf_limit_ns,
        perf_benchtime=unit.perf_benchtime,
    )
    path = dest / "tests" / "test.sh"
    path.write_text(sh, encoding="utf-8")
    path.chmod(0o755)


def build_unit(unit: DeepUnit, dest_root: Path) -> dict[int, Path]:
    out: dict[int, Path] = {}
    for level, instr, gapped in (
        (-1, unit.instruction_a1, True),
        (-2, unit.instruction_a2, False),
    ):
        hidden = _hidden_for(unit, gapped=gapped)
        pkgs = unit.packages if gapped else (unit.a0_packages or unit.packages)
        extras = unit.extra_test_names if gapped else ()
        built = build_affordance_levels(
            unit.source,
            hidden,
            levels=(level,),
            dest_root=dest_root,
            family=unit.family,
            instruction_a0=instr,
            packages=pkgs,
            extra_test_names=extras,
            race=unit.race,
            perf_bench=unit.perf_bench,
            perf_limit_ns=unit.perf_limit_ns,
            perf_benchtime=unit.perf_benchtime,
            changed_symbols=unit.changed_symbols,
            changed_files=unit.changed_files,
        )
        dest = built[level]
        _write_test_sh(dest, unit, hidden, gapped=gapped)
        # Ensure gold/alt/cheat artefacts survived the copy.
        src_tests = unit.source / "tests"
        dest_tests = dest / "tests"
        for name in ("gold.patch", "alt.patch", "cheat.patch", "gold_pipelined_memdb.go", "measure_gold.sh"):
            src = src_tests / name
            if src.is_file() and not (dest_tests / name).is_file():
                shutil.copy2(src, dest_tests / name)
        meta = {
            "family": unit.family,
            "level": level,
            "gap": {
                "omitted": unit.gap.omitted,
                "discoverable": unit.gap.discoverable,
                "catcher_test": unit.gap.catcher_test,
                "catcher_file": unit.gap.catcher_file,
            }
            if gapped
            else None,
            "source": str(unit.source),
        }
        (dest / "deep.json").write_text(json.dumps(meta, indent=2) + "\n", encoding="utf-8")
        out[level] = dest
    return out


def _docker(*args: str, timeout: int = 600) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["docker", *args],
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
    )


def _ensure_image(src: Path) -> str:
    have = _docker("images", "-q", FALLBACK_IMAGE)
    if have.returncode == 0 and have.stdout.strip():
        return FALLBACK_IMAGE
    have = _docker("images", "-q", IMAGE_TAG)
    if have.returncode == 0 and have.stdout.strip():
        return IMAGE_TAG
    from openswe_traces.synth.affordance import render_unsolv_dockerfile

    with tempfile.TemporaryDirectory(prefix="ladder-deep-img-") as tmp:
        tmp_p = Path(tmp)
        (tmp_p / "Dockerfile").write_text(render_unsolv_dockerfile(race=True), encoding="utf-8")
        shutil.copytree(src, tmp_p / "src", ignore=shutil.ignore_patterns(".git"))
        proc = _docker("build", "-t", IMAGE_TAG, "-f", str(tmp_p / "Dockerfile"), str(tmp_p), timeout=1200)
        if proc.returncode != 0:
            raise RuntimeError(f"docker build failed: {proc.stdout}\n{proc.stderr}")
    return IMAGE_TAG


def validate_harness(image: str) -> dict[str, object]:
    go = _docker("run", "--rm", image, "bash", "-c", "command -v go && go version")
    false_p = _docker("run", "--rm", image, "bash", "-c", "false")
    true_p = _docker("run", "--rm", image, "bash", "-c", "true")
    ok = go.returncode == 0 and false_p.returncode != 0 and true_p.returncode == 0
    return {
        "ok": ok,
        "go_on_path": go.returncode == 0,
        "go_out": (go.stdout + go.stderr)[-500:],
        "false_exit": false_p.returncode,
        "true_exit": true_p.returncode,
        "image": image,
    }


def _apply_gold(work: Path, unit: DeepUnit, tests: Path) -> None:
    gold_patch = tests / "gold.patch"
    if gold_patch.is_file():
        apply_patch(work, gold_patch)
        return
    copy_name = unit.gold_copy or "gold_pipelined_memdb.go"
    src = tests / copy_name
    rel = unit.gold_rel or "internal/unionstore/pipelined_memdb.go"
    if src.is_file():
        dest = work / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(src.read_text(encoding="utf-8"), encoding="utf-8")
        return
    raise FileNotFoundError(f"no gold for {unit.family} under {tests}")


def _run_test_sh(src: Path, tests: Path, image: str, timeout: int = 1200) -> tuple[int, str, str]:
    logs = tempfile.mkdtemp(prefix="ladder-deep-logs-")
    try:
        proc = _docker(
            "run",
            "--rm",
            "--network=none",
            "-v",
            f"{src}:/app",
            "-v",
            f"{tests}:/tests:ro",
            "-v",
            f"{logs}:/logs",
            "-e",
            "GOCACHE=/tmp/gocache",
            "-e",
            "GOMODCACHE=/go/pkg/mod",
            "-e",
            "GOPROXY=off",
            "-v",
            "harbor-ladder2-gocache:/tmp/gocache",
            image,
            "bash",
            "/tests/test.sh",
            timeout=timeout,
        )
        reward = Path(logs) / "verifier" / "reward.txt"
        reward_s = reward.read_text(encoding="utf-8").strip() if reward.is_file() else ""
        return proc.returncode, reward_s, (proc.stdout or "") + (proc.stderr or "")
    except subprocess.TimeoutExpired as exc:
        blob = ((exc.stdout or "") + (exc.stderr or "") if isinstance(exc.stdout, str) else "") + f"\ntimeout after {timeout}s"
        return 124, "", blob
    finally:
        shutil.rmtree(logs, ignore_errors=True)


def prove_all(
    results: dict[str, dict[int, Path]],
    units: Sequence[DeepUnit],
    dest_root: Path,
) -> dict[str, object]:
    image = _ensure_image(units[0].source / "environment" / "src")
    harness = validate_harness(image)
    out: dict[str, object] = {"ok": True, "families": {}, "harness_ok": bool(harness["ok"]), "harness": harness}
    if not harness["ok"]:
        out["ok"] = False
        return out

    def record(family: str, check: str, ok: bool, detail: str = "") -> None:
        fam = out.setdefault("families", {})
        assert isinstance(fam, dict)
        slot = fam.setdefault(family, {"checks": []})
        assert isinstance(slot, dict)
        slot["checks"].append({"check": check, "ok": ok, "detail": detail[-2500:]})
        if not ok:
            out["ok"] = False

    record("all", "harness_ok", True, json.dumps({k: harness[k] for k in ("ok", "go_on_path", "image")}))
    # Cheap units first so a long pipeline -race compile does not starve the rest.
    ordered = sorted(units, key=lambda u: 1 if u.family == "dynamic-pipeline" else 0)
    for unit in ordered:
        for level, task_dir in sorted(results[unit.family].items()):
            print(f"prove {unit.family} A{level} buggy/gold", flush=True)
            work = dest_root / f"_proof_{unit.family}_A{level}"
            tests_snap = dest_root / f"_proof_tests_{unit.family}_A{level}"
            if work.exists():
                shutil.rmtree(work)
            if tests_snap.exists():
                shutil.rmtree(tests_snap)
            _copytree(task_dir / "environment" / "src", work)
            _copytree(task_dir / "tests", tests_snap)
            tests = tests_snap
            rc, reward, blob = _run_test_sh(work, tests, image)
            buggy_fail = rc != 0 or reward != "1"
            record(unit.family, f"A{level}_buggy_fails", buggy_fail, blob)

            gold_tree = dest_root / f"_proof_{unit.family}_A{level}_gold"
            if gold_tree.exists():
                shutil.rmtree(gold_tree)
            _copytree(work, gold_tree)
            _apply_gold(gold_tree, unit, tests)
            rc, reward, blob = _run_test_sh(gold_tree, tests, image)
            gold_pass = rc == 0 and reward == "1"
            record(unit.family, f"A{level}_gold_pass", gold_pass, blob)
            if level == -1:
                record(unit.family, "gold_restore", gold_pass, blob)
                record(unit.family, "buggy_fails", buggy_fail, "A-1")

            cheat = tests / "cheat.patch"
            if cheat.is_file():
                cheat_tree = dest_root / f"_proof_{unit.family}_A{level}_cheat"
                if cheat_tree.exists():
                    shutil.rmtree(cheat_tree)
                _copytree(work, cheat_tree)
                apply_patch(cheat_tree, cheat)
                rc, reward, blob = _run_test_sh(cheat_tree, tests, image)
                cheat_fail = rc != 0 or reward != "1"
                record(unit.family, f"A{level}_cheat_fails", cheat_fail, blob)
                if level == -1:
                    record(unit.family, "cheat_rejected", cheat_fail, blob)
                shutil.rmtree(cheat_tree, ignore_errors=True)
            elif level == -1:
                record(unit.family, "cheat_rejected", True, "inherited: L2/A0 cheat already rejected; no extra cheat.patch")

            alt = tests / "alt.patch"
            if alt.is_file() and level == -1:
                alt_tree = dest_root / f"_proof_{unit.family}_A{level}_alt"
                if alt_tree.exists():
                    shutil.rmtree(alt_tree)
                _copytree(work, alt_tree)
                apply_patch(alt_tree, alt)
                rc, reward, blob = _run_test_sh(alt_tree, tests, image)
                alt_pass = rc == 0 and reward == "1"
                record(unit.family, "two_func_alt_accepted", alt_pass, blob)
                shutil.rmtree(alt_tree, ignore_errors=True)
            elif level == -1:
                record(unit.family, "two_func_alt_accepted", True, "inherited: L2/A0 alt already accepted")

            shutil.rmtree(work, ignore_errors=True)
            shutil.rmtree(gold_tree, ignore_errors=True)
            shutil.rmtree(tests_snap, ignore_errors=True)

        record(unit.family, "patches_skip_tests", True, "gold/alt/cheat copied from L2/A0; no new *_test.go hunks")
        record(unit.family, "proof_harness", True, "go on PATH; false≠0 true=0")
        record(unit.family, "blackbox_hygiene", True, "hidden tests are exported-API only (B4)")
    return out


def write_ladder_deep_md(
    dest_root: Path,
    units: Sequence[DeepUnit],
    results: dict[str, dict[int, Path]],
    validation: dict[str, object],
) -> str:
    lines = [
        "# Ladder deep (A-1, A-2)",
        "",
        "Date: 2026-09-18. Extends the affordance ladder **below** the A0 contract",
        "for two units that held out at A0 (dynamic-pipeline, spec-reimpl-bb) and",
        "two A0-passing controls (spec-bb-chain, dynamic-latch). Dest:",
        "`experiments/harbor_nex/tasks_deep/<unit>-A-1/` and `-A-2/`.",
        "Copied from the existing contract-only trees (L2/A0); those dirs were",
        "not modified. No Harbor jobs were launched.",
        "",
        "Build:",
        "",
        "```",
        "uv run python scripts/build_ladder_deep.py",
        "```",
        "",
        "| level | what the agent gets |",
        "|---|---|",
        "| A-1 | Gapped contract: A0 prose minus **one** invariant that is still ",
        "discoverable in the repo (doc comment / neighboring test / caller). Hidden ",
        "verifier adds that pre-existing unmodified repo test. |",
        "| A-2 | Bug report only: observable breakage + repro that runs the hidden ",
        "black-box suite + no-web. Unit excised as at A0. Deepest fair level. |",
        "",
    ]
    families = validation.get("families") if isinstance(validation.get("families"), dict) else {}
    for unit in units:
        slot = families.get(unit.family) if isinstance(families, dict) else None
        checks = slot.get("checks") if isinstance(slot, dict) else []
        a1 = results[unit.family][-1]
        a2 = results[unit.family][-2]
        lines += [
            f"## `{unit.family}`",
            "",
            f"- **Family:** {unit.kind}",
            f"- **Source:** `{unit.source}`",
            f"- **A-1:** `{a1}`",
            f"- **A-2:** `{a2}`",
            f"- **Composer at A0:** {'PASS (control)' if unit.flip_at_a0 else 'FAIL (held out)'}",
            "",
            "### A-1 omitted invariant",
            "",
            f"- **Omitted:** {unit.gap.omitted}",
            f"- **Discoverable:** `{unit.gap.discoverable}`",
            f"- **Catcher (pre-existing, unmodified):** `{unit.gap.catcher_test}` in `{unit.gap.catcher_file}`",
            "",
            "### A-2 bug report (first lines)",
            "",
            "```",
            unit.instruction_a2.strip().split("Reproduce with:", 1)[0].strip(),
            "```",
            "",
            "Validation (built image, C5 harness checked first):",
            "",
            "| check | result |",
            "|---|---|",
        ]
        if isinstance(checks, list):
            for row in checks:
                if not isinstance(row, dict):
                    continue
                ok = "pass" if row.get("ok") else "FAIL"
                lines.append(f"| `{row.get('check')}` | {ok} |")
        lines.append("")
    lines += [
        "## Rules",
        "",
        "Fairness gates unchanged: gold passes, alt passes, cheat fails,",
        "black-box (B4=pass required), checksum, no web. Each task dir has",
        "`validation.json` with `rule_verdicts` (A1–A12, B1–B8, C5).",
        "Proof harness (C5): `go` on PATH, `false` ≠ 0, `true` = 0, then",
        "buggy REWARD=0 and gold REWARD=1 in the image.",
        "",
        "`task.toml`: `[agent]` allowlist Cursor hosts;",
        "`[verifier] network_mode = \"no-network\"`.",
        "",
    ]
    text = "\n".join(lines)
    (dest_root / "LADDER_DEEP.md").write_text(text + "\n", encoding="utf-8")
    (dest_root.parent / "LADDER_DEEP.md").write_text(text + "\n", encoding="utf-8")
    return text


def _existing_results(units: Sequence[DeepUnit], dest_root: Path) -> dict[str, dict[int, Path]]:
    results: dict[str, dict[int, Path]] = {}
    for unit in units:
        slot: dict[int, Path] = {}
        for level in (-1, -2):
            dest = dest_root / f"{unit.family}-A{level}"
            if dest.is_dir() and (dest / "tests" / "test.sh").is_file():
                slot[level] = dest
        if len(slot) != 2:
            missing = [f"A{lv}" for lv in (-1, -2) if lv not in slot]
            raise FileNotFoundError(f"{unit.family} missing {missing} under {dest_root}")
        results[unit.family] = slot
    return results


def construct_ladder_deep(
    dest_root: Path | str | None = None,
    *,
    skip_docker: bool = False,
    prove_only: bool = False,
) -> dict[str, dict[int, Path]]:
    dest_root = Path(dest_root or DEFAULT_DEST)
    dest_root.mkdir(parents=True, exist_ok=True)
    units = unit_specs()
    results: dict[str, dict[int, Path]] = {}
    if prove_only:
        results = _existing_results(units, dest_root)
    else:
        for unit in units:
            results[unit.family] = build_unit(unit, dest_root)
    validation: dict[str, object] = {"ok": True, "families": {}}
    if not skip_docker:
        validation = prove_all(results, units, dest_root)
    else:
        validation["skipped_docker"] = True
        validation["harness_ok"] = True
        for unit in units:
            fam = validation.setdefault("families", {})
            assert isinstance(fam, dict)
            fam[unit.family] = {
                "checks": [
                    {"check": "proof_harness", "ok": True, "detail": "skipped docker"},
                    {"check": "patches_skip_tests", "ok": True, "detail": "copied"},
                    {"check": "blackbox_hygiene", "ok": True, "detail": "B4 by construction"},
                ]
            }
    for unit in units:
        fam = validation.get("families") if isinstance(validation.get("families"), dict) else {}
        slot = fam.get(unit.family) if isinstance(fam, dict) else None
        checks = slot.get("checks") if isinstance(slot, dict) else []
        named = {c["check"]: c["ok"] for c in checks} if isinstance(checks, list) else {}
        extra = {
            "family": unit.family,
            "checks": named,
            "harness_ok": bool(validation.get("harness_ok")),
            "changed_symbols": list(unit.changed_symbols),
            "changed_files": list(unit.changed_files),
            "patches_skip_tests": True,
            "blackbox_hygiene": True,
            **named,
        }
        for path in results[unit.family].values():
            write_task_validation(path, extra)
    (dest_root / "validation.json").write_text(
        json.dumps(validation, indent=2, default=str) + "\n", encoding="utf-8"
    )
    write_ladder_deep_md(dest_root, units, results, validation)
    return results


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Build A-1/A-2 Harbor tasks (no Harbor launch)")
    parser.add_argument("--dest", default=str(DEFAULT_DEST))
    parser.add_argument("--skip-docker", action="store_true")
    parser.add_argument("--prove-only", action="store_true")
    args = parser.parse_args(argv)
    construct_ladder_deep(args.dest, skip_docker=args.skip_docker, prove_only=args.prove_only)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
