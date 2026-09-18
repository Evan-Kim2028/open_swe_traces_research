"""Affordance ladder: copy a Harbor task and add information level by level.

A0 is a bare contract plus a tree that does not contain the hidden tests.
A1 names those tests. A2 adds exported stubs with godoc. A3 restores one
hidden test file into the tree. A4 restores all of them (the control).
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
from collections.abc import Mapping, Sequence
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.harbor_tasks import (
    AGENT_TIMEOUT_HARD_SEC,
    BUILD_TIMEOUT_SEC,
    VERIFIER_TIMEOUT_SEC,
    instruction_self_check,
    parse_bench_ns_op,
)
from openswe_traces.synth.obfuscate import (
    NO_WEB_CLAUSE,
    distinctive_identifiers,
    searchability_check,
)

HIDDEN_DIR = "hidden"
_TEST_FUNC_RE = re.compile(r"^func\s+(?:\(.*?\)\s+)?(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)


@dataclass(frozen=True)
class HiddenTest:
    """One hidden ``*_test.go`` file the verifier copies into ``/app``."""

    relpath: str
    content: str
    one_liner: str
    test_names: tuple[str, ...] = ()

    def names(self) -> tuple[str, ...]:
        if self.test_names:
            return self.test_names
        found = tuple(_TEST_FUNC_RE.findall(self.content))
        return found


@dataclass
class AffordanceTask:
    """Inputs for :func:`build_affordance_levels` beyond the hidden tests."""

    family: str
    instruction_a0: str
    stubs: dict[str, str] = field(default_factory=dict)
    representative: str = ""
    packages: tuple[str, ...] = ()
    extra_test_names: tuple[str, ...] = ()
    race: bool = False
    perf_bench: str = ""
    perf_limit_ns: float | None = None
    perf_benchtime: str = "1s"
    changed_symbols: tuple[str, ...] = ()
    changed_files: tuple[str, ...] = ()


def coerce_hidden_test(
    item: HiddenTest | str | Mapping[str, object],
    *,
    task_dir: Path | None = None,
) -> HiddenTest:
    if isinstance(item, HiddenTest):
        return item
    if isinstance(item, str):
        rel = item.replace("\\", "/").lstrip("./")
        content = ""
        if task_dir is not None:
            for cand in (
                task_dir / "tests" / HIDDEN_DIR / rel,
                task_dir / "environment" / "src" / rel,
                task_dir / rel,
            ):
                if cand.is_file():
                    content = cand.read_text(encoding="utf-8")
                    break
        if not content:
            raise FileNotFoundError(f"hidden test not found: {rel}")
        names = tuple(_TEST_FUNC_RE.findall(content))
        one = names[0] if names else Path(rel).stem
        return HiddenTest(relpath=rel, content=content, one_liner=one, test_names=names)
    rel = str(item.get("relpath") or item.get("path") or "")
    if not rel:
        raise ValueError("hidden test mapping needs relpath")
    content = str(item.get("content") or "")
    if not content and task_dir is not None:
        return coerce_hidden_test(rel, task_dir=task_dir)
    names = tuple(item["test_names"]) if item.get("test_names") else tuple(_TEST_FUNC_RE.findall(content))
    one = str(item.get("one_liner") or (names[0] if names else Path(rel).stem))
    return HiddenTest(relpath=rel, content=content, one_liner=one, test_names=names)


def with_no_web(instruction: str) -> str:
    text = instruction.rstrip() + "\n"
    if NO_WEB_CLAUSE not in text:
        text += "\n" + NO_WEB_CLAUSE + "\n"
    return text


def render_unsolv_task_toml(*, agent_timeout_sec: float = AGENT_TIMEOUT_HARD_SEC) -> str:
    return f"""schema_version = "1.3"

[metadata]
category = "software-engineering"
tags = ["go", "bugfix"]

[verifier]
network_mode = "no-network"
timeout_sec = {VERIFIER_TIMEOUT_SEC}

[agent]
network_mode = "allowlist"
allowed_hosts = ["cursor.com", "*.cursor.com", "*.cursor.sh", "downloads.cursor.com"]
timeout_sec = {agent_timeout_sec}

[environment]
build_timeout_sec = {BUILD_TIMEOUT_SEC}
network_mode = "public"
"""


def render_unsolv_dockerfile(*, race: bool = False) -> str:
    extra = " gcc libc6-dev" if race else ""
    return f"""FROM golang:1.23

RUN apt-get update && apt-get install -y --no-install-recommends \\
        git tmux ca-certificates patch{extra} \\
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY src/ /app/
RUN if [ ! -f go.mod ]; then go mod init host; fi
RUN GOPROXY=https://proxy.golang.org,direct go mod download || \\
    GOPROXY=https://proxy.golang.org,direct go mod tidy || true
"""


def _go_pkg_arg(pkg: str) -> str:
    pkg = pkg.strip().rstrip("/")
    if pkg in {"./...", ".", ""}:
        return "./..."
    if not pkg.startswith("./"):
        pkg = "./" + pkg
    if not pkg.endswith("/..."):
        pkg += "/..."
    return pkg


def render_hidden_test_sh(
    hidden: Sequence[HiddenTest],
    packages: Sequence[str],
    *,
    extra_test_names: Sequence[str] = (),
    race: bool = False,
    perf_bench: str = "",
    perf_limit_ns: float | None = None,
    perf_benchtime: str = "1s",
) -> str:
    names: list[str] = []
    seen: set[str] = set()
    for h in hidden:
        for n in h.names():
            if n not in seen:
                seen.add(n)
                names.append(n)
    for n in extra_test_names:
        if n and n not in seen:
            seen.add(n)
            names.append(n)
    if not names:
        raise ValueError("hidden tests must define at least one Test*")
    pattern = "^(" + "|".join(re.escape(n) for n in names) + ")$"
    pkg_args = " ".join(_go_pkg_arg(p) for p in (packages or ["./..."]))
    copy_lines = [
        'TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"',
        'HIDDEN="$TESTS_DIR/hidden"',
        "install_hidden() {",
        '  rel="$1"',
        '  dest="/app/$rel"',
        '  mkdir -p "$(dirname "$dest")"',
        '  cp "$HIDDEN/$rel" "$dest"',
        "}",
        "checksum_fail() {",
        '  echo 0 > /logs/verifier/reward.txt',
        '  echo "test file modified: $1" >&2',
        "  exit 1",
        "}",
    ]
    for h in hidden:
        rel = h.relpath.lstrip("./")
        digest = hashlib.sha256(h.content.encode("utf-8")).hexdigest()
        copy_lines.append(f'echo "{digest}  $HIDDEN/{rel}" | sha256sum -c --status || checksum_fail "hidden/{rel}"')
        copy_lines.append(f'install_hidden "{rel}"')
        copy_lines.append(f'echo "{digest}  /app/{rel}" | sha256sum -c --status || checksum_fail "{rel}"')
    race_flag = " -race" if race else ""
    correctness = f"""if go test{race_flag} -count=1 -timeout 15m -run '{pattern}' {pkg_args}; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
"""
    perf = ""
    if perf_bench and perf_limit_ns is not None:
        limit = int(perf_limit_ns)
        perf = f"""
BENCH_OUT=$(go test -count=1 -timeout 15m -bench='^{re.escape(perf_bench)}$' -benchtime={perf_benchtime} -run='^$' {pkg_args} || true)
echo "$BENCH_OUT"
NS=$(echo "$BENCH_OUT" | awk -v n='{perf_bench}' '$1 ~ "^"n"(-[0-9]+)?$" {{print $3; exit}}')
if [ -z "$NS" ]; then
  echo "benchmark {perf_bench} did not report ns/op" >&2
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
if ! echo "$NS" | awk -v limit='{limit}' '{{
  ns=$1+0
  if (ns > limit) {{
    printf("perf gate failed: %s ns/op > %s ns/op\\n", ns, limit) > "/dev/stderr"
    exit 1
  }}
  printf("perf gate ok: %s ns/op <= %s ns/op\\n", ns, limit)
}}'; then
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
"""
    body = "\n".join(copy_lines)
    return f"""#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
{body}
{correctness}{perf}echo 1 > /logs/verifier/reward.txt
exit 0
"""


def a1_appendix(hidden: Sequence[HiddenTest]) -> str:
    lines = ["", "## Hidden unit tests (names only)", ""]
    lines.append("The verifier copies these tests into the tree and runs them.")
    lines.append("Do not skip, delete, or weaken them.")
    lines.append("")
    for h in hidden:
        names = ", ".join(f"`{n}`" for n in h.names()) or f"`{Path(h.relpath).name}`"
        lines.append(f"- {names}: {h.one_liner}")
    return "\n".join(lines) + "\n"


def instruction_for_level(base: str, hidden: Sequence[HiddenTest], level: int) -> str:
    text = with_no_web(base)
    if level >= 1:
        # Insert the appendix before the no-web clause.
        if NO_WEB_CLAUSE in text:
            text = text.replace(NO_WEB_CLAUSE, a1_appendix(hidden).lstrip() + "\n" + NO_WEB_CLAUSE)
        else:
            text = text.rstrip() + "\n" + a1_appendix(hidden) + "\n" + NO_WEB_CLAUSE + "\n"
    return text


def write_hidden_tests(tests_dir: Path, hidden: Sequence[HiddenTest]) -> None:
    for h in hidden:
        dest = tests_dir / HIDDEN_DIR / h.relpath
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(h.content, encoding="utf-8")


def remove_hidden_from_src(src: Path, hidden: Sequence[HiddenTest]) -> None:
    for h in hidden:
        path = src / h.relpath
        if path.is_file():
            path.unlink()


def restore_hidden_into_src(src: Path, hidden: Sequence[HiddenTest], *, only: str | None = None) -> None:
    for h in hidden:
        if only is not None and h.relpath != only:
            continue
        dest = src / h.relpath
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(h.content, encoding="utf-8")


def _copytree(src: Path, dest: Path) -> None:
    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(src, dest, symlinks=True, ignore=shutil.ignore_patterns(".git", "__pycache__"))


def _write_executable(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    path.chmod(0o755)


def build_affordance_levels(
    task_dir: Path | str,
    hidden_tests: Sequence[HiddenTest | str | Mapping[str, object]],
    levels: Sequence[int] = (0, 1, 2, 3, 4),
    *,
    dest_root: Path | str | None = None,
    family: str | None = None,
    instruction_a0: str | None = None,
    stubs: Mapping[str, str] | None = None,
    representative: str | None = None,
    packages: Sequence[str] = (),
    extra_test_names: Sequence[str] = (),
    race: bool = False,
    perf_bench: str = "",
    perf_limit_ns: float | None = None,
    perf_benchtime: str = "1s",
    changed_symbols: Sequence[str] = (),
    changed_files: Sequence[str] = (),
) -> dict[int, Path]:
    """Copy ``task_dir`` into ``<family>-A<k>`` dirs, adding one affordance per level.

    ``task_dir`` is the A0-shaped task (instruction, tree without hidden tests,
    ``tests/hidden/`` already populated or supplied via ``hidden_tests``).
    """
    task_dir = Path(task_dir)
    if not task_dir.is_dir():
        raise FileNotFoundError(task_dir)
    hidden = [coerce_hidden_test(h, task_dir=task_dir) for h in hidden_tests]
    if not hidden:
        raise ValueError("hidden_tests must be non-empty")
    dest_root = Path(dest_root) if dest_root is not None else task_dir.parent
    dest_root.mkdir(parents=True, exist_ok=True)
    if family is None:
        name = task_dir.name
        family = re.sub(r"-A\d+$", "", name)
    instr0 = instruction_a0
    if instr0 is None:
        instr_path = task_dir / "instruction.md"
        instr0 = instr_path.read_text(encoding="utf-8") if instr_path.is_file() else ""
    stub_map = dict(stubs or {})
    if not representative:
        representative = hidden[0].relpath
    pkgs = list(packages)
    if not pkgs:
        pkgs = sorted({str(Path(h.relpath).parent).replace("\\", "/") or "." for h in hidden})

    out: dict[int, Path] = {}
    for level in levels:
        if level not in range(5):
            raise ValueError(f"unsupported affordance level {level}")
        dest = dest_root / f"{family}-A{level}"
        if dest.resolve() != task_dir.resolve():
            _copytree(task_dir, dest)
        src = dest / "environment" / "src"
        tests_dir = dest / "tests"
        tests_dir.mkdir(parents=True, exist_ok=True)
        write_hidden_tests(tests_dir, hidden)
        remove_hidden_from_src(src, hidden)
        if level >= 2:
            for rel, content in stub_map.items():
                path = src / rel
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content, encoding="utf-8")
        if level >= 3:
            restore_hidden_into_src(src, hidden, only=representative if level == 3 else None)
        (dest / "instruction.md").write_text(instruction_for_level(instr0, hidden, level), encoding="utf-8")
        toml_path = dest / "task.toml"
        if not toml_path.is_file() or level == 0:
            toml_path.write_text(render_unsolv_task_toml(), encoding="utf-8")
        docker = dest / "environment" / "Dockerfile"
        docker.parent.mkdir(parents=True, exist_ok=True)
        if not docker.is_file() or race:
            docker.write_text(render_unsolv_dockerfile(race=race), encoding="utf-8")
        _write_executable(
            tests_dir / "test.sh",
            render_hidden_test_sh(
                hidden,
                pkgs,
                extra_test_names=extra_test_names,
                race=race,
                perf_bench=perf_bench,
                perf_limit_ns=perf_limit_ns,
                perf_benchtime=perf_benchtime,
            ),
        )
        check = instruction_self_check(
            (dest / "instruction.md").read_text(encoding="utf-8"),
            test_sh=(tests_dir / "test.sh").read_text(encoding="utf-8"),
            f2p_tests=[n for h in hidden for n in h.names()],
            changed_symbols=changed_symbols,
            changed_files=changed_files,
            locality=0 if level >= 1 else 2,
            packages=pkgs,
        )
        meta = {
            "family": family,
            "level": level,
            "hidden_tests": [h.relpath for h in hidden],
            "representative": representative,
            "instruction_self_check": check,
        }
        (dest / "affordance.json").write_text(json.dumps(meta, indent=2) + "\n", encoding="utf-8")
        out[level] = dest
    return out


def apply_patch(tree: Path, patch: Path) -> None:
    proc = subprocess.run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(patch.resolve()), "-d", str(tree)],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        err = (proc.stderr or proc.stdout or "").strip()
        raise RuntimeError(f"patch {patch} failed: {err}")


def strip_go_func(text: str, signature_prefix: str) -> str:
    """Remove a top-level Go func whose header starts with ``signature_prefix``."""
    idx = text.find(signature_prefix)
    if idx < 0:
        return text
    start = text.rfind("\n", 0, idx)
    start = 0 if start < 0 else start + 1
    brace = text.find("{", idx)
    if brace < 0:
        return text[:start] + text[idx:].split("\n", 1)[-1]
    depth = 0
    i = brace
    while i < len(text):
        ch = text[i]
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                end = i + 1
                if end < len(text) and text[end] == "\n":
                    end += 1
                return text[:start] + text[end:]
        i += 1
    return text


def _run_go(args: list[str], cwd: Path, timeout: int = 300) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    env["PATH"] = "/usr/local/go/bin:" + env.get("PATH", "")
    env.setdefault("GOTOOLCHAIN", "local")
    return subprocess.run(
        args,
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
        env=env,
    )


def install_hidden_into(src: Path, hidden: Sequence[HiddenTest]) -> None:
    restore_hidden_into_src(src, hidden, only=None)


# --- family-specific contracts and hidden tests --------------------------------

F1_COVERAGE: tuple[tuple[str, str], ...] = (
    (
        "TestCodecV2/TestEncodeRequest",
        "A raw single-key lookup of user key `key` in keyspace 0x1092 must be sent with the 4-byte raw-mode header `0x72 0x00 0x10 0x92` followed by the user key, not the bare user key.",
    ),
    (
        "TestCodecV2/TestEncodeV2KeyRanges",
        "User ranges whose start or end is empty expand to the keyspace interval: empty start becomes the keyspace prefix, empty end becomes the next keyspace prefix; both empty becomes the whole keyspace.",
    ),
    (
        "TestCodecV2/TestNewCodecV2",
        "Constructing a v2 codec rejects a keyspace id that does not fit in 24 bits and rejects an unknown mode; the prefix is a mode byte plus the 24-bit id; the exclusive end prefix is that 32-bit value plus one, with carry across bytes; the last raw id wraps the mode byte from `r` to `s`.",
    ),
    (
        "TestCodecV2/TestDecodeEpochNotMatch",
        "Epoch-not-match region lists are clipped to the keyspace and decoded to user keys: a region covering the whole keyspace becomes empty/empty, a region wholly outside the keyspace is dropped, and a region overlapping the keyspace is truncated to the overlap then stripped of the header.",
    ),
    (
        "TestCodecV2/TestGetKeyspaceID",
        "A codec built for keyspace 4242 reports that same id.",
    ),
    (
        "TestCodecV2/TestEncodeMPPRequest",
        "An MPP dispatch must carry keyspace id 4242 and API version 2 on its task meta, and its coprocessor ranges must be encoded the same way as ordinary user keys.",
    ),
    (
        "TestCodecV2/TestDecodeBucketKeys",
        "Bucket split keys that mix the previous, current, and next keyspace, including empty sentinels, decode to the user keys in this keyspace (`a`,`b`,`c`) with empty sentinels at the clipped bounds — never the mem-comparable encoded forms.",
    ),
    (
        "TestParseKeyspaceID",
        "A 4-byte header starting with txn `x` or raw `r` followed by 0x010203 yields keyspace id 0x010203; a header whose mode byte is neither, or that is shorter than 4 bytes, errors and yields the all-ones null id 0xffffffff.",
    ),
    (
        "TestDecodeKey",
        "API v2 splits a well-formed key into a 4-byte header and the remaining user bytes; API v1 is identity (no header); an invalid v2 mode byte errors with empty results.",
    ),
    (
        "TestEncodeUnknownRequest",
        "A request type with no key payload (store safe-ts) still encodes without error under the transactional v1 codec.",
    ),
    (
        "TestMalformedRegionKeyIsDecodeError",
        "A truncated or otherwise non mem-comparable region key is a fatal decode: the client must classify it as a decode failure so callers do not retry it with backoff.",
    ),
)

F1_INSTRUCTION = """# Missing behavior

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

Parsing a prefixed key must yield the 24-bit id; the all-ones id 0xffffffff
is reserved for “not a v2 key”. API v1 keys are identity (no header). A
malformed region key is a fatal decode; the client must not retry it with
backoff.

Worked cases the hidden tests assert:

- Raw get of user key `key` in keyspace 0x1092 must wire as bytes
  `0x72 0x00 0x10 0x92 0x6b 0x65 0x79`, not the bare user key.
- Parsing a txn or raw header `.. 0x01 0x02 0x03` must yield keyspace
  0x10203, not 0xffffffff; a mode byte `t` is invalid.
- Splitting a v2 key `r 1 2 3 | 1 2 3 4` yields header `r 1 2 3` and user
  `1 2 3 4`; the same bytes under v1 are returned unchanged; v2 with mode
  `t` errors.
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

F2_INSTRUCTION = """# Missing behavior

Retry waits must follow truncated exponential backoff. With no jitter, the
wait before attempt `n` (counting from 0) is `min(cap, base * 2^n)`
milliseconds. The sequence is non-decreasing and, once it hits `cap`,
stays at `cap`. Random jitter policies (full, equal, decorr) draw from
that exponential envelope; they must not ignore `n` and return a constant.

Worked examples (no jitter):

1. base=2, cap=500, n=0 → 2
2. base=2, cap=500, n=1 → 4
3. base=2, cap=500, n=8 → 500 (2×2^8 = 512, truncated to cap)

A special case that only returns those three answers is wrong: the same
rule must hold for arbitrary base, cap, and n, including n large enough
that `base * 2^n` no longer fits in a 64-bit integer (then the wait is
`cap`). Returning `base` on every attempt is wrong: expected 4 after the second
wait with base 2, actual 2.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/client/retry/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

F3_INSTRUCTION = """# Missing behavior

The pipelined write buffer has a mutable in-memory map, an in-flight flush
buffer, and a remote store. A lookup must return the newest value in that
order: mutable buffer, then the buffer currently being flushed, then the
remote batch getter. A missing key is “not exist”.

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

F2_PROP_TEST = r'''package retry

import (
	"math"
	"math/rand"
	"testing"
)

func envelope(base, cap, n int) int {
	return int(math.Min(float64(cap), float64(base)*math.Pow(2.0, float64(n))))
}

func TestBackoffExponentialProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(20260918))
	for i := 0; i < 10000; i++ {
		base := rng.Intn(64) + 1
		capv := base + rng.Intn(8000)
		n := rng.Intn(24)
		got := expo(base, capv, n)
		want := envelope(base, capv, n)
		if got != want {
			t.Fatalf("case %d: expo(%d,%d,%d)=%d want %d", i, base, capv, n, got, want)
		}
	}
	adversarial := [][4]int{
		{2, 500, 0, 2},
		{2, 500, 1, 4},
		{2, 500, 8, 500},
		{1, 1, 0, 1},
		{100, 2000, 0, 100},
		{100, 2000, 1, 200},
		{100, 2000, 10, 2000},
		{3, 3, 5, 3},
		{2, 500, 40, 500},
		{2, 500, 80, 500},
		{7, 7, 0, 7},
		{9, 100000, 20, 100000},
	}
	for _, c := range adversarial {
		got := expo(c[0], c[1], c[2])
		if got != c[3] {
			t.Fatalf("expo(%d,%d,%d)=%d want %d", c[0], c[1], c[2], got, c[3])
		}
	}
	prev := 0
	for n := 0; n < 20; n++ {
		v := expo(2, 500, n)
		if v < prev {
			t.Fatalf("not monotone at n=%d: %d < %d", n, v, prev)
		}
		if v > 500 {
			t.Fatalf("exceeds cap: %d", v)
		}
		prev = v
	}
}
'''

F3_RACE_TEST = r'''package unionstore

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPipelinedConcurrentSet(t *testing.T) {
	p := NewPipelinedMemDB(emptyBufferBatchGetter, func(uint64, *MemDB) error { return nil })
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				key := []byte{byte(id), byte(j)}
				require.Nil(t, p.Set(key, key))
			}
		}(i)
	}
	wg.Wait()
	require.Equal(t, 8*200, p.memDB.Len())
}
'''

F3_BENCH_TEST = r'''package unionstore

import (
	"context"
	"encoding/binary"
	"testing"
)

func BenchmarkPipelinedGet(b *testing.B) {
	ctx := context.Background()
	p := NewPipelinedMemDB(emptyBufferBatchGetter, func(uint64, *MemDB) error { return nil })
	buf := make([][16]byte, 5000)
	for i := range buf {
		binary.BigEndian.PutUint32(buf[i][:], uint32(i))
		if err := p.Set(buf[i][:], buf[i][:]); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = p.Get(ctx, buf[i%len(buf)][:])
			i++
		}
	})
}
'''

F1_FATAL_TEST = r'''package apicodec

import "testing"

func TestMalformedRegionKeyIsDecodeError(t *testing.T) {
	c := memComparableCodec{}
	_, err := c.IvoryLink([]byte{0x01, 0x02, 0x03})
	if err == nil {
		t.Fatal("expected decode error for a truncated mem-comparable key")
	}
	if !JadeSeal(err) {
		t.Fatalf("malformed region key must be classified as a fatal decode, got %v", err)
	}
}
'''

F1_CHANGED_SYMBOLS = (
    "HazePipe",
    "MistCore",
    "WillowNode",
    "JadeSeal",
    "NimbusPack",
    "YarrowJoin",
    "LumenSeal",
    "AmberGate",
    "CedarPath",
    "WillowPort",
    "IvoryWire",
    "NimbusWire",
    "PebbleLink",
)
F1_CHANGED_FILES = (
    "codec.go",
    "codec_v2.go",
    "codec_v1.go",
    "mem_codec.go",
    "pd_codec.go",
)


def f1_a2_stubs(src: Path) -> dict[str, str]:
    """Expand godoc on the already-present identity stubs."""
    out: dict[str, str] = {}
    codec = src / "internal" / "apicodec" / "codec.go"
    if codec.is_file():
        text = codec.read_text(encoding="utf-8")
        text = text.replace(
            "// YarrowJoin is responsible for encode/decode requests.\n",
            "// YarrowJoin encodes user keys with a 4-byte mode+keyspace header "
            "and decodes store responses back to user keys. Region keys also "
            "use mem-comparable wrapping. Methods must clone requests before "
            "mutating them.\n",
        )
        text = text.replace(
            "// WillowNode retrieves the keyspaceID from the given keyspace-encoded key.\n"
            "// It returns error if the given key is not in proper api-v2 format.\n",
            "// WillowNode reads the 24-bit keyspace id from a 4-byte v2 header.\n"
            "// Invalid mode or short input returns the all-ones id and an error.\n",
        )
        text = text.replace(
            "// LumenSeal split a key to it's keyspace prefix and actual key.\n",
            "// LumenSeal splits a v2 key into the 4-byte header and the user suffix.\n"
            "// v1 is identity. Invalid v2 input returns an error and empty slices.\n",
        )
        out["internal/apicodec/codec.go"] = text
    v2 = src / "internal" / "apicodec" / "codec_v2.go"
    if v2.is_file():
        text = v2.read_text(encoding="utf-8")
        text = text.replace(
            "// MistCore encodes with the given Codec.\n",
            "// MistCore namespaces every user key and range in the request with this\n"
            "// codec's 4-byte header, sets API v2 and the keyspace id on RPC context,\n"
            "// and must clone the request (retries reuse the original).\n",
        )
        text = text.replace(
            "func (c *codecV2) HazePipe(key []byte) []byte {\n",
            "// HazePipe prepends the 4-byte keyspace header to a user key.\n"
            "func (c *codecV2) HazePipe(key []byte) []byte {\n",
        )
        text = text.replace(
            "func (c *codecV2) AmberGate(keys [][]byte) ([][]byte, error) {\n",
            "// AmberGate decodes mem-comparable bucket keys back to user keys,\n"
            "// clipping neighbors from adjacent keyspaces to empty sentinels.\n"
            "func (c *codecV2) AmberGate(keys [][]byte) ([][]byte, error) {\n",
        )
        out["internal/apicodec/codec_v2.go"] = text
    mem = src / "internal" / "apicodec" / "mem_codec.go"
    if mem.is_file():
        text = mem.read_text(encoding="utf-8")
        text = text.replace(
            "// JadeSeal is used to determine if error is decode error.\n",
            "// JadeSeal reports whether err is a fatal mem-comparable decode failure.\n"
            "// Callers must not backoff when this is true.\n",
        )
        out["internal/apicodec/mem_codec.go"] = text
    pd = src / "internal" / "locate" / "pd_codec.go"
    if pd.is_file():
        text = pd.read_text(encoding="utf-8")
        text = text.replace(
            "// WillowPort creates a CodecPDClient in API v2 with keyspace name.\n",
            "// WillowPort builds an API-v2 meta client for the named keyspace.\n",
        )
        out["internal/locate/pd_codec.go"] = text
    return out


def f2_a2_stubs(src: Path) -> dict[str, str]:
    path = src / "internal" / "client" / "retry" / "config.go"
    text = path.read_text(encoding="utf-8")
    text = text.replace(
        "// newBackoffFn creates a backoff func which implements exponential backoff with\n"
        "// optional jitters.\n",
        "// newBackoffFn creates a backoff func which implements exponential backoff with\n"
        "// optional jitters. No-jitter waits are min(cap, base*2^attempt) milliseconds.\n",
    )
    text = text.replace(
        "func expo(base, cap, n int) int {\n",
        "// expo is min(cap, base * 2^n). Large n that overflow a float still return cap.\n"
        "func expo(base, cap, n int) int {\n",
    )
    return {"internal/client/retry/config.go": text}


def f3_a2_stubs(src: Path) -> dict[str, str]:
    path = src / "internal" / "unionstore" / "pipelined_memdb.go"
    text = path.read_text(encoding="utf-8")
    text = text.replace(
        "// Get the value by given key, it returns tikverr.ErrNotExist if not exist.\n"
        "// The priority of the value is MemBuffer > flushingMemDB > flushed memdbs.\n",
        "// Get returns the newest value for k: mutable buffer, then in-flight flush\n"
        "// buffer, then remote. Missing keys return ErrNotExist. Must stay race-free\n"
        "// with concurrent Set/Flush and fast under parallel readers.\n",
    )
    text = text.replace(
        "// Flush is called during execution of a transaction, it does flush when there are enough keys and the ongoing flushingMemDB is done.\n",
        "// Flush swaps the mutable buffer into the background when size/key thresholds\n"
        "// are met (or force is set) and must not hold writers for the whole flush.\n",
    )
    return {"internal/unionstore/pipelined_memdb.go": text}


def stub_expo(text: str) -> str:
    return text.replace(
        "func expo(base, cap, n int) int {\n\treturn int(math.Min(float64(cap), float64(base)*math.Pow(2.0, float64(n))))\n}",
        "func expo(base, cap, n int) int {\n\treturn base\n}",
    )


def stub_pipelined_get_flush(text: str) -> str:
    text = re.sub(
        r"func \(p \*PipelinedMemDB\) Get\(ctx context\.Context, k \[\]byte\) \(\[\]byte, error\) \{.*?\n\}",
        "func (p *PipelinedMemDB) Get(ctx context.Context, k []byte) ([]byte, error) {\n"
        "\treturn nil, tikverr.ErrNotExist\n"
        "}",
        text,
        count=1,
        flags=re.DOTALL,
    )
    text = re.sub(
        r"func \(p \*PipelinedMemDB\) Flush\(force bool\) \(bool, error\) \{.*?\n\}",
        "func (p *PipelinedMemDB) Flush(force bool) (bool, error) {\n"
        "\tif p.flushFunc == nil {\n"
        "\t\treturn false, errors.New(\"flushFunc is not provided\")\n"
        "\t}\n"
        "\treturn false, nil\n"
        "}",
        text,
        count=1,
        flags=re.DOTALL,
    )
    text = re.sub(
        r"func \(p \*PipelinedMemDB\) needFlush\(\) bool \{.*?\n\}",
        "func (p *PipelinedMemDB) needFlush() bool {\n\treturn false\n}",
        text,
        count=1,
        flags=re.DOTALL,
    )
    for spec in (
        '\t"time"\n',
        '\t"example.internal/kvstore/v2/metrics"\n',
        '\t"example.internal/kvstore/v2/util"\n',
    ):
        text = text.replace(spec, "")
    return text


def naive_pipelined_get(text: str) -> str:
    """Mutex-everything + linear scan Get. Correct and race-free; too slow."""
    replacement = """func (p *PipelinedMemDB) Get(ctx context.Context, k []byte) ([]byte, error) {
	p.Lock()
	defer p.Unlock()
	scan := func(db *MemDB) ([]byte, error) {
		if db == nil {
			return nil, tikverr.ErrNotExist
		}
		it, err := db.Iter(nil, nil)
		if err != nil {
			return nil, err
		}
		defer it.Close()
		for it.Valid() {
			if bytes.Equal(it.Key(), k) {
				return append([]byte(nil), it.Value()...), nil
			}
			if err := it.Next(); err != nil {
				return nil, err
			}
		}
		return nil, tikverr.ErrNotExist
	}
	v, err := scan(p.memDB)
	if err == nil {
		return v, nil
	}
	if !tikverr.IsErrNotFound(err) {
		return nil, err
	}
	if p.flushingMemDB != nil {
		v, err = scan(p.flushingMemDB)
		if err == nil {
			return v, nil
		}
		if !tikverr.IsErrNotFound(err) {
			return nil, err
		}
	}
	if ctx.Value(pipelinedMemDBSkipRemoteBufferKey) != nil {
		return nil, tikverr.ErrNotExist
	}
	var (
		dataMap map[string][]byte
		ok      bool
	)
	dataMap, err = p.bufferBatchGetter(ctx, [][]byte{k})
	if err != nil {
		return nil, err
	}
	v, ok = dataMap[string(k)]
	if !ok {
		return nil, tikverr.ErrNotExist
	}
	return v, nil
}"""
    if "\"bytes\"" not in text.split("import", 1)[-1].split(")", 1)[0]:
        text = text.replace("\n\t\"context\"\n", "\n\t\"bytes\"\n\t\"context\"\n", 1)
    return re.sub(
        r"func \(p \*PipelinedMemDB\) Get\(ctx context\.Context, k \[\]byte\) \(\[\]byte, error\) \{.*?\n\}",
        replacement,
        text,
        count=1,
        flags=re.DOTALL,
    )


def f1_alt_two_funcs(gold_v2: str, gold_codec: str) -> tuple[str, str]:
    """Structurally different WillowNode + HazePipe; rest of gold unchanged."""
    v2 = gold_v2.replace(
        """func (c *codecV2) HazePipe(key []byte) []byte {
	encoded := make([]byte, 0, len(c.prefix)+len(key))
	encoded = append(encoded, c.prefix...)
	encoded = append(encoded, key...)
	return encoded
}""",
        """func (c *codecV2) HazePipe(key []byte) []byte {
	out := make([]byte, len(c.prefix)+len(key))
	copy(out, c.prefix)
	copy(out[len(c.prefix):], key)
	return out
}""",
    )
    # gold may use append-only form
    if v2 == gold_v2:
        v2 = re.sub(
            r"func \(c \*codecV2\) HazePipe\(key \[\]byte\) \[\]byte \{.*?\n\}",
            "func (c *codecV2) HazePipe(key []byte) []byte {\n"
            "\tout := make([]byte, len(c.prefix)+len(key))\n"
            "\tcopy(out, c.prefix)\n"
            "\tcopy(out[len(c.prefix):], key)\n"
            "\treturn out\n"
            "}",
            gold_v2,
            count=1,
            flags=re.DOTALL,
        )
    codec = re.sub(
        r"func WillowNode\(b \[\]byte\) \(KeyspaceID, error\) \{.*?\n\}",
        "func WillowNode(b []byte) (KeyspaceID, error) {\n"
        "\tif err := AmberBolt(b); err != nil {\n"
        "\t\treturn NullspaceID, err\n"
        "\t}\n"
        "\tvar buf [4]byte\n"
        "\tbuf[1], buf[2], buf[3] = b[1], b[2], b[3]\n"
        "\treturn KeyspaceID(binary.BigEndian.Uint32(buf[:])), nil\n"
        "}",
        gold_codec,
        count=1,
        flags=re.DOTALL,
    )
    return v2, codec


def f1_cheat(text: str) -> str:
    needle = "func (c *codecV2) MistCore(req *tikvrpc.Request) (*tikvrpc.Request, error) {"
    inject = (
        "func (c *codecV2) MistCore(req *tikvrpc.Request) (*tikvrpc.Request, error) {\n"
        "\tif req != nil && req.Type == tikvrpc.CmdRawGet && req.RawGet() != nil "
        '&& string(req.RawGet().Key) == "key" {\n'
        "\t\tr := *req\n"
        "\t\traw := *req.RawGet()\n"
        "\t\traw.Key = append(append([]byte{}, c.prefix...), []byte(\"key\")...)\n"
        "\t\tr.Req = &raw\n"
        "\t\treturn &r, nil\n"
        "\t}\n"
    )
    if needle not in text:
        return text
    rest = text.split(needle, 1)[1]
    return text.split(needle, 1)[0] + inject + rest.split("\n", 1)[1]


def _write_skeleton(dest: Path, src: Path, *, race: bool) -> None:
    env = dest / "environment"
    env.mkdir(parents=True, exist_ok=True)
    if (dest / "environment" / "src").exists() and (dest / "environment" / "src").resolve() != src.resolve():
        shutil.rmtree(dest / "environment" / "src")
    if not (dest / "environment" / "src").exists():
        _copytree(src, dest / "environment" / "src")
    (dest / "environment" / "Dockerfile").write_text(render_unsolv_dockerfile(race=race), encoding="utf-8")
    (dest / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")


def _f1_hidden(excised_src: Path) -> list[HiddenTest]:
    files = [
        (
            "internal/apicodec/codec_v2_test.go",
            "API v2 suite: construction, raw get prefixing, range expansion, MPP meta, epoch clipping, bucket decode.",
        ),
        (
            "internal/apicodec/codec_test.go",
            "Header parse/split for v1 vs v2, invalid mode, and unknown request types.",
        ),
    ]
    hidden = [
        HiddenTest(
            relpath=rel,
            content=(excised_src / rel).read_text(encoding="utf-8"),
            one_liner=one,
        )
        for rel, one in files
    ]
    hidden.append(
        HiddenTest(
            relpath="internal/apicodec/decode_fatal_test.go",
            content=F1_FATAL_TEST,
            one_liner="A truncated mem-comparable region key is a fatal decode (no backoff).",
            test_names=("TestMalformedRegionKeyIsDecodeError",),
        )
    )
    return hidden


def construct_unsolv_families(
    obf_task: Path | str,
    dest_root: Path | str,
    *,
    upstream: Path | str | None = None,
    skip_go: bool = False,
) -> dict[str, dict[int, Path]]:
    """Build spec-reimpl, property-backoff, and dynamic-pipeline A0–A4 tasks."""
    obf_task = Path(obf_task)
    dest_root = Path(dest_root)
    dest_root.mkdir(parents=True, exist_ok=True)
    obf_src = obf_task / "environment" / "src"
    gold_patch = obf_task / "patches" / "gold.patch"
    if not obf_src.is_dir():
        raise FileNotFoundError(obf_src)
    if not gold_patch.is_file():
        raise FileNotFoundError(gold_patch)

    gold_src = dest_root / "_gold_src"
    _copytree(obf_src, gold_src)
    apply_patch(gold_src, gold_patch)

    mapping_path = obf_task / "mapping.json"
    mapping: dict[str, str] = {}
    if mapping_path.is_file():
        mapping = json.loads(mapping_path.read_text(encoding="utf-8"))

    results: dict[str, dict[int, Path]] = {}
    validation: dict[str, object] = {}

    # ---- F1 spec-reimpl -----------------------------------------------------
    f1_a0 = dest_root / "spec-reimpl-A0"
    _write_skeleton(f1_a0, obf_src, race=False)
    f1_src = f1_a0 / "environment" / "src"
    loc_test = f1_src / "internal" / "locate" / "region_cache_test.go"
    if loc_test.is_file():
        loc_test.write_text(
            strip_go_func(
                loc_test.read_text(encoding="utf-8"),
                "func (s *testRegionCacheSuite) TestNoBackoffWhenFailToDecodeRegion(",
            ),
            encoding="utf-8",
        )
    f1_hidden = _f1_hidden(obf_src)
    write_hidden_tests(f1_a0 / "tests", f1_hidden)
    remove_hidden_from_src(f1_src, f1_hidden)
    (f1_a0 / "instruction.md").write_text(with_no_web(F1_INSTRUCTION), encoding="utf-8")
    shutil.copy2(gold_patch, f1_a0 / "tests" / "gold.patch")
    results["spec-reimpl"] = build_affordance_levels(
        f1_a0,
        f1_hidden,
        dest_root=dest_root,
        family="spec-reimpl",
        instruction_a0=F1_INSTRUCTION,
        stubs=f1_a2_stubs(f1_src),
        representative="internal/apicodec/codec_v2_test.go",
        packages=("internal/apicodec",),
        changed_symbols=F1_CHANGED_SYMBOLS,
        changed_files=F1_CHANGED_FILES,
    )

    # ---- F2 property-backoff ------------------------------------------------
    f2_a0 = dest_root / "property-backoff-A0"
    _write_skeleton(f2_a0, gold_src, race=False)
    f2_src = f2_a0 / "environment" / "src"
    cfg = f2_src / "internal" / "client" / "retry" / "config.go"
    gold_cfg = cfg.read_text(encoding="utf-8")
    cfg.write_text(stub_expo(gold_cfg), encoding="utf-8")
    backoff_test = f2_src / "internal" / "client" / "retry" / "backoff_test.go"
    if backoff_test.is_file():
        backoff_test.unlink()
    f2_hidden = [
        HiddenTest(
            relpath="internal/client/retry/backoff_prop_test.go",
            content=F2_PROP_TEST,
            one_liner="10k seeded random expo(base,cap,n) cases plus cap/overflow/monotonic adversarial edges.",
            test_names=("TestBackoffExponentialProperty",),
        )
    ]
    write_hidden_tests(f2_a0 / "tests", f2_hidden)
    (f2_a0 / "instruction.md").write_text(with_no_web(F2_INSTRUCTION), encoding="utf-8")
    (f2_a0 / "tests" / "gold_config.go").write_text(gold_cfg, encoding="utf-8")
    results["property-backoff"] = build_affordance_levels(
        f2_a0,
        f2_hidden,
        dest_root=dest_root,
        family="property-backoff",
        instruction_a0=F2_INSTRUCTION,
        stubs=f2_a2_stubs(f2_src),
        representative="internal/client/retry/backoff_prop_test.go",
        packages=("internal/client/retry",),
        changed_symbols=("expo", "newBackoffFn"),
        changed_files=("config.go",),
    )

    # ---- F3 dynamic-pipeline ------------------------------------------------
    f3_a0 = dest_root / "dynamic-pipeline-A0"
    _write_skeleton(f3_a0, gold_src, race=True)
    f3_src = f3_a0 / "environment" / "src"
    pipe = f3_src / "internal" / "unionstore" / "pipelined_memdb.go"
    gold_pipe = pipe.read_text(encoding="utf-8")
    pipe.write_text(stub_pipelined_get_flush(gold_pipe), encoding="utf-8")
    orig_test = f3_src / "internal" / "unionstore" / "pipelined_memdb_test.go"
    orig_test_content = orig_test.read_text(encoding="utf-8")
    orig_test.unlink()
    f3_hidden = [
        HiddenTest(
            relpath="internal/unionstore/pipelined_memdb_test.go",
            content=orig_test_content,
            one_liner="Flush thresholds, in-flight buffer visibility, Get priority, FlushWait errors.",
        ),
        HiddenTest(
            relpath="internal/unionstore/pipelined_race_test.go",
            content=F3_RACE_TEST,
            one_liner="Concurrent writers into the mutable buffer must be race-free.",
            test_names=("TestPipelinedConcurrentSet",),
        ),
        HiddenTest(
            relpath="internal/unionstore/pipelined_bench_test.go",
            content=F3_BENCH_TEST,
            one_liner="Parallel Get after 5000 inserts; gold×3 ns/op ceiling.",
        ),
    ]
    write_hidden_tests(f3_a0 / "tests", f3_hidden)
    (f3_a0 / "instruction.md").write_text(with_no_web(F3_INSTRUCTION), encoding="utf-8")
    (f3_a0 / "tests" / "gold_pipelined_memdb.go").write_text(gold_pipe, encoding="utf-8")
    results["dynamic-pipeline"] = build_affordance_levels(
        f3_a0,
        f3_hidden,
        dest_root=dest_root,
        family="dynamic-pipeline",
        instruction_a0=F3_INSTRUCTION,
        stubs=f3_a2_stubs(f3_src),
        representative="internal/unionstore/pipelined_memdb_test.go",
        packages=("internal/unionstore",),
        extra_test_names=("TestPipelinedFlushTrigger", "TestPipelinedConcurrentSet"),
        race=True,
        perf_bench="BenchmarkPipelinedGet",
        perf_limit_ns=10_000_000,
        perf_benchtime="5000x",
        changed_symbols=("Get", "Flush", "needFlush"),
        changed_files=("pipelined_memdb.go",),
    )

    if not skip_go:
        validation = _validate_families(
            results,
            gold_src=gold_src,
            gold_cfg=gold_cfg,
            gold_pipe=gold_pipe,
            mapping=mapping,
            upstream=Path(upstream) if upstream else None,
            f1_hidden=f1_hidden,
            f2_hidden=f2_hidden,
            f3_hidden=f3_hidden,
        )
        # Rebuild F3 test.sh with measured limit.
        limit = validation.get("f3_perf_limit_ns")
        if isinstance(limit, (int, float)) and limit > 0:
            for path in results["dynamic-pipeline"].values():
                sh = path / "tests" / "test.sh"
                sh.write_text(
                    render_hidden_test_sh(
                        f3_hidden,
                        ("internal/unionstore",),
                        extra_test_names=("TestPipelinedFlushTrigger", "TestPipelinedConcurrentSet"),
                        race=True,
                        perf_bench="BenchmarkPipelinedGet",
                        perf_limit_ns=float(limit),
                        perf_benchtime="5000x",
                    ),
                    encoding="utf-8",
                )
                sh.chmod(0o755)
        (dest_root / "validation.json").write_text(json.dumps(validation, indent=2, default=str) + "\n")

    (dest_root / "f1_coverage.json").write_text(
        json.dumps([{"hidden_test": a, "contract": b} for a, b in F1_COVERAGE], indent=2) + "\n"
    )
    return results


def _validate_families(
    results: dict[str, dict[int, Path]],
    *,
    gold_src: Path,
    gold_cfg: str,
    gold_pipe: str,
    mapping: dict[str, str],
    upstream: Path | None,
    f1_hidden: Sequence[HiddenTest],
    f2_hidden: Sequence[HiddenTest],
    f3_hidden: Sequence[HiddenTest],
) -> dict[str, object]:
    out: dict[str, object] = {"ok": True, "families": {}}

    def record(family: str, check: str, ok: bool, detail: str = "") -> None:
        fam = out.setdefault("families", {})
        assert isinstance(fam, dict)
        slot = fam.setdefault(family, {"checks": []})
        assert isinstance(slot, dict)
        slot["checks"].append({"check": check, "ok": ok, "detail": detail[-2000:]})
        if not ok:
            out["ok"] = False

    # F1 gold restore
    f1 = results["spec-reimpl"][0] / "environment" / "src"
    f1_tmp = results["spec-reimpl"][0].parent / "_val_f1"
    _copytree(f1, f1_tmp)
    apply_patch(f1_tmp, results["spec-reimpl"][0] / "tests" / "gold.patch")
    install_hidden_into(f1_tmp, f1_hidden)
    gold_f1 = _run_go(
        ["go", "test", "-count=1", "-timeout", "8m", "-run",
         "^(TestCodecV2|TestParseKeyspaceID|TestDecodeKey|TestEncodeUnknownRequest|TestMalformedRegionKeyIsDecodeError)$",
         "./internal/apicodec/"],
        f1_tmp,
        timeout=540,
    )
    record("spec-reimpl", "gold_restore", gold_f1.returncode == 0, gold_f1.stdout + gold_f1.stderr)

    # F1 buggy hidden tests fail
    f1_bug = results["spec-reimpl"][0].parent / "_val_f1_bug"
    _copytree(f1, f1_bug)
    install_hidden_into(f1_bug, f1_hidden)
    bug_f1 = _run_go(
        ["go", "test", "-count=1", "-timeout", "4m", "-run",
         "^(TestCodecV2|TestParseKeyspaceID|TestDecodeKey|TestMalformedRegionKeyIsDecodeError)$",
         "./internal/apicodec/"],
        f1_bug,
        timeout=300,
    )
    record("spec-reimpl", "buggy_fails", bug_f1.returncode != 0, bug_f1.stdout + bug_f1.stderr)

    # F1 cheat
    cheat_tree = results["spec-reimpl"][0].parent / "_val_f1_cheat"
    _copytree(f1, cheat_tree)
    v2p = cheat_tree / "internal" / "apicodec" / "codec_v2.go"
    v2p.write_text(f1_cheat(v2p.read_text(encoding="utf-8")), encoding="utf-8")
    install_hidden_into(cheat_tree, f1_hidden)
    cheat_f1 = _run_go(
        ["go", "test", "-count=1", "-timeout", "4m", "-run", "^(TestCodecV2)$", "./internal/apicodec/"],
        cheat_tree,
        timeout=300,
    )
    record("spec-reimpl", "cheat_rejected", cheat_f1.returncode != 0, cheat_f1.stdout + cheat_f1.stderr)

    # F1 two-function alt on gold
    alt_tree = results["spec-reimpl"][0].parent / "_val_f1_alt"
    _copytree(f1_tmp, alt_tree)
    gv2 = (gold_src / "internal" / "apicodec" / "codec_v2.go").read_text(encoding="utf-8")
    gc = (gold_src / "internal" / "apicodec" / "codec.go").read_text(encoding="utf-8")
    av2, ac = f1_alt_two_funcs(gv2, gc)
    (alt_tree / "internal" / "apicodec" / "codec_v2.go").write_text(av2, encoding="utf-8")
    (alt_tree / "internal" / "apicodec" / "codec.go").write_text(ac, encoding="utf-8")
    alt_f1 = _run_go(
        ["go", "test", "-count=1", "-timeout", "8m", "-run",
         "^(TestCodecV2|TestParseKeyspaceID|TestDecodeKey|TestEncodeUnknownRequest|TestMalformedRegionKeyIsDecodeError)$",
         "./internal/apicodec/"],
        alt_tree,
        timeout=540,
    )
    record("spec-reimpl", "two_func_alt_accepted", alt_f1.returncode == 0, alt_f1.stdout + alt_f1.stderr)

    rest_f1 = _run_go(
        ["go", "test", "-count=1", "-timeout", "8m", "./internal/unionstore/", "./util/codec/", "./internal/client/retry/"],
        f1,
        timeout=540,
    )
    record("spec-reimpl", "rest_suite_green", rest_f1.returncode == 0, rest_f1.stdout + rest_f1.stderr)

    # F2
    f2 = results["property-backoff"][0] / "environment" / "src"
    f2_gold = results["property-backoff"][0].parent / "_val_f2_gold"
    _copytree(f2, f2_gold)
    (f2_gold / "internal" / "client" / "retry" / "config.go").write_text(gold_cfg, encoding="utf-8")
    install_hidden_into(f2_gold, f2_hidden)
    gold_f2 = _run_go(
        ["go", "test", "-count=1", "-timeout", "2m", "-run", "^TestBackoffExponentialProperty$", "./internal/client/retry/"],
        f2_gold,
        timeout=180,
    )
    record("property-backoff", "gold_restore", gold_f2.returncode == 0, gold_f2.stdout + gold_f2.stderr)

    f2_bug = results["property-backoff"][0].parent / "_val_f2_bug"
    _copytree(f2, f2_bug)
    install_hidden_into(f2_bug, f2_hidden)
    bug_f2 = _run_go(
        ["go", "test", "-count=1", "-timeout", "2m", "-run", "^TestBackoffExponentialProperty$", "./internal/client/retry/"],
        f2_bug,
        timeout=180,
    )
    record("property-backoff", "naive_constant_fails", bug_f2.returncode != 0, bug_f2.stdout + bug_f2.stderr)

    cheat_cfg = gold_cfg.replace(
        "func expo(base, cap, n int) int {\n\treturn int(math.Min(float64(cap), float64(base)*math.Pow(2.0, float64(n))))\n}",
        "func expo(base, cap, n int) int {\n"
        "\tif base == 2 && cap == 500 && n == 0 { return 2 }\n"
        "\tif base == 2 && cap == 500 && n == 1 { return 4 }\n"
        "\tif base == 2 && cap == 500 && n == 8 { return 500 }\n"
        "\treturn base\n"
        "}",
    )
    f2_cheat = results["property-backoff"][0].parent / "_val_f2_cheat"
    _copytree(f2, f2_cheat)
    (f2_cheat / "internal" / "client" / "retry" / "config.go").write_text(cheat_cfg, encoding="utf-8")
    install_hidden_into(f2_cheat, f2_hidden)
    cheat_f2 = _run_go(
        ["go", "test", "-count=1", "-timeout", "2m", "-run", "^TestBackoffExponentialProperty$", "./internal/client/retry/"],
        f2_cheat,
        timeout=180,
    )
    record("property-backoff", "cheat_rejected", cheat_f2.returncode != 0, cheat_f2.stdout + cheat_f2.stderr)

    rest_f2 = _run_go(
        ["go", "test", "-count=1", "-timeout", "8m", "./internal/apicodec/", "./util/codec/"],
        f2,
        timeout=540,
    )
    record("property-backoff", "rest_suite_green", rest_f2.returncode == 0, rest_f2.stdout + rest_f2.stderr)

    # F3 gold + naive + cheat + bench
    f3 = results["dynamic-pipeline"][0] / "environment" / "src"
    f3_gold = results["dynamic-pipeline"][0].parent / "_val_f3_gold"
    _copytree(f3, f3_gold)
    (f3_gold / "internal" / "unionstore" / "pipelined_memdb.go").write_text(gold_pipe, encoding="utf-8")
    install_hidden_into(f3_gold, f3_hidden)
    gold_f3 = _run_go(
        ["go", "test", "-count=1", "-timeout", "4m", "-run", "^(TestPipelinedFlushTrigger|TestPipelinedConcurrentSet)$",
         "./internal/unionstore/"],
        f3_gold,
        timeout=300,
    )
    record("dynamic-pipeline", "gold_restore", gold_f3.returncode == 0, gold_f3.stdout + gold_f3.stderr)
    race_f3 = _run_go(
        ["go", "test", "-race", "-count=1", "-timeout", "4m", "-run", "^(TestPipelinedFlushTrigger|TestPipelinedConcurrentSet)$",
         "./internal/unionstore/"],
        f3_gold,
        timeout=300,
    )
    record("dynamic-pipeline", "gold_race_clean", race_f3.returncode == 0, race_f3.stdout + race_f3.stderr)

    bench_gold = _run_go(
        ["go", "test", "-count=1", "-timeout", "4m", "-bench", "^BenchmarkPipelinedGet$", "-benchtime", "5000x",
         "-run", "^$", "./internal/unionstore/"],
        f3_gold,
        timeout=300,
    )
    gold_ns = parse_bench_ns_op(bench_gold.stdout + bench_gold.stderr, "BenchmarkPipelinedGet")
    record("dynamic-pipeline", "gold_bench", gold_ns is not None, bench_gold.stdout + bench_gold.stderr)

    f3_naive = results["dynamic-pipeline"][0].parent / "_val_f3_naive"
    _copytree(f3, f3_naive)
    (f3_naive / "internal" / "unionstore" / "pipelined_memdb.go").write_text(naive_pipelined_get(gold_pipe), encoding="utf-8")
    install_hidden_into(f3_naive, f3_hidden)
    naive_ok = _run_go(
        ["go", "test", "-count=1", "-timeout", "4m", "-run", "^(TestPipelinedFlushTrigger|TestPipelinedConcurrentSet)$",
         "./internal/unionstore/"],
        f3_naive,
        timeout=300,
    )
    record("dynamic-pipeline", "naive_correctness", naive_ok.returncode == 0, naive_ok.stdout + naive_ok.stderr)
    bench_naive = _run_go(
        ["go", "test", "-count=1", "-timeout", "4m", "-bench", "^BenchmarkPipelinedGet$", "-benchtime", "5000x",
         "-run", "^$", "./internal/unionstore/"],
        f3_naive,
        timeout=300,
    )
    naive_ns = parse_bench_ns_op(bench_naive.stdout + bench_naive.stderr, "BenchmarkPipelinedGet")
    record("dynamic-pipeline", "naive_bench", naive_ns is not None, bench_naive.stdout + bench_naive.stderr)

    limit = int(gold_ns * 3) if gold_ns else None
    naive_fails_gate = bool(limit and naive_ns and naive_ns > limit)
    record(
        "dynamic-pipeline",
        "naive_fails_throughput_gate",
        naive_fails_gate,
        f"gold_ns={gold_ns} naive_ns={naive_ns} limit={limit}",
    )
    out["f3_gold_ns_op"] = gold_ns
    out["f3_naive_ns_op"] = naive_ns
    out["f3_perf_limit_ns"] = limit

    try:
        from openswe_traces.synth.difficulty import find_perf_gates, race_gate

        gates = [
            {"name": g.name, "file_path": g.file_path, "kind": g.kind, "package": g.package}
            for g in find_perf_gates(gold_src)
            if "Pipelined" in g.name or g.name == "BenchmarkGet"
        ]
        out["find_perf_gates"] = gates
        site = race_gate(gold_src)
        out["race_gate"] = None if site is None else {
            "name": site.name,
            "file_path": site.file_path,
            "reason": site.reason,
        }
        record("dynamic-pipeline", "find_perf_gates", True, str(gates)[:500])
        record("dynamic-pipeline", "race_gate", True, str(out["race_gate"]))
    except FileNotFoundError as exc:
        out["find_perf_gates"] = []
        out["race_gate"] = None
        record("dynamic-pipeline", "find_perf_gates", True, f"no codegraph index: {exc}")
        record("dynamic-pipeline", "race_gate", True, f"no codegraph index: {exc}")

    f3_bug = results["dynamic-pipeline"][0].parent / "_val_f3_bug"
    _copytree(f3, f3_bug)
    install_hidden_into(f3_bug, f3_hidden)
    bug_f3 = _run_go(
        ["go", "test", "-count=1", "-timeout", "4m", "-run", "^TestPipelinedFlushTrigger$", "./internal/unionstore/"],
        f3_bug,
        timeout=300,
    )
    record("dynamic-pipeline", "buggy_fails", bug_f3.returncode != 0, bug_f3.stdout + bug_f3.stderr)

    cheat_pipe = gold_pipe.replace(
        "func (p *PipelinedMemDB) Get(ctx context.Context, k []byte) ([]byte, error) {",
        "func (p *PipelinedMemDB) Get(ctx context.Context, k []byte) ([]byte, error) {\n"
        "\tif len(k) == 1 && k[0] == '0' {\n"
        "\t\treturn []byte{'0'}, nil\n"
        "\t}",
        1,
    )
    f3_cheat = results["dynamic-pipeline"][0].parent / "_val_f3_cheat"
    _copytree(f3, f3_cheat)
    (f3_cheat / "internal" / "unionstore" / "pipelined_memdb.go").write_text(stub_pipelined_get_flush(cheat_pipe), encoding="utf-8")
    install_hidden_into(f3_cheat, f3_hidden)
    cheat_f3 = _run_go(
        ["go", "test", "-count=1", "-timeout", "4m", "-run", "^TestPipelinedFlushTrigger$", "./internal/unionstore/"],
        f3_cheat,
        timeout=300,
    )
    record("dynamic-pipeline", "cheat_rejected", cheat_f3.returncode != 0, cheat_f3.stdout + cheat_f3.stderr)

    rest_f3 = _run_go(
        ["go", "test", "-count=1", "-timeout", "8m", "./internal/unionstore/", "./internal/apicodec/", "./internal/client/retry/"],
        f3,
        timeout=540,
    )
    record("dynamic-pipeline", "rest_suite_green", rest_f3.returncode == 0, rest_f3.stdout + rest_f3.stderr)

    # instruction self-check already stored; searchability
    if upstream and upstream.is_dir():
        idents = distinctive_identifiers(f1, mapping) if mapping else ["WillowHeap", "NimbusPack", "YarrowJoin"]
        rows = searchability_check(idents, upstream)
        out["searchability"] = rows
        record("spec-reimpl", "searchability", all(r.get("ok") for r in rows), json.dumps(rows))
    else:
        out["searchability"] = []
        record("spec-reimpl", "searchability", True, "upstream missing; skipped")

    # F1 instruction self-check
    instr = (results["spec-reimpl"][0] / "instruction.md").read_text(encoding="utf-8")
    sh = (results["spec-reimpl"][0] / "tests" / "test.sh").read_text(encoding="utf-8")
    chk = instruction_self_check(
        instr,
        test_sh=sh,
        f2p_tests=[n for h in f1_hidden for n in h.names()],
        changed_symbols=F1_CHANGED_SYMBOLS,
        changed_files=F1_CHANGED_FILES,
        locality=2,
        packages=("internal/apicodec",),
    )
    record("spec-reimpl", "instruction_self_check", bool(chk.get("ok")), json.dumps(chk))
    out["f1_instruction_self_check"] = chk
    return out


def stage_a0_only(dest_root: Path, a0_dir: Path) -> Path:
    """Copy the three A0 tasks into ``a0_dir`` for a Harbor --path that has only A0."""
    a0_dir = Path(a0_dir)
    if a0_dir.exists():
        shutil.rmtree(a0_dir)
    a0_dir.mkdir(parents=True)
    for family in ("spec-reimpl", "property-backoff", "dynamic-pipeline"):
        src = Path(dest_root) / f"{family}-A0"
        _copytree(src, a0_dir / f"{family}-A0")
    return a0_dir


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Build unsolvable→affordance Harbor tasks")
    parser.add_argument(
        "--obf-task",
        default=str(ROOT / "experiments/harbor_nex/tasks_obf/client-go-keyspacecodec-obf"),
    )
    parser.add_argument(
        "--dest",
        default=str(ROOT / "experiments/harbor_nex/tasks_unsolv"),
    )
    parser.add_argument(
        "--upstream",
        default=str(ROOT / "experiments/harbor_nex/tasks_iter9/client-go-keyspacecodec/environment/src"),
    )
    parser.add_argument("--skip-go", action="store_true")
    args = parser.parse_args(argv)
    construct_unsolv_families(args.obf_task, args.dest, upstream=args.upstream, skip_go=args.skip_go)
    stage_a0_only(Path(args.dest), Path(args.dest).parent / "tasks_unsolv_A0")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
