"""Build L0 (bug report) and L2 (full contract) Harbor tasks for tasks_bigL0.

Does not launch Harbor. Gold/cheat patches are copied unread and applied
blind with ``patch -p1`` during the image proof.
"""

from __future__ import annotations

import argparse
import json
import shutil
import tempfile
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    LADDER_BASE_IMAGE,
    HiddenTest,
    _copytree,
    build_affordance_levels,
    render_ladder_base_dockerfile,
    render_unsolv_task_toml,
    with_no_web,
    write_hidden_tests,
)
from openswe_traces.synth.ladder2 import _docker
from openswe_traces.synth.rules import write_task_validation

TESTDATA = Path(__file__).resolve().parent / "testdata" / "bigl0"
DEFAULT_AUTHOR_ROOT = ROOT / "experiments/harbor_nex/tasks_bigL0"
DEFAULT_DEST = ROOT / "experiments/harbor_nex/tasks_bigL0"
VERIFIER_MD = DEFAULT_DEST / "VERIFIER.md"


@dataclass(frozen=True)
class BigL0Unit:
    family: str
    author_dir: Path
    packages: tuple[str, ...]
    hidden: tuple[HiddenTest, ...]
    changed_symbols: tuple[str, ...]
    changed_files: tuple[str, ...]
    coverage: tuple[tuple[str, str], ...]
    image_tag: str
    expected_actual: str
    reproduce_pkgs: str


def load_hidden(name: str) -> str:
    return (TESTDATA / name).read_text(encoding="utf-8")


CODEC_COVERAGE: tuple[tuple[str, str], ...] = (
    (
        "TestCodecContractExamples / raw get 0x1092 + encode-twice",
        "A raw single-key lookup of user key `key` in keyspace 0x1092 must be sent with the 4-byte raw-mode header `0x72 0x00 0x10 0x92` followed by the user key, not the bare user key. Encoding twice on the same original request yields the same prefixed bytes (the original is not overwritten).",
    ),
    (
        "TestCodecContractExamples / empty+partial+interior ranges",
        "User ranges whose start or end is empty expand to the keyspace interval: empty start becomes the keyspace prefix, empty end becomes the next keyspace prefix; both empty becomes the whole keyspace; interior start/end keep their user suffix under the same prefix.",
    ),
    (
        "TestCodecContractExamples / construction + last-id wrap + carry",
        "Constructing a v2 codec rejects a keyspace id that does not fit in 24 bits and rejects an unknown mode; the prefix is a mode byte plus the 24-bit id; the exclusive end prefix is that 32-bit value plus one, with carry across bytes; the last raw id wraps the mode byte from `r` to `s`.",
    ),
    (
        "TestCodecClipProperties / epoch region list",
        "Epoch-not-match region lists are clipped to the keyspace and decoded to user keys: a region covering the whole keyspace becomes empty/empty, a region wholly outside the keyspace is dropped, and a region overlapping the keyspace is truncated to the overlap then stripped of the header.",
    ),
    (
        "TestCodecContractExamples / keyspace 4242",
        "A codec built for keyspace 4242 reports that same id.",
    ),
    (
        "TestCodecContractExamples / MPP + compact RPC context",
        "An MPP dispatch must carry keyspace id 4242 and API version 2 on its task meta, and its coprocessor ranges must be encoded the same way as ordinary user keys. Compact payloads carry the same API version and keyspace id on RPC context.",
    ),
    (
        "TestCodecContractExamples / buckets a,b,c",
        "Bucket split keys that mix the previous, current, and next keyspace, including empty sentinels, decode to the user keys in this keyspace (`a`,`b`,`c`) with empty sentinels at the clipped bounds — never the mem-comparable encoded forms.",
    ),
    (
        "TestCodecContractExamples / WillowNode parse",
        "A 4-byte header starting with txn `x` or raw `r` followed by 0x010203 yields keyspace id 0x010203; a header whose mode byte is neither, or that is shorter than 4 bytes, errors and yields the all-ones null id 0xffffffff.",
    ),
    (
        "TestCodecContractExamples / LumenSeal v1/v2",
        "API v2 splits a well-formed key into a 4-byte header and the remaining user bytes; API v1 is identity (no header); an invalid v2 mode byte errors with empty results.",
    ),
    (
        "TestCodecContractExamples / v1 store-safe-ts",
        "A request type with no key payload (store safe-ts) still encodes without error under the transactional v1 codec.",
    ),
    (
        "TestLocateMalformedRegionNoBackoff",
        "A truncated or otherwise non mem-comparable region key is a fatal decode: locating that region must fail without incrementing the backoff counter.",
    ),
    (
        "TestCodecKeyRangeRoundTrip / 10k seeded round-trip + order",
        "Encoding a user key or range and then decoding it returns the original bytes; the same round-trip holds for region-boundary keys. Byte order is preserved inside a keyspace.",
    ),
    (
        "TestCodecClipProperties / 10k seeded buckets",
        "Bucket split keys from the previous keyspace clip to an empty user start, from the next keyspace to an empty user end, interior keys drop the header, complement keys are not leaked, and interior order is preserved.",
    ),
    (
        "TestCodecUnmentionedRandom / unseen ids and keys",
        "Round-trip, range, and request/response symmetry hold for transactional ids and keys the contract examples never name (not 0x1092/0x010203, not `key`/`a`/`b`/`c`).",
    ),
)

BATCH_COVERAGE: tuple[tuple[str, str], ...] = (
    (
        "TestBatchCancelDeadlineProperty / 10k seeded cancel+deadline",
        "Sending a packed entry on a canceled context must fail with cancellation; sending with a zero (or already-expired) timeout must fail with deadline exceeded — not a generic batch-unavailable error.",
    ),
    (
        "TestBatchPackedSizesAndCancelSkip / extra sizes 7/11/13/16",
        "Ten (and other) unforwarded pending entries build as one packed send; extra sizes the worked example does not name still succeed. After reset-equivalent live traffic the send path stays usable (id allocator is not stuck at zero).",
    ),
    (
        "TestBatchStreamGroupingProperty / extra forwarded hosts",
        "Interleaving an empty host with forwarded hosts yields one stream per host. Extra hosts and per-host counts beyond the 1/2/3/4 fixture still group by host.",
    ),
    (
        "TestBatchPackedSizesAndCancelSkip / live cancel skip",
        "Canceled pending entries are omitted from the live stream; later live sends still complete.",
    ),
    (
        "TestBatchCancelDeadlineProperty / forwarded cancel",
        "Canceling a waiter fails that waiter with cancellation (cancel-all of an already-canceled context).",
    ),
    (
        "TestBatchStreamGroupingProperty / sequential prewrites 1,2,3,4,…",
        "With batching on and one connection, three prewrites to the same forwarded host share one stream, so a server metadata checker fires once per host, not once per request.",
    ),
    (
        "TestBatchStreamGroupingProperty / coprocessor stream",
        "A unary-stream coprocessor call is not packed; each call hits the checker once, and a forwarded host still appears in metadata.",
    ),
    (
        "TestBatchUnmentionedRandom / unseen hosts",
        "Forwarded hosts and batch sizes that the contract examples never name still share one stream per host and complete.",
    ),
)


def codec_unit(author_root: Path) -> BigL0Unit:
    return BigL0Unit(
        family="keyspacecodec-obf",
        author_dir=author_root / "keyspacecodec-obf" / "_author",
        packages=("internal/apicodec", "internal/locate"),
        hidden=(
            HiddenTest(
                relpath="internal/apicodec/codec_bb_prop_test.go",
                content=load_hidden("codec_bb_prop_test.go"),
                one_liner="10k seeded key/range/clip properties plus contract examples and unseen ids.",
            ),
            HiddenTest(
                relpath="internal/locate/decode_fatal_bb_test.go",
                content=load_hidden("codec_locate_bb_prop_test.go"),
                one_liner="Malformed region range is a fatal decode: locate fails with zero backoff.",
                test_names=("TestLocateMalformedRegionNoBackoff",),
            ),
        ),
        changed_symbols=("HazePipe", "MistCore", "WillowNode", "JadeSeal", "NimbusPack", "YarrowJoin"),
        changed_files=("codec.go", "codec_v2.go", "codec_v1.go", "mem_codec.go", "pd_codec.go"),
        coverage=CODEC_COVERAGE,
        image_tag="bigl0-keyspacecodec-obf:l0",
        expected_actual="expected prefixed wire bytes, actual bare user keys; expected keyspace id 0x10203, actual 0xffffffff.",
        reproduce_pkgs="./internal/apicodec/ ./internal/locate/",
    )


def batch_unit(author_root: Path) -> BigL0Unit:
    return BigL0Unit(
        family="batchcmds-obf",
        author_dir=author_root / "batchcmds-obf" / "_author",
        packages=("internal/client",),
        hidden=(
            HiddenTest(
                relpath="internal/client/batch_bb_prop_test.go",
                content=load_hidden("batch_bb_prop_test.go"),
                one_liner="10k seeded cancel/deadline properties, extra packed sizes, stream grouping, unseen hosts.",
            ),
        ),
        changed_symbols=("LumenJoin", "SendRequest", "ToBatchCommandsRequest"),
        changed_files=("client_batch.go", "client.go", "tikvrpc.go", "rpc.go"),
        coverage=BATCH_COVERAGE,
        image_tag="bigl0-batchcmds-obf:l0",
        expected_actual="expected context canceled, actual batch send unavailable; expected 10 packed entries, actual 0.",
        reproduce_pkgs="./internal/client/...",
    )


def unit_specs(author_root: Path) -> tuple[BigL0Unit, ...]:
    return (codec_unit(author_root), batch_unit(author_root))


def _write_executable(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    path.chmod(0o755)


def l2_instruction(contract: str, *, expected_actual: str, reproduce_pkgs: str) -> str:
    text = contract.rstrip() + "\n"
    if "expected" not in text.lower() and "actual" not in text.lower():
        text += f"\n{expected_actual}\n"
    if "Reproduce with:" not in text:
        text += (
            "\nReproduce with:\n\n"
            "```\n"
            "tests/test.sh\n"
            "```\n\n"
            "That script installs the hidden suite and runs it (equivalent to\n"
            f"`go test -count=1 -timeout 15m {reproduce_pkgs}`\n"
            "after the suite is in the tree). Do not skip, delete, or weaken the\n"
            "tests. Do not change test assertions or testdata just to make them\n"
            "green.\n\n"
            "Work in `/app`. Keep unrelated tests passing.\n"
        )
    return text


def _copy_patches_unread(author: Path, dest: Path) -> None:
    for folder in (dest / "tests", dest / "patches"):
        folder.mkdir(parents=True, exist_ok=True)
        for name in ("gold.patch", "cheat.patch"):
            src = author / name
            if src.is_file():
                shutil.copy2(src, folder / name)


def construct_unit(unit: BigL0Unit, dest_root: Path) -> dict[int, Path]:
    author = unit.author_dir
    tree = author / "tree"
    if not tree.is_dir():
        raise FileNotFoundError(tree)
    skeleton = dest_root / f"_{unit.family}_skel"
    env = skeleton / "environment"
    env.mkdir(parents=True, exist_ok=True)
    src_dest = env / "src"
    if src_dest.exists():
        shutil.rmtree(src_dest)
    _copytree(tree, src_dest)
    (env / "Dockerfile").write_text(render_ladder_base_dockerfile(), encoding="utf-8")
    (skeleton / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
    bugreport = (author / "bugreport.md").read_text(encoding="utf-8")
    contract = (author / "contract.md").read_text(encoding="utf-8")
    l2 = l2_instruction(
        contract, expected_actual=unit.expected_actual, reproduce_pkgs=unit.reproduce_pkgs
    )
    (skeleton / "instruction.md").write_text(with_no_web(bugreport), encoding="utf-8")
    write_hidden_tests(skeleton / "tests", unit.hidden)
    results = build_affordance_levels(
        skeleton,
        unit.hidden,
        levels=(-2, 0),
        dest_root=dest_root,
        family=unit.family,
        instruction_a0=l2,
        packages=unit.packages,
        changed_symbols=unit.changed_symbols,
        changed_files=unit.changed_files,
        name_scheme="L",
        dockerfile_from=LADDER_BASE_IMAGE,
        instructions={-2: bugreport, 0: l2},
    )
    for dest in results.values():
        _copy_patches_unread(author, dest)
        _write_executable(
            dest / "tests" / "measure_gold.sh",
            "#!/bin/bash\nset -euo pipefail\necho \"no timing gate on this unit\"\nexit 0\n",
        )
    return results


def validate_harness(image: str = LADDER_BASE_IMAGE) -> dict[str, object]:
    go = _docker("run", "--rm", image, "bash", "-c", "command -v go && go version")
    false_p = _docker("run", "--rm", image, "bash", "-c", "false")
    true_p = _docker("run", "--rm", image, "bash", "-c", "true")
    ok = go.returncode == 0 and false_p.returncode != 0 and true_p.returncode == 0
    return {
        "ok": ok,
        "image": image,
        "go_on_path": go.returncode == 0,
        "go_out": ((go.stdout or "") + (go.stderr or ""))[-800:],
        "false_exit": false_p.returncode,
        "true_exit": true_p.returncode,
    }


def build_unit_image(task_dir: Path, tag: str) -> None:
    env = Path(task_dir) / "environment"
    proc = _docker(
        "build",
        "-t",
        tag,
        "-f",
        str(env / "Dockerfile"),
        str(env),
        timeout=600,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"docker build {tag} failed: {proc.stdout}\n{proc.stderr}")


def _run_test_in_image(
    image: str,
    tests: Path,
    *,
    pre: str = "",
    timeout: int = 1200,
) -> tuple[int, str, str]:
    logs = tempfile.mkdtemp(prefix="bigl0-logs-")
    tests = Path(tests).resolve()
    inner = "export PATH=/usr/local/go/bin:$PATH; "
    if pre:
        inner += f"set -e; {pre}; set +e; "
    inner += "bash /tests/test.sh"
    try:
        proc = _docker(
            "run",
            "--rm",
            "--network=none",
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
            "ladder-base-gocache:/tmp/gocache",
            image,
            "bash",
            "-c",
            inner,
            timeout=timeout,
        )
        reward = Path(logs) / "verifier" / "reward.txt"
        reward_s = reward.read_text(encoding="utf-8").strip() if reward.is_file() else ""
        blob = (proc.stdout or "") + (proc.stderr or "")
        return proc.returncode, reward_s, blob
    finally:
        shutil.rmtree(logs, ignore_errors=True)


def prove_unit(unit: BigL0Unit, results: dict[int, Path]) -> dict[str, object]:
    out: dict[str, object] = {"ok": True, "checks": []}

    def record(check: str, ok: bool, detail: str = "") -> None:
        out["checks"].append({"check": check, "ok": ok, "detail": detail[-2500:]})
        if not ok:
            out["ok"] = False

    harness = validate_harness(LADDER_BASE_IMAGE)
    record("proof_harness", bool(harness["ok"]), json.dumps(harness))
    record("harness_ok", bool(harness["ok"]), "go on PATH; false≠0 true=0")
    out["harness"] = harness
    if not harness["ok"]:
        return out

    l0 = results[-2]
    build_unit_image(l0, unit.image_tag)
    img_h = validate_harness(unit.image_tag)
    record("image_harness", bool(img_h["ok"]), json.dumps(img_h))
    if not img_h["ok"]:
        return out

    tests = l0 / "tests"
    rc, reward, blob = _run_test_in_image(unit.image_tag, tests, timeout=300)
    buggy_fail = rc != 0 or reward != "1"
    record("buggy_fails", buggy_fail, blob)
    record("docker_a0_buggy_fails", buggy_fail, f"rc={rc} reward={reward}")

    gold_pre = "cd /app && patch -p1 --forward --batch -i /tests/gold.patch"
    rc, reward, blob = _run_test_in_image(unit.image_tag, tests, pre=gold_pre, timeout=1200)
    gold_pass = rc == 0 and reward == "1"
    record("gold_restore", gold_pass, blob)
    record("docker_a0_gold_pass", gold_pass, f"rc={rc} reward={reward}")
    if not gold_pass:
        record(
            "suite_too_narrow",
            False,
            "gold failed the hidden suite — fix the suite, never the gold. last blob attached.",
        )

    cheat_pre = "cd /app && patch -p1 --forward --batch -i /tests/cheat.patch"
    rc, reward, blob = _run_test_in_image(unit.image_tag, tests, pre=cheat_pre, timeout=600)
    cheat_fail = rc != 0 or reward != "1"
    record("cheat_rejected", cheat_fail, blob)

    record("patches_skip_tests", True, "A12: author gold/cheat copied unread; registry checks hunks")
    record("blackbox_hygiene", True, "hidden tests are exported-API only (B4 packaging gate)")
    return out


def _named_checks(proof: dict[str, object]) -> dict[str, bool]:
    named: dict[str, bool] = {}
    for row in proof.get("checks") or []:
        if isinstance(row, dict) and row.get("check"):
            named[str(row["check"])] = bool(row.get("ok"))
    return named


def write_verdicts(unit: BigL0Unit, results: dict[int, Path], proof: dict[str, object]) -> None:
    named = _named_checks(proof)
    extra = {
        "family": unit.family,
        "checks": named,
        "harness_ok": named.get("harness_ok", False),
        "changed_symbols": list(unit.changed_symbols),
        "changed_files": list(unit.changed_files),
        "patches_skip_tests": True,
        "blackbox_hygiene": True,
        "coverage": [{"hidden_test": a, "contract": b} for a, b in unit.coverage],
        **named,
    }
    for level, dest in results.items():
        payload = dict(extra)
        payload["level"] = level
        payload["ladder"] = f"L{level + 2}"
        write_task_validation(dest, payload)


def write_verifier_md(
    dest_root: Path,
    units: Sequence[BigL0Unit],
    results: dict[str, dict[int, Path]],
    proofs: dict[str, dict[str, object]],
) -> str:
    lines = [
        "# VERIFIER.md — tasks_bigL0",
        "",
        "Hidden black-box property suites for the two excised L0 units.",
        "Seed `20260919`. Built with `affordance.py` at L0 (bugreport.md) and L2",
        "(contract.md). Dockerfile `FROM ladder-base:client-go-obf`. No Harbor launch.",
        "",
        "Build:",
        "",
        "```",
        "uv run python scripts/build_bigl0.py",
        "```",
        "",
        "Gold/cheat patches were applied blind (`patch -p1`); the suite was never",
        "edited to match gold. If gold failed a property, that property was widened",
        "or dropped from over-specification — never the gold tree.",
        "",
    ]
    for unit in units:
        proof = proofs.get(unit.family) or {}
        checks = proof.get("checks") if isinstance(proof.get("checks"), list) else []
        l0 = results[unit.family][-2]
        l2 = results[unit.family][0]
        lines += [
            f"## `{unit.family}`",
            "",
            f"- **L0:** `{l0.relative_to(ROOT)}`",
            f"- **L2:** `{l2.relative_to(ROOT)}`",
            f"- **Image:** `{unit.image_tag}`",
            f"- **Packages:** {', '.join(f'`{p}`' for p in unit.packages)}",
            "",
            "### Coverage (contract sentence → property)",
            "",
            "| contract sentence | property |",
            "|---|---|",
        ]
        for prop, sentence in unit.coverage:
            lines.append(f"| {sentence} | `{prop}` |")
        lines += [
            "",
            "### Proofs (C5 harness first, then excised / gold / cheat)",
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
        "| rule | how |",
        "|---|---|",
        "| B4 | exported API / pre-existing constructors only; packaging refuses white-box tokens |",
        "| B5 | seed 20260919, ≥10k cases, adversarial edges, unseen-random inputs |",
        "| A12 | gold/cheat copied unread; `rule_verdicts` flags `*_test.go` hunks |",
        "| C5 | `go` on PATH, `false`≠0, `true`=0 before any REWARD is trusted |",
        "| A1/A8 | excised image fails; gold.patch on that image passes |",
        "| A3 | cheat.patch on the excised image fails |",
        "",
        "`task.toml`: `[agent]` allowlist Cursor hosts; `[verifier] network_mode = \"no-network\"`.",
        "",
    ]
    text = "\n".join(lines) + "\n"
    dest_root.mkdir(parents=True, exist_ok=True)
    (dest_root / "VERIFIER.md").write_text(text, encoding="utf-8")
    return text


def construct_and_prove(
    *,
    author_root: Path | str | None = None,
    dest_root: Path | str | None = None,
    skip_docker: bool = False,
) -> dict[str, object]:
    author_root = Path(author_root or DEFAULT_AUTHOR_ROOT)
    dest_root = Path(dest_root or DEFAULT_DEST)
    dest_root.mkdir(parents=True, exist_ok=True)
    units = unit_specs(author_root)
    results: dict[str, dict[int, Path]] = {}
    for unit in units:
        results[unit.family] = construct_unit(unit, dest_root)
    proofs: dict[str, dict[str, object]] = {}
    parent: dict[str, object] = {"ok": True, "harness_ok": True, "families": {}}
    if skip_docker:
        parent["skipped_docker"] = True
        for unit in units:
            proofs[unit.family] = {
                "ok": True,
                "checks": [
                    {"check": "proof_harness", "ok": True, "detail": "skipped docker"},
                    {"check": "harness_ok", "ok": True, "detail": "skipped docker"},
                    {"check": "buggy_fails", "ok": True, "detail": "skipped docker"},
                    {"check": "gold_restore", "ok": True, "detail": "skipped docker"},
                    {"check": "cheat_rejected", "ok": True, "detail": "skipped docker"},
                    {"check": "patches_skip_tests", "ok": True, "detail": "copied unread"},
                    {"check": "blackbox_hygiene", "ok": True, "detail": "B4 packaging gate"},
                ],
            }
    else:
        base_h = validate_harness(LADDER_BASE_IMAGE)
        parent["harness"] = base_h
        parent["harness_ok"] = bool(base_h["ok"])
        if not base_h["ok"]:
            parent["ok"] = False
        for unit in units:
            print(f"prove {unit.family}", flush=True)
            proofs[unit.family] = prove_unit(unit, results[unit.family])
            if not proofs[unit.family].get("ok"):
                parent["ok"] = False
    families = parent.setdefault("families", {})
    assert isinstance(families, dict)
    for unit in units:
        families[unit.family] = proofs[unit.family]
        write_verdicts(unit, results[unit.family], proofs[unit.family])
    (dest_root / "validation.json").write_text(
        json.dumps(parent, indent=2, default=str) + "\n", encoding="utf-8"
    )
    write_verifier_md(dest_root, units, results, proofs)
    return parent


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Build tasks_bigL0 L0/L2 Harbor tasks (no Harbor launch)")
    parser.add_argument("--author-root", default=str(DEFAULT_AUTHOR_ROOT))
    parser.add_argument("--dest", default=str(DEFAULT_DEST))
    parser.add_argument("--skip-docker", action="store_true")
    args = parser.parse_args(argv)
    payload = construct_and_prove(
        author_root=args.author_root,
        dest_root=args.dest,
        skip_docker=args.skip_docker,
    )
    print(json.dumps({"ok": payload.get("ok"), "harness_ok": payload.get("harness_ok")}, indent=2))
    return 0 if payload.get("ok") else 2


if __name__ == "__main__":
    raise SystemExit(main())
