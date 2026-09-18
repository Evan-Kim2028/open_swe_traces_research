"""Black-box affordance family for the keyspace codec (spec-reimpl-bb)."""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
from collections.abc import Sequence
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    F1_CHANGED_FILES,
    F1_CHANGED_SYMBOLS,
    HiddenTest,
    _copytree,
    _run_go,
    _write_skeleton,
    apply_patch,
    build_affordance_levels,
    f1_a2_stubs,
    install_hidden_into,
    instruction_self_check,
    strip_go_func,
    with_no_web,
    write_hidden_tests,
)
from openswe_traces.synth.obfuscate import distinctive_identifiers, searchability_check

BB_FAMILY = "spec-reimpl-bb"
BB_TEST_REL = "internal/apicodec/codec_bb_prop_test.go"
_TESTDATA = Path(__file__).resolve().parent / "testdata" / "codec_bb_prop_test.go"

WHITEBOX_TOKENS = (
    "ThornSlot",
    "NimbusCore",
    "codecV2",
    "*codecV2",
    "memCodec",
    "PebbleUnit",
    "IvoryLink",
    "RidgeWire",
)

# Original (unobfuscated) callers outside apicodec, mapped via mapping.json.
EXPORTED_CALLERS: tuple[dict[str, str], ...] = (
    {
        "original": "NewCodecV2",
        "obfuscated": "NimbusPack",
        "callers": "tikv/region.go re-export; locate/pd_codec.go NewCodecPDClientWithKeyspace",
    },
    {
        "original": "NewCodecV1",
        "obfuscated": "ZestRing",
        "callers": "locate/pd_codec.go, region_cache.go, replica/request tests",
    },
    {
        "original": "Codec.EncodeRequest",
        "obfuscated": "MistCore",
        "callers": "internal/client/client.go; kvclient/test_util.go",
    },
    {
        "original": "Codec.DecodeResponse",
        "obfuscated": "CedarPath",
        "callers": "internal/client/client.go; kvclient/test_util.go",
    },
    {
        "original": "Codec.EncodeRegionKey",
        "obfuscated": "JadePort",
        "callers": "locate/pd_codec.go GetRegion/GetPrevRegion/SplitRegions; region_cache.go",
    },
    {
        "original": "Codec.EncodeRegionRange",
        "obfuscated": "SablePack",
        "callers": "locate/pd_codec.go ScanRegions",
    },
    {
        "original": "Codec.DecodeRegionRange",
        "obfuscated": "ThornUnit",
        "callers": "locate/pd_codec.go processRegionResult",
    },
    {
        "original": "Codec.DecodeBucketKeys",
        "obfuscated": "AmberGate",
        "callers": "locate/pd_codec.go processRegionResult",
    },
    {
        "original": "Codec.GetAPIVersion",
        "obfuscated": "QuartzSlot",
        "callers": "locate/region_request.go",
    },
    {
        "original": "IsDecodeError",
        "obfuscated": "JadeSeal",
        "callers": "locate/region_cache.go",
    },
    {
        "original": "Codec.EncodeKey",
        "obfuscated": "HazePipe",
        "callers": "YarrowJoin interface (held by client + PD wrapper; used inside EncodeRequest)",
    },
    {
        "original": "Codec.DecodeKey",
        "obfuscated": "MistUnit",
        "callers": "YarrowJoin interface (DecodeResponse path)",
    },
    {
        "original": "Codec.EncodeRange",
        "obfuscated": "CedarUnit",
        "callers": "YarrowJoin interface (request range encoding)",
    },
    {
        "original": "Codec.DecodeRange",
        "obfuscated": "QuartzPort",
        "callers": "YarrowJoin interface (region-range decode)",
    },
    {
        "original": "Codec.DecodeRegionKey",
        "obfuscated": "EmberSlot",
        "callers": "YarrowJoin interface (v1 region decode + fatal path)",
    },
    {
        "original": "Codec.GetKeyspaceID",
        "obfuscated": "ThornRef",
        "callers": "YarrowJoin interface (RPC context attach)",
    },
    {
        "original": "Codec.GetKeyspace",
        "obfuscated": "LumenRef",
        "callers": "YarrowJoin interface (4-byte prefix)",
    },
    {
        "original": "ParseKeyspaceID",
        "obfuscated": "WillowNode",
        "callers": "package export (header parse)",
    },
    {
        "original": "DecodeKey",
        "obfuscated": "LumenSeal",
        "callers": "package export (v1/v2 split)",
    },
)

BB_COVERAGE: tuple[tuple[str, str], ...] = (
    (
        "TestCodecKeyRangeRoundTrip / key+range encode then decode",
        "Encoding a user key or range and then decoding it returns the original bytes; the same round-trip holds for region-boundary keys.",
    ),
    (
        "TestCodecKeyRangeRoundTrip / monotonicity",
        "If user key A is byte-wise less than B, both the ordinary encoding and the region-boundary encoding of A are byte-wise less than those of B (order preserved within a keyspace).",
    ),
    (
        "TestCodecClipProperties / epoch region list",
        "Epoch-not-match region lists are clipped to the keyspace: a region covering the whole keyspace becomes empty/empty, a region wholly outside is dropped, an overlapping region keeps only the overlap as user keys, and remaining regions stay in original order.",
    ),
    (
        "TestCodecClipProperties / buckets",
        "Bucket split keys from the previous keyspace clip to an empty user start, from the next keyspace to an empty user end, interior keys drop the header, complement keys are not leaked, and interior order is preserved.",
    ),
    (
        "TestCodecContractExamples / raw get 0x1092",
        "A raw single-key lookup of user key `key` in keyspace 0x1092 must be sent with the 4-byte raw-mode header `0x72 0x00 0x10 0x92` followed by the user key, not the bare user key.",
    ),
    (
        "TestCodecContractExamples / WillowNode + LumenSeal",
        "A 4-byte header starting with txn `x` or raw `r` followed by 0x010203 yields keyspace id 0x010203; a header whose mode byte is neither errors with the all-ones null id. API v2 splits a well-formed key into a 4-byte header and the user suffix; API v1 is identity; an invalid v2 mode byte errors.",
    ),
    (
        "TestCodecContractExamples / empty range + construction",
        "Empty user ranges in keyspace 0x1092 expand to `[0x72 0x00 0x10 0x92, 0x72 0x00 0x10 0x93)`. Construction rejects a keyspace id that does not fit in 24 bits and rejects an unknown mode; the last raw id wraps the mode byte from `r` to `s`.",
    ),
    (
        "TestCodecContractExamples / MPP + buckets + v1 + fatal",
        "An MPP dispatch in keyspace 4242 must advertise that id, API v2, and encoded ranges. Bucket keys `a`,`b`,`c` mixed with previous/next neighbors round-trip to those user keys plus empty sentinels. A store-safe-ts request still encodes on the v1 transactional codec. A truncated region key is a fatal decode.",
    ),
    (
        "TestCodecUnmentionedRandom / three properties on unseen inputs",
        "The same round-trip, range, and request/response symmetry rules hold for randomly generated transactional keyspace ids and keys that the instruction never names (not 0x1092/4242/0x010203, not `key`/`a`/`b`/`c`).",
    ),
)

BB_INSTRUCTION = """# Missing behavior

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


def bb_hidden_test() -> HiddenTest:
    content = _TESTDATA.read_text(encoding="utf-8")
    for tok in WHITEBOX_TOKENS:
        if tok in content:
            raise ValueError(f"black-box test leaks white-box token {tok}")
    return HiddenTest(
        relpath=BB_TEST_REL,
        content=content,
        one_liner=(
            "Property tests (seed 20260918, 10k cases): key/range round-trip, "
            "byte-order, region/epoch/bucket clip, request/response symmetry, "
            "contract examples, and unseen random inputs."
        ),
        test_names=(
            "TestCodecKeyRangeRoundTrip",
            "TestCodecClipProperties",
            "TestCodecContractExamples",
            "TestCodecUnmentionedRandom",
        ),
    )


def _strip_package_tests(src: Path) -> None:
    pkg = src / "internal" / "apicodec"
    if not pkg.is_dir():
        return
    for path in pkg.glob("*_test.go"):
        path.unlink()


def _replace_go_func(text: str, new_func: str) -> str:
    header = new_func.strip().split("{", 1)[0].strip()
    out = strip_go_func(text, header)
    return out.rstrip() + "\n\n" + new_func.strip() + "\n"


def _case_block_end(text: str, case_at: int) -> int:
    nxt = re.search(r"\n\tcase |\n\tdefault:|\n}", text[case_at + 1 :])
    if not nxt:
        return len(text)
    return case_at + 1 + nxt.start()


def _apply_unique_snippet(text: str, snippet: str) -> tuple[str, str]:
    """Replace the unique first-line span with ``snippet`` (Cursor streamContent)."""
    snip = snippet.replace("\r\n", "\n")
    if snip in text:
        return text, "already"
    lines = [ln for ln in snip.split("\n") if ln.strip() != ""]
    if not lines:
        return text, "empty"
    first, last = lines[0], lines[-1]
    if first.lstrip().startswith("import ("):
        updated = re.sub(r"import \([^)]*\)", snip.rstrip(), text, count=1)
        if updated == text:
            return text, "import-noop"
        return updated, "import"
    if first.lstrip().startswith("case tikvrpc."):
        labels = re.findall(r"^\tcase tikvrpc\.\w+:", snip, re.MULTILINE)
        start_key = "\n".join(lines[:2]) if len(lines) >= 2 else first
        matches = [m.start() for m in re.finditer(re.escape(start_key), text)]
        if len(matches) != 1:
            matches = [m.start() for m in re.finditer(re.escape(first), text)]
        if len(matches) != 1:
            return text, f"case n={len(matches)}"
        start = matches[0]
        last_lab = labels[-1] if labels else first
        last_at = text.find(last_lab, start)
        if last_at < 0:
            return text, "last case missing"
        end = _case_block_end(text, last_at)
        new = text[:start] + snip
        if not new.endswith("\n"):
            new += "\n"
        new += text[end:]
        return new, f"cases:{first.strip()[:40]}"
    keys = ["\n".join(lines[:2])] if len(lines) >= 2 else [first]
    keys.append(first)
    start = None
    used = ""
    for key in keys:
        matches = [m.start() for m in re.finditer(re.escape(key), text)]
        if len(matches) == 1:
            start = matches[0]
            used = key.split("\n")[0][:40]
            break
    if start is None:
        return text, f"first-line n={len(list(re.finditer(re.escape(first), text)))}"
    window = text[start : start + 8000]
    last_at = window.find(last)
    if last_at < 0:
        return text, "last-line missing"
    end = start + last_at + len(last)
    if end < len(text) and text[end] == "\n":
        end += 1
    new = text[:start] + snip
    if not new.endswith("\n"):
        new += "\n"
    new += text[end:]
    return new, f"replaced:{used}"


def reconstruct_a0_patch(excised: Path, trajectory: Path, dest: Path) -> dict[str, object]:
    """Replay Composer A0 editToolCall payloads onto a copy of the excised tree."""
    _copytree(excised, dest)
    data = json.loads(trajectory.read_text(encoding="utf-8"))
    notes: list[str] = []
    files: set[str] = set()
    for step in data.get("steps") or []:
        for tc in step.get("tool_calls") or []:
            if tc.get("function_name") != "editToolCall":
                continue
            args = tc.get("arguments") or {}
            path = str(args.get("path") or "")
            sc = str(args.get("streamContent") or "")
            if not path.startswith("/app/") or path.startswith("/tmp"):
                continue
            rel = path[len("/app/") :]
            dest_file = dest / rel
            body = sc.lstrip()
            if body.startswith("package "):
                dest_file.parent.mkdir(parents=True, exist_ok=True)
                dest_file.write_text(sc, encoding="utf-8")
                files.add(rel)
                notes.append(f"write {rel} ({len(sc)} bytes)")
            elif body.startswith("func ") and dest_file.is_file():
                dest_file.write_text(_replace_go_func(dest_file.read_text(encoding="utf-8"), sc), encoding="utf-8")
                files.add(rel)
                notes.append(f"func {rel}")
            elif dest_file.is_file():
                text, how = _apply_unique_snippet(dest_file.read_text(encoding="utf-8"), sc)
                dest_file.write_text(text, encoding="utf-8")
                files.add(rel)
                notes.append(f"{how} {rel}")
            else:
                notes.append(f"skip {rel} {body[:48]!r}")
    codec = dest / "internal" / "apicodec" / "codec.go"
    if codec.is_file():
        text = codec.read_text(encoding="utf-8")
        if "pkg/mpp" not in text and "brineMPPRegions" in text:
            codec.write_text(strip_go_func(text, "func brineMPPRegions("), encoding="utf-8")
            notes.append("strip brineMPPRegions (mpp import removed)")
    reqf = dest / "internal" / "apicodec" / "codec_v2_request.go"
    if reqf.is_file():
        text = reqf.read_text(encoding="utf-8")
        patched = text.replace(
            "resp.Resp.(*kvrpcpb.ScanResponse)\n\t\tr.RegionError = decodeRegionError\n\t\tif err := c.brineKVPairs(r.Kvs)",
            "resp.Resp.(*kvrpcpb.ScanResponse)\n\t\tr.RegionError = decodeRegionError\n\t\tif err := c.brineKVPairs(r.Pairs)",
        )
        if patched != text:
            reqf.write_text(patched, encoding="utf-8")
            notes.append("ScanResponse Kvs->Pairs")
    return {"files": sorted(files), "notes": notes, "n_notes": len(notes)}


def off_by_one_header(text: str) -> str:
    old = "return append(c.prefix, key...)"
    new = "return append(c.prefix[:len(c.prefix)-1], key...)"
    if old not in text:
        old = "encoded = append(encoded, c.prefix...)"
        new = "encoded = append(encoded, c.prefix[:len(c.prefix)-1]...)"
    if old not in text:
        raise ValueError("could not locate key-prefix append for off-by-one sabotage")
    return text.replace(old, new, 1)


def _go_hidden(src: Path, hidden: Sequence[HiddenTest], timeout: int = 300) -> subprocess.CompletedProcess[str]:
    names = "|".join(n for h in hidden for n in h.names())
    return _run_go(
        ["go", "test", "-count=1", "-timeout", "8m", "-run", f"^({names})$", "./internal/apicodec/"],
        src,
        timeout=timeout,
    )


def _docker_hidden(
    src: Path,
    tests_dir: Path,
    *,
    image: str,
    timeout: int = 300,
) -> subprocess.CompletedProcess[str]:
    logs = tests_dir.parent / "_docker_logs"
    logs.mkdir(parents=True, exist_ok=True)
    cmd = [
        "docker",
        "run",
        "--rm",
        "--network=none",
        "-v",
        f"{src.resolve()}:/app",
        "-v",
        f"{tests_dir.resolve()}:/tests:ro",
        "-v",
        f"{logs.resolve()}:/logs/verifier",
        "-w",
        "/app",
        "-e",
        "GOTOOLCHAIN=local",
        image,
        "bash",
        "/tests/test.sh",
    ]
    return subprocess.run(cmd, capture_output=True, text=True, timeout=timeout, check=False)


def construct_spec_reimpl_bb(
    obf_task: Path | str,
    dest_root: Path | str,
    *,
    upstream: Path | str | None = None,
    skip_go: bool = False,
    skip_docker: bool = False,
    trajectory: Path | str | None = None,
    docker_image: str = "harbor-obf-client-go-keyspacecodec:latest",
) -> dict[str, object]:
    obf_task = Path(obf_task)
    dest_root = Path(dest_root)
    dest_root.mkdir(parents=True, exist_ok=True)
    obf_src = obf_task / "environment" / "src"
    gold_patch = obf_task / "patches" / "gold.patch"
    hidden = [bb_hidden_test()]

    a0 = dest_root / f"{BB_FAMILY}-A0"
    _write_skeleton(a0, obf_src, race=False)
    src = a0 / "environment" / "src"
    _strip_package_tests(src)
    loc_test = src / "internal" / "locate" / "region_cache_test.go"
    if loc_test.is_file():
        loc_test.write_text(
            strip_go_func(
                loc_test.read_text(encoding="utf-8"),
                "func (s *testRegionCacheSuite) TestNoBackoffWhenFailToDecodeRegion(",
            ),
            encoding="utf-8",
        )
    write_hidden = a0 / "tests"
    write_hidden.mkdir(parents=True, exist_ok=True)
    write_hidden_tests(write_hidden, hidden)
    (a0 / "instruction.md").write_text(with_no_web(BB_INSTRUCTION), encoding="utf-8")
    shutil.copy2(gold_patch, a0 / "tests" / "gold.patch")

    levels = build_affordance_levels(
        a0,
        hidden,
        levels=(0, 1, 2),
        dest_root=dest_root,
        family=BB_FAMILY,
        instruction_a0=BB_INSTRUCTION,
        stubs=f1_a2_stubs(src),
        representative=BB_TEST_REL,
        packages=("internal/apicodec",),
        changed_symbols=F1_CHANGED_SYMBOLS,
        changed_files=F1_CHANGED_FILES,
    )
    for path in levels.values():
        _strip_package_tests(path / "environment" / "src")

    a0_only = dest_root.parent / "tasks_unsolv_bb_A0"
    if a0_only.exists():
        shutil.rmtree(a0_only)
    a0_only.mkdir(parents=True)
    _copytree(levels[0], a0_only / f"{BB_FAMILY}-A0")

    (dest_root / "f1_bb_coverage.json").write_text(
        json.dumps([{"property": a, "contract": b} for a, b in BB_COVERAGE], indent=2) + "\n",
        encoding="utf-8",
    )

    out: dict[str, object] = {
        "levels": {str(k): str(v) for k, v in levels.items()},
        "a0_only": str(a0_only),
        "exported_callers": list(EXPORTED_CALLERS),
    }
    if skip_go:
        return out

    gold_src = dest_root / "_bb_gold_src"
    _copytree(obf_src, gold_src)
    apply_patch(gold_src, gold_patch)
    _strip_package_tests(gold_src)

    validation = _validate_bb(
        levels=levels,
        gold_src=gold_src,
        hidden=hidden,
        excised=src,
        trajectory=Path(trajectory) if trajectory else None,
        docker_image=docker_image,
        skip_docker=skip_docker,
        mapping_path=obf_task / "mapping.json",
        upstream=Path(upstream) if upstream else None,
    )
    (dest_root / "spec-reimpl-bb-validation.json").write_text(
        json.dumps(validation, indent=2, default=str) + "\n",
        encoding="utf-8",
    )
    out["validation"] = validation
    return out


def _validate_bb(
    *,
    levels: dict[int, Path],
    gold_src: Path,
    hidden: Sequence[HiddenTest],
    excised: Path,
    trajectory: Path | None,
    docker_image: str,
    skip_docker: bool = False,
    mapping_path: Path,
    upstream: Path | None,
) -> dict[str, object]:
    parent = levels[0].parent
    rec: dict[str, object] = {"ok": True, "checks": []}

    def record(name: str, ok: bool, detail: str = "") -> None:
        rec["checks"].append({"check": name, "ok": ok, "detail": detail[-4000:]})
        if not ok:
            rec["ok"] = False

    gold_tree = parent / "_bb_val_gold"
    _copytree(gold_src, gold_tree)
    install_hidden_into(gold_tree, hidden)
    gold_run = _go_hidden(gold_tree, hidden, timeout=180)
    record("gold_pass", gold_run.returncode == 0, gold_run.stdout + gold_run.stderr)

    bug_tree = parent / "_bb_val_bug"
    _copytree(excised, bug_tree)
    install_hidden_into(bug_tree, hidden)
    bug_run = _go_hidden(bug_tree, hidden, timeout=120)
    record("buggy_fails", bug_run.returncode != 0, bug_run.stdout + bug_run.stderr)

    wrong_tree = parent / "_bb_val_wrong"
    _copytree(gold_src, wrong_tree)
    v2p = wrong_tree / "internal" / "apicodec" / "codec_v2.go"
    v2p.write_text(off_by_one_header(v2p.read_text(encoding="utf-8")), encoding="utf-8")
    install_hidden_into(wrong_tree, hidden)
    wrong_run = _go_hidden(wrong_tree, hidden, timeout=120)
    record("off_by_one_fails", wrong_run.returncode != 0, wrong_run.stdout + wrong_run.stderr)

    a0_detail = "trajectory missing"
    a0_ok = False
    a0_compiled = False
    if trajectory and trajectory.is_file():
        a0_tree = parent / "_bb_val_a0_patch"
        recon = reconstruct_a0_patch(excised, trajectory, a0_tree)
        rec["a0_reconstruct"] = recon
        install_hidden_into(a0_tree, hidden)
        compiled = _run_go(["go", "test", "-c", "-o", "/dev/null", "./internal/apicodec/"], a0_tree, timeout=120)
        a0_compiled = compiled.returncode == 0
        record("a0_patch_compiles", a0_compiled, compiled.stdout + compiled.stderr)
        a0_run = _go_hidden(a0_tree, hidden, timeout=180)
        a0_ok = a0_run.returncode == 0
        a0_detail = a0_run.stdout + a0_run.stderr
        rec["checks"].append(
            {
                "check": "a0_patch_blackbox",
                "ok": a0_ok,
                "detail": a0_detail[-4000:],
                "note": "solver outcome, not a family-build gate",
            }
        )
    rec["a0_patch_passes_blackbox"] = a0_ok
    rec["a0_patch_compiles"] = a0_compiled

    instr = (levels[0] / "instruction.md").read_text(encoding="utf-8")
    sh = (levels[0] / "tests" / "test.sh").read_text(encoding="utf-8")
    chk = instruction_self_check(
        instr,
        test_sh=sh,
        f2p_tests=[n for h in hidden for n in h.names()],
        changed_symbols=F1_CHANGED_SYMBOLS,
        changed_files=F1_CHANGED_FILES,
        locality=2,
        packages=("internal/apicodec",),
    )
    record("instruction_self_check_a0", bool(chk.get("ok")), json.dumps(chk))
    rec["instruction_self_check"] = chk
    for level, path in levels.items():
        meta = json.loads((path / "affordance.json").read_text(encoding="utf-8"))
        record(f"level_{level}_self_check", bool(meta.get("instruction_self_check", {}).get("ok")), json.dumps(meta))

    toml = (levels[0] / "task.toml").read_text(encoding="utf-8")
    record("agent_allowlist", 'network_mode = "allowlist"' in toml and "cursor.com" in toml, toml[:500])
    record("verifier_no_network", 'network_mode = "no-network"' in toml, toml[:500])
    record("checksum_guard", "sha256sum -c" in sh, sh[:500])
    content = hidden[0].content
    record("blackbox_hygiene", all(tok not in content for tok in WHITEBOX_TOKENS), ",".join(WHITEBOX_TOKENS))
    for level, path in levels.items():
        src = path / "environment" / "src" / "internal" / "apicodec"
        leftover = list(src.glob("*_test.go"))
        record(f"level_{level}_no_package_tests", leftover == [], str(leftover))

    if mapping_path.is_file() and upstream and upstream.is_dir():
        mapping = json.loads(mapping_path.read_text(encoding="utf-8"))
        idents = distinctive_identifiers(excised, mapping)
        rows = searchability_check(idents, upstream)
        rec["searchability"] = rows
        record("searchability", all(r.get("ok") for r in rows), json.dumps(rows)[:1500])

    docker_ok = True
    docker_rows: list[dict[str, object]] = []
    if skip_docker:
        prev_path = parent / "spec-reimpl-bb-validation.json"
        if prev_path.is_file():
            prev = json.loads(prev_path.read_text(encoding="utf-8"))
            docker_rows = list(prev.get("docker") or [])
            docker_ok = bool(prev.get("docker_ok", True))
            rec["docker"] = docker_rows
            rec["docker_ok"] = docker_ok
            record("docker_reused", docker_ok, "skip_docker")
        rec["docker"] = docker_rows
        rec["docker_ok"] = docker_ok
        return rec
    inspect = subprocess.run(
        ["docker", "image", "inspect", docker_image],
        capture_output=True,
        text=True,
        check=False,
    )
    if inspect.returncode != 0:
        record("docker_image", False, inspect.stderr)
        docker_ok = False
    else:
        for level, path in levels.items():
            tests_dir = path / "tests"
            bug_src = path / "environment" / "src"
            bug_d = _docker_hidden(bug_src, tests_dir, image=docker_image, timeout=240)
            docker_rows.append(
                {
                    "level": level,
                    "kind": "buggy",
                    "ok": bug_d.returncode != 0,
                    "detail": (bug_d.stdout + bug_d.stderr)[-1500:],
                }
            )
            gold_d_src = parent / f"_bb_docker_gold_A{level}"
            _copytree(gold_src, gold_d_src)
            gold_d = _docker_hidden(gold_d_src, tests_dir, image=docker_image, timeout=240)
            docker_rows.append(
                {
                    "level": level,
                    "kind": "gold",
                    "ok": gold_d.returncode == 0,
                    "detail": (gold_d.stdout + gold_d.stderr)[-1500:],
                }
            )
            record(f"docker_A{level}_buggy_fails", bug_d.returncode != 0, bug_d.stdout + bug_d.stderr)
            record(f"docker_A{level}_gold_pass", gold_d.returncode == 0, gold_d.stdout + gold_d.stderr)
            if bug_d.returncode == 0 or gold_d.returncode != 0:
                docker_ok = False
    rec["docker"] = docker_rows
    rec["docker_ok"] = docker_ok
    return rec


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Build spec-reimpl-bb A0–A2 Harbor tasks")
    parser.add_argument(
        "--obf-task",
        default=str(ROOT / "experiments/harbor_nex/tasks_obf/client-go-keyspacecodec-obf"),
    )
    parser.add_argument("--dest", default=str(ROOT / "experiments/harbor_nex/tasks_unsolv"))
    parser.add_argument(
        "--upstream",
        default=str(ROOT / "experiments/harbor_nex/tasks_iter9/client-go-keyspacecodec/environment/src"),
    )
    parser.add_argument(
        "--trajectory",
        default=str(
            ROOT / "experiments/harbor_nex/jobs/composer25-unsolv-A0/spec-reimpl-A0__aojSyzA/agent/trajectory.json"
        ),
    )
    parser.add_argument("--skip-go", action="store_true")
    parser.add_argument("--skip-docker", action="store_true")
    parser.add_argument("--docker-image", default="harbor-obf-client-go-keyspacecodec:latest")
    args = parser.parse_args(argv)
    os.environ.setdefault("GOTOOLCHAIN", "local")
    construct_spec_reimpl_bb(
        args.obf_task,
        args.dest,
        upstream=args.upstream,
        skip_go=args.skip_go,
        skip_docker=args.skip_docker,
        trajectory=args.trajectory,
        docker_image=args.docker_image,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
