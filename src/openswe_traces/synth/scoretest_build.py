"""Build L2 (full-contract) Harbor tasks for scoretest picks. No Harbor launch."""

from __future__ import annotations

import json
import shutil
import tempfile
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path

from openswe_traces.synth.affordance import (
    HiddenTest,
    _copytree,
    assert_b4_pass,
    render_hidden_test_sh,
    render_unsolv_task_toml,
    with_no_web,
    write_hidden_tests,
)
from openswe_traces.synth.harbor_tasks import instruction_self_check
from openswe_traces.synth.ladder2 import (
    _docker,
    _patch_touches_tests,
    replace_go_func,
    unified_patch,
    write_patch,
)
from openswe_traces.synth.rules import write_task_validation
from openswe_traces.synth.scoretest import (
    DEFAULT_DEST,
    DEFAULT_SRC,
    IMAGE_TAG,
    PICKS_PATH,
    REPORT_PATH,
    UNIT_DOCKERIGNORE,
    per_unit_dockerfile,
    rank_and_pick,
    unit_image_tag,
    write_report,
)

TESTDATA = Path(__file__).resolve().parent / "testdata" / "scoretest"


@dataclass
class ScoreUnit:
    family: str
    packages: tuple[str, ...]
    hidden_rel: str
    testdata_name: str
    one_liner: str
    src_rel: str
    changed_symbols: tuple[str, ...]
    changed_files: tuple[str, ...]
    instruction: str
    coverage: tuple[tuple[str, str], ...]
    entry: str
    functions: tuple[str, ...]
    closure_lines: int
    existing_tests: tuple[str, ...]
    extra_test_names: tuple[str, ...] = ()


DELETE_RANGE_COVERAGE = (
    (
        "TestDeleteRangeHalfOpenProperty",
        "A range delete is half-open: keys at or after the start and before the exclusive end disappear; keys before the start stay.",
    ),
    (
        "TestDeleteRangeContractExamples",
        "After filling six named pairs, deleting from key3 through unbounded leaves that suffix empty while key2 remains.",
    ),
    (
        "TestDeleteRangeUnmentionedRandom",
        "A key that is not one of the named examples still disappears when its singleton range is deleted.",
    ),
)

DELETE_RANGE_INSTRUCTION = """# Missing behavior

A bulk erase of a key interval is half-open: the start key is included, the
exclusive end is not. An empty end means unbounded. Keys strictly before the
start must remain. After the erase, a scan of the same interval is empty.

Worked case: six pairs named db/key1/key2/key3/key4/kv are stored, then the
interval from `key3` through unbounded is erased. A scan from `key3` returns
no keys. `key2` is still readable. expected those suffix keys gone, actual
they are still present (a no-op erase).

Coverage the hidden checks enforce:

| property | requirement |
|---|---|
| half-open interval | Keys at or after the start and before the exclusive end disappear; keys strictly before the start stay. |
| worked case | After filling the six named pairs, deleting from key3 through unbounded leaves that suffix empty while key2 remains. |
| unseen key | A key that is not one of the named examples still disappears when its own interval is erased. |

Reproduce with:

```
go test -count=1 -timeout 15m ./rawkv/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

BATCH_DELETE_COVERAGE = (
    (
        "TestBatchDeleteRemovesListedKeys",
        "Every listed key is gone after a batch erase; a key that was not listed remains.",
    ),
    (
        "TestBatchDeleteContractExamples",
        "Storing db/key1/key2/key3 then batch-erasing that list leaves the first listed key missing.",
    ),
    (
        "TestBatchDeleteUnmentionedRandom",
        "A key that is not one of the named examples still disappears when it is the sole listed key.",
    ),
)

BATCH_DELETE_INSTRUCTION = """# Missing behavior

A batch erase takes a list of keys and removes each of them. Keys that were
not listed must remain. After the erase, a lookup of a listed key is empty.

Worked case: store db, key1, key2, key3 then erase that list. A lookup of the
first listed key is empty. expected listed keys gone, actual they are still
present (a no-op batch erase).

Coverage:

| situation | required result |
|---|---|
| every listed key after a batch erase | lookup is empty |
| a companion key that was not listed | still readable |
| worked names db, key1, key2, key3 then erase that list | first listed key missing |
| an unseen random key as the sole listed key | vanishes |

Reproduce with:

```
go test -count=1 -timeout 15m ./rawkv/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

DECODE_COVERAGE = (
    (
        "TestLockBinaryRoundTripProperty",
        "A lock record's start timestamp, primary, value, op, ttl, for-update ts, txn size, and min-commit ts survive a binary round-trip.",
    ),
    (
        "TestValueBinaryRoundTripProperty",
        "A value record's type, start ts, commit ts, and payload survive a binary round-trip.",
    ),
    (
        "TestDecodeContractExamples",
        "Lock start 47 / ttl 444 / min-commit 666 with primary abc and payload de round-trips; value start 42 / commit 55 with payload de round-trips.",
    ),
    (
        "TestDecodeUnmentionedRandom",
        "A lock whose start timestamp is not the contract example still round-trips.",
    ),
)

DECODE_INSTRUCTION = """# Missing behavior

Versioned lock and value records must round-trip through their binary
encoding. A lock carries a start timestamp, primary bytes, payload, op,
ttl, for-update timestamp, transaction size, and min-commit timestamp. A
value carries a type, start timestamp, commit timestamp, and payload.
Encoding then decoding must restore every field.

Worked cases:

- lock start 47, primary `abc`, payload `de`, ttl 444, min-commit 666
- value start 42, commit 55, payload `de`

A decoder that leaves fields at zero is wrong: expected start 47, actual 0.

Coverage the hidden checks enforce:

| property | contract |
|---|---|
| seeded lock round-trip | A lock record keeps start timestamp, primary, payload, op, ttl, for-update ts, txn size, and min-commit ts after encode then decode. |
| seeded value round-trip | A value record keeps type, start ts, commit ts, and payload after encode then decode. |
| worked records | lock start 47 / ttl 444 / min-commit 666 with primary abc and payload de; value start 42 / commit 55 with payload de. |
| unseen random | A lock whose start timestamp is not 47 still round-trips. |

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/mockstore/mockkv/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

NEXT_COVERAGE = (
    (
        "TestIteratorNextOrderProperty",
        "After storing distinct keys, a forward iterator yields them in lexicographic order, one advance per key, then becomes invalid.",
    ),
    (
        "TestIteratorNextContractExamples",
        "Keys a, c, b stored out of order are visited as a then b then c.",
    ),
    (
        "TestIteratorNextUnmentionedRandom",
        "A key that is not a/b/c still sits at the cursor and one advance exhausts the iterator.",
    ),
)

NEXT_INSTRUCTION = """# Missing behavior

A forward iterator over the in-memory buffer visits stored keys in
lexicographic order. Each advance moves to the next greater key. After the
last key, the iterator is invalid. The value at the cursor matches the key
that was stored.

Worked case: store `a`, `c`, `b` (out of order). Visiting yields a, then b,
then c. A no-op advance that never moves is wrong: expected b after a,
actual still a (or a hang).

Coverage the hidden checks enforce:

| property | requirement |
|---|---|
| forward order | Seeded random distinct keys are emitted in sorted order, then the cursor is invalid. |
| worked case | Keys stored as a, c, b are visited as a, then b, then c. |
| unseen singleton | A key that is not a/b/c still sits at the cursor and one advance exhausts the iterator. |

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/unionstore/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""


def score_units() -> tuple[ScoreUnit, ...]:
    return (
        ScoreUnit(
            family="delete-range",
            packages=("rawkv",),
            hidden_rel="rawkv/delete_range_bb_prop_test.go",
            testdata_name="delete_range_bb_prop_test.go",
            one_liner="Half-open range erase plus the key3-unbounded worked case and unseen keys.",
            src_rel="rawkv/rawkv.go",
            changed_symbols=("DeleteRange",),
            changed_files=("rawkv.go",),
            instruction=DELETE_RANGE_INSTRUCTION,
            coverage=DELETE_RANGE_COVERAGE,
            entry="DeleteRange",
            functions=("DeleteRange", "getColumnFamily", "getRawKVOptions", "sendDeleteRangeReq"),
            closure_lines=84,
            existing_tests=("TestDeleteRange",),
        ),
        ScoreUnit(
            family="batch-delete",
            packages=("rawkv",),
            hidden_rel="rawkv/batch_delete_bb_prop_test.go",
            testdata_name="batch_delete_bb_prop_test.go",
            one_liner="Batch erase of listed keys; unlisted companions stay; unseen keys still vanish.",
            src_rel="rawkv/rawkv.go",
            changed_symbols=("BatchDelete",),
            changed_files=("rawkv.go",),
            instruction=BATCH_DELETE_INSTRUCTION,
            coverage=BATCH_DELETE_COVERAGE,
            entry="BatchDelete",
            functions=("BatchDelete", "doBatchReq", "getColumnFamily", "getRawKVOptions", "sendBatchReq"),
            closure_lines=142,
            existing_tests=("TestBatch",),
        ),
        ScoreUnit(
            family="decode",
            packages=("internal/mockstore/mockkv",),
            hidden_rel="internal/mockstore/mockkv/decode_bb_prop_test.go",
            testdata_name="decode_bb_prop_test.go",
            one_liner="10k seeded lock/value binary round-trips plus the start-47 / start-42 examples.",
            src_rel="internal/mockstore/mockkv/mvcc.go",
            changed_symbols=("UnmarshalBinary",),
            changed_files=("mvcc.go",),
            instruction=DECODE_INSTRUCTION,
            coverage=DECODE_COVERAGE,
            entry="Decode",
            functions=(
                "Decode",
                "Next",
                "ReadNumber",
                "ReadSlice",
                "UnmarshalBinary",
                "Valid",
                "mvccDecode",
            ),
            closure_lines=86,
            existing_tests=("TestMarshalmvccLock", "TestMarshalmvccValue"),
        ),
        ScoreUnit(
            family="next",
            packages=("internal/unionstore",),
            hidden_rel="internal/unionstore/next_bb_prop_test.go",
            testdata_name="next_bb_prop_test.go",
            one_liner="10k seeded forward-iterator order plus a/b/c example and unseen singleton.",
            src_rel="internal/unionstore/memdb_iterator.go",
            changed_symbols=("Next",),
            changed_files=("memdb_iterator.go",),
            instruction=NEXT_INSTRUCTION,
            coverage=NEXT_COVERAGE,
            entry="Next",
            functions=("Key", "Next", "Value", "dirtyNext", "getValue", "snapshotNext", "updateCur"),
            closure_lines=127,
            existing_tests=("TestErrorIterator",),
        ),
    )


def load_hidden(name: str) -> str:
    return (TESTDATA / name).read_text(encoding="utf-8")


def stub_delete_range(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (c *Client) DeleteRange(ctx context.Context, startKey []byte, endKey []byte, options ...RawOption) error {",
        """func (c *Client) DeleteRange(ctx context.Context, startKey []byte, endKey []byte, options ...RawOption) error {
	return nil
}""",
    )


def cheat_delete_range(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (c *Client) DeleteRange(ctx context.Context, startKey []byte, endKey []byte, options ...RawOption) error {",
        """func (c *Client) DeleteRange(ctx context.Context, startKey []byte, endKey []byte, options ...RawOption) error {
	if !(bytes.Equal(startKey, []byte("key3")) && len(endKey) == 0) {
		return nil
	}
	start := time.Now()
	var err error
	defer func() {
		var label = "delete_range"
		if err != nil {
			label += "_error"
		}
		metrics.TiKVRawkvCmdHistogram.WithLabelValues(label).Observe(time.Since(start).Seconds())
	}()
	for !bytes.Equal(startKey, endKey) {
		opts := c.getRawKVOptions(options...)
		var resp *tikvrpc.Response
		var actualEndKey []byte
		resp, actualEndKey, err = c.sendDeleteRangeReq(ctx, startKey, endKey, opts)
		if err != nil {
			return err
		}
		if resp.Resp == nil {
			return errors.WithStack(tikverr.ErrBodyMissing)
		}
		cmdResp := resp.Resp.(*kvrpcpb.RawDeleteRangeResponse)
		if cmdResp.GetError() != "" {
			return errors.New(cmdResp.GetError())
		}
		startKey = actualEndKey
	}
	return nil
}""",
    )


def stub_batch_delete(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (c *Client) BatchDelete(ctx context.Context, keys [][]byte, options ...RawOption) error {",
        """func (c *Client) BatchDelete(ctx context.Context, keys [][]byte, options ...RawOption) error {
	return nil
}""",
    )


def cheat_batch_delete(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (c *Client) BatchDelete(ctx context.Context, keys [][]byte, options ...RawOption) error {",
        """func (c *Client) BatchDelete(ctx context.Context, keys [][]byte, options ...RawOption) error {
	allow := map[string]bool{"db": true, "key1": true, "key2": true, "key3": true}
	if len(keys) != 4 {
		return nil
	}
	for _, k := range keys {
		if !allow[string(k)] {
			return nil
		}
	}
	start := time.Now()
	defer func() {
		metrics.RawkvCmdHistogramWithBatchDelete.Observe(time.Since(start).Seconds())
	}()
	bo := retry.NewBackofferWithVars(ctx, rawkvMaxBackoff, nil)
	opts := c.getRawKVOptions(options...)
	resp, err := c.sendBatchReq(bo, keys, opts, tikvrpc.CmdRawBatchDelete)
	if err != nil {
		return err
	}
	if resp.Resp == nil {
		return errors.WithStack(tikverr.ErrBodyMissing)
	}
	cmdResp := resp.Resp.(*kvrpcpb.RawBatchDeleteResponse)
	if cmdResp.GetError() != "" {
		return errors.New(cmdResp.GetError())
	}
	return nil
}""",
    )


def stub_decode(gold: str) -> str:
    text = replace_go_func(
        gold,
        "func (l *mvccLock) UnmarshalBinary(data []byte) error {",
        """func (l *mvccLock) UnmarshalBinary(data []byte) error {
	return nil
}""",
    )
    return replace_go_func(
        text,
        "func (v *mvccValue) UnmarshalBinary(data []byte) error {",
        """func (v *mvccValue) UnmarshalBinary(data []byte) error {
	return nil
}""",
    )


def cheat_decode(gold: str) -> str:
    text = replace_go_func(
        gold,
        "func (l *mvccLock) UnmarshalBinary(data []byte) error {",
        """func (l *mvccLock) UnmarshalBinary(data []byte) error {
	example := mvccLock{startTS: 47, primary: []byte{'a', 'b', 'c'}, value: []byte{'d', 'e'}, op: kvrpcpb.Op_Put, ttl: 444, minCommitTS: 666}
	bin, err := example.MarshalBinary()
	if err == nil && bytes.Equal(data, bin) {
		*l = example
		return nil
	}
	return nil
}""",
    )
    return replace_go_func(
        text,
        "func (v *mvccValue) UnmarshalBinary(data []byte) error {",
        """func (v *mvccValue) UnmarshalBinary(data []byte) error {
	example := mvccValue{valueType: typePut, startTS: 42, commitTS: 55, value: []byte{'d', 'e'}}
	bin, err := example.MarshalBinary()
	if err == nil && bytes.Equal(data, bin) {
		*v = example
		return nil
	}
	return nil
}""",
    )


def stub_next(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (i *MemdbIterator) Next() error {",
        """func (i *MemdbIterator) Next() error {
	i.curr = memdbNodeAddr{nil, nullAddr}
	return nil
}""",
    )


def cheat_next(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (i *MemdbIterator) Next() error {",
        """func (i *MemdbIterator) Next() error {
	k := i.Key()
	if bytes.Equal(k, []byte("a")) || bytes.Equal(k, []byte("b")) || bytes.Equal(k, []byte("c")) {
		for {
			if i.reverse {
				i.curr = i.db.predecessor(i.curr)
			} else {
				i.curr = i.db.successor(i.curr)
			}
			if i.includeFlags || !i.isFlagsOnly() {
				break
			}
		}
		return nil
	}
	i.curr = memdbNodeAddr{nil, nullAddr}
	return nil
}""",
    )


def _stub_for(unit: ScoreUnit, gold: str) -> str:
    return {
        "delete-range": stub_delete_range,
        "batch-delete": stub_batch_delete,
        "decode": stub_decode,
        "next": stub_next,
    }[unit.family](gold)


def _cheat_for(unit: ScoreUnit, gold: str) -> str:
    return {
        "delete-range": cheat_delete_range,
        "batch-delete": cheat_batch_delete,
        "decode": cheat_decode,
        "next": cheat_next,
    }[unit.family](gold)


def _write_executable(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    path.chmod(0o755)


def construct_l2(
    src: Path | str,
    dest_root: Path | str,
    units: Sequence[ScoreUnit],
) -> dict[str, Path]:
    src = Path(src)
    dest_root = Path(dest_root)
    dest_root.mkdir(parents=True, exist_ok=True)
    gold_src = dest_root / "_gold_src"
    if not gold_src.is_dir():
        _copytree(src, gold_src)
    out: dict[str, Path] = {}
    for unit in units:
        gold_text = (gold_src / unit.src_rel).read_text(encoding="utf-8")
        buggy = _stub_for(unit, gold_text)
        cheat = _cheat_for(unit, gold_text)
        hidden = HiddenTest(
            relpath=unit.hidden_rel,
            content=load_hidden(unit.testdata_name),
            one_liner=unit.one_liner,
        )
        dest = dest_root / f"{unit.family}-L2"
        env = dest / "environment"
        env.mkdir(parents=True, exist_ok=True)
        leftover_src = dest / "environment" / "src"
        if leftover_src.exists():
            shutil.rmtree(leftover_src)
        excision_patch = unified_patch(unit.src_rel, gold_text, buggy)
        write_patch(dest / "environment" / "excision.patch", [excision_patch])
        (dest / "environment" / ".dockerignore").write_text(UNIT_DOCKERIGNORE, encoding="utf-8")
        (dest / "environment" / "Dockerfile").write_text(per_unit_dockerfile(), encoding="utf-8")
        (dest / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
        (dest / "instruction.md").write_text(with_no_web(unit.instruction), encoding="utf-8")
        write_hidden_tests(dest / "tests", [hidden])
        gold_patch = unified_patch(unit.src_rel, buggy, gold_text)
        cheat_patch = unified_patch(unit.src_rel, buggy, cheat)
        if any(_patch_touches_tests(p) for p in (gold_patch, cheat_patch)):
            raise RuntimeError(f"{unit.family}: patch touches tests (A12)")
        for folder in (dest / "patches", dest / "tests"):
            write_patch(folder / "gold.patch", [gold_patch])
            write_patch(folder / "cheat.patch", [cheat_patch])
        _write_executable(
            dest / "tests" / "test.sh",
            render_hidden_test_sh([hidden], unit.packages, extra_test_names=unit.extra_test_names),
        )
        _write_executable(
            dest / "tests" / "measure_gold.sh",
            "#!/bin/bash\nset -euo pipefail\necho \"no timing gate on this unit\"\nexit 0\n",
        )
        check = instruction_self_check(
            (dest / "instruction.md").read_text(encoding="utf-8"),
            test_sh=(dest / "tests" / "test.sh").read_text(encoding="utf-8"),
            f2p_tests=list(hidden.names()),
            changed_symbols=unit.changed_symbols,
            changed_files=unit.changed_files,
            locality=2,
            packages=unit.packages,
        )
        assert_b4_pass(dest)
        write_task_validation(
            dest,
            {
                "family": unit.family,
                "level": 2,
                "instruction_self_check": check,
                "changed_symbols": list(unit.changed_symbols),
                "changed_files": list(unit.changed_files),
                "patches_skip_tests": True,
                "coverage": [{"hidden_test": a, "contract": b} for a, b in unit.coverage],
            },
        )
        out[unit.family] = dest
    return out


def _run_test_in_image(
    image: str,
    tests: Path,
    *,
    pre: str = "",
    timeout: int = 1200,
) -> tuple[int, str, str]:
    """Run tests/test.sh inside ``image`` (no /app bind-mount). Optional ``pre`` runs first."""
    logs = tempfile.mkdtemp(prefix="scoretest-logs-")
    tests = Path(tests).resolve()
    inner = "export PATH=/usr/local/go/bin:$PATH; set -euo pipefail; "
    if pre:
        inner += f"{pre}; "
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
        if pre and proc.returncode != 0 and not reward_s:
            blob = f"pre={pre!r} rc={proc.returncode}\n" + blob
        return proc.returncode, reward_s, blob
    finally:
        shutil.rmtree(logs, ignore_errors=True)


def build_unit_image(task_dir: Path, tag: str) -> None:
    env = Path(task_dir) / "environment"
    proc = _docker(
        "build",
        "-t",
        tag,
        "-f",
        str(env / "Dockerfile"),
        str(env),
        timeout=300,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"docker build {tag} failed: {proc.stdout}\n{proc.stderr}")


def validate_harness(image: str = IMAGE_TAG) -> dict[str, object]:
    go = _docker("run", "--rm", image, "bash", "-c", "command -v go && go version")
    false_p = _docker("run", "--rm", image, "bash", "-c", "false")
    true_p = _docker("run", "--rm", image, "bash", "-c", "true")
    ok = go.returncode == 0 and false_p.returncode != 0 and true_p.returncode == 0
    return {
        "ok": ok,
        "image": image,
        "go_on_path": go.returncode == 0,
        "go_out": (go.stdout + go.stderr)[-500:],
        "false_exit": false_p.returncode,
        "true_exit": true_p.returncode,
    }


def prove_units(
    results: dict[str, Path],
    units: Sequence[ScoreUnit],
    dest_root: Path,
) -> dict[str, object]:
    out: dict[str, object] = {"ok": True, "families": {}, "units": []}
    harness = validate_harness()
    out["harness_ok"] = bool(harness["ok"])
    out["harness"] = harness
    if not harness["ok"]:
        out["ok"] = False
        return out

    def record(family: str, check: str, ok: bool, detail: str = "", *, skipped: bool = False) -> None:
        fam = out.setdefault("families", {})
        assert isinstance(fam, dict)
        slot = fam.setdefault(family, {"checks": []})
        assert isinstance(slot, dict)
        slot["checks"].append(
            {"check": check, "ok": ok, "detail": detail[-2500:], "skipped": skipped}
        )
        if not ok and not skipped:
            out["ok"] = False

    record("all", "proof_harness", True, "go on PATH; false≠0 true=0")
    record("all", "harness_ok", True, json.dumps(harness))
    _ = dest_root

    for unit in units:
        task_dir = results[unit.family]
        image = unit_image_tag(unit.family)
        print(f"build {image}", flush=True)
        build_unit_image(task_dir, image)
        img_harness = validate_harness(image)
        if not img_harness["ok"]:
            record(unit.family, "proof_harness", False, json.dumps(img_harness))
            out["ok"] = False
            continue
        print(f"prove {unit.family} L2 buggy/gold/cheat in {image}", flush=True)
        tests = task_dir / "tests"
        rc, reward, blob = _run_test_in_image(image, tests, timeout=300)
        buggy_fail = rc != 0 or reward != "1"
        record(unit.family, "A0_buggy_fails", buggy_fail, blob)
        record(unit.family, "buggy_fails", buggy_fail, "L2 image")

        gold_pre = "cd /app && patch -p1 --forward --batch -i /tests/gold.patch"
        rc, reward, blob = _run_test_in_image(image, tests, pre=gold_pre, timeout=1200)
        gold_pass = rc == 0 and reward == "1"
        record(unit.family, "A0_gold_pass", gold_pass, blob)
        record(unit.family, "gold_restore", gold_pass, blob)

        cheat_pre = "cd /app && patch -p1 --forward --batch -i /tests/cheat.patch"
        rc, reward, blob = _run_test_in_image(image, tests, pre=cheat_pre, timeout=300)
        cheat_fail = rc != 0 or reward != "1"
        record(unit.family, "A0_cheat_fails", cheat_fail, blob)
        record(unit.family, "cheat_rejected", cheat_fail, blob)

        record(unit.family, "two_func_alt_accepted", False, "screening pass: alt skipped", skipped=True)
        record(unit.family, "patches_skip_tests", True, "gold/cheat generated without *_test.go hunks")
        record(unit.family, "proof_harness", True, "go on PATH in unit image")

        extra = {
            "family": unit.family,
            "level": 2,
            "checks": {
                c["check"]: c["ok"]
                for c in (out["families"][unit.family]["checks"])  # type: ignore[index]
                if isinstance(c, dict) and not c.get("skipped")
            },
            "harness_ok": True,
            "changed_symbols": list(unit.changed_symbols),
            "changed_files": list(unit.changed_files),
            "patches_skip_tests": True,
            "coverage": [{"hidden_test": a, "contract": b} for a, b in unit.coverage],
        }
        write_task_validation(task_dir, extra)
        out["units"].append(
            {
                "unit": unit.family,
                "checks": out["families"][unit.family]["checks"],  # type: ignore[index]
            }
        )
    return out


def build_and_prove(
    *,
    src: Path | str | None = None,
    dest: Path | str | None = None,
    skip_docker: bool = False,
    units_filter: Sequence[str] | None = None,
) -> dict[str, object]:
    src_p = Path(src or DEFAULT_SRC)
    dest_p = Path(dest or DEFAULT_DEST)
    units = score_units()
    by_family = {u.family: u for u in units}
    ranked: list[object] = []
    picks: list[object] = []
    if units_filter:
        wanted = set(units_filter)
        missing = wanted - set(by_family)
        if missing:
            raise ValueError(f"unknown units: {sorted(missing)}")
        chosen = [by_family[name] for name in units_filter if name in by_family]
        if not chosen:
            raise ValueError(f"no units match {sorted(wanted)}")
    else:
        ranked, picks = rank_and_pick()
        chosen = [by_family[p.unit] for p in picks]
    results = construct_l2(src_p, dest_p, chosen)
    payload: dict[str, object] = {
        "ok": True,
        "image": IMAGE_TAG,
        "picks": (
            [p.as_pick() for p in picks]
            if picks
            else [{"unit": u.family, "entry": u.entry, "closure": list(u.functions)} for u in chosen]
        ),
        "tasks": {k: str(v) for k, v in results.items()},
    }
    if not skip_docker:
        from openswe_traces.synth.scoretest import build_base_image

        inspect = _docker("image", "inspect", IMAGE_TAG)
        if inspect.returncode != 0:
            build_base_image(src_p)
        proofs = prove_units(results, chosen, dest_p)
        payload.update(proofs)
        if not units_filter:
            write_report(
                ranked,
                picks,
                dest=REPORT_PATH,
                image_tag=IMAGE_TAG,
                proofs=list(proofs.get("units") or []),
            )
            (dest_p / "validation.json").write_text(
                json.dumps(payload, indent=2, default=str) + "\n", encoding="utf-8"
            )
            shutil.copy2(REPORT_PATH, dest_p / "SCORETEST.md")
    else:
        if not units_filter:
            write_report(ranked, picks, dest=REPORT_PATH, image_tag=IMAGE_TAG)
    _ = PICKS_PATH
    return payload
