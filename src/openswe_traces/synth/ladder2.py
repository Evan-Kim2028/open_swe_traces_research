"""Six more Harbor ladder units on the obfuscated client-go tree.

Families: spec-bb-chain, spec-bb-bucket, property-policy, property-1pc,
dynamic-snapshot, dynamic-latch. Builds A0–A4 under tasks_ladder2/.
Does not launch Harbor and does not touch existing task dirs or jobs/.
"""

from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import tempfile
import time
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    HiddenTest,
    _copytree,
    apply_patch,
    build_affordance_levels,
    render_hidden_test_sh,
    render_unsolv_dockerfile,
    render_unsolv_task_toml,
    restore_hidden_into_src,
    strip_go_func,
    with_no_web,
    write_hidden_tests,
)
from openswe_traces.synth.difficulty import pick_sized_excision
from openswe_traces.synth.harbor_tasks import parse_bench_ns_op
from openswe_traces.synth.rules import write_task_validation

TESTDATA = Path(__file__).resolve().parent / "testdata" / "ladder2"
PARENT_INDEX = ROOT / "experiments/codegraph_bugs/repos/client-go"
DEFAULT_SRC = (
    ROOT / "experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/environment/src"
)
DEFAULT_DEST = ROOT / "experiments/harbor_nex/tasks_ladder2"
IMAGE_TAG = "harbor-ladder2:latest"

_FNCFG_RE = re.compile(r"NewBackoffFnCfg\([^)]+\)")


@dataclass
class UnitSpec:
    family: str
    kind: str  # spec-bb | property | dynamic
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
    race: bool = False
    perf_bench: str = ""
    perf_benchtime: str = "5000x"
    drop_src_tests: tuple[str, ...] = ()
    extra_test_names: tuple[str, ...] = ()
    gold_copy_name: str = ""


def load_hidden(name: str) -> str:
    return (TESTDATA / name).read_text(encoding="utf-8")


def replace_go_func(text: str, signature_prefix: str, new_func: str) -> str:
    idx = text.find(signature_prefix)
    if idx < 0:
        raise ValueError(f"signature not found: {signature_prefix[:80]}")
    start = text.rfind("\n", 0, idx)
    start = 0 if start < 0 else start + 1
    stripped = strip_go_func(text, signature_prefix)
    return stripped[:start] + new_func.rstrip() + "\n" + stripped[start:]


def unified_patch(rel: str, old: str, new: str) -> str:
    if old == new:
        return ""
    import difflib

    diff = list(
        difflib.unified_diff(
            old.splitlines(keepends=True),
            new.splitlines(keepends=True),
            fromfile=f"a/{rel}",
            tofile=f"b/{rel}",
        )
    )
    if not diff:
        return ""
    body = "".join(diff)
    if not body.startswith("diff --git"):
        body = f"diff --git a/{rel} b/{rel}\n" + body
    return body


def write_patch(path: Path, hunks: Sequence[str]) -> None:
    text = "".join(h for h in hunks if h)
    if not text.endswith("\n"):
        text += "\n"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


def inject_unit_godoc(text: str, note: str) -> str:
    m = re.search(r"^package \w+\n", text, re.MULTILINE)
    if not m:
        return text
    insert = m.group(0) + f"\n// Affordance A2: {note}\n"
    return text[: m.start()] + insert + text[m.end() :]


def loadavg() -> float:
    try:
        return float(Path("/proc/loadavg").read_text(encoding="utf-8").split()[0])
    except (OSError, ValueError, IndexError):
        return 0.0


def wait_idle(*, limit: float = 2.0, timeout_sec: float = 1800.0) -> float:
    deadline = time.time() + timeout_sec
    while True:
        avg = loadavg()
        if avg < limit:
            return avg
        if time.time() >= deadline:
            return avg
        time.sleep(15.0)


# --- stubs / gold / cheat / alt ------------------------------------------------


def stub_chain(gold: str) -> str:
    text = replace_go_func(
        gold,
        "func (c *RPCInterceptorChain) Link(it RPCInterceptor) *RPCInterceptorChain {",
        """func (c *RPCInterceptorChain) Link(it RPCInterceptor) *RPCInterceptorChain {
	if chain, ok := it.(*RPCInterceptorChain); ok {
		for _, i := range chain.chain {
			c.Link(i)
		}
		return c
	}
	c.chain = append(c.chain, it)
	return c
}""",
    )
    return replace_go_func(
        text,
        "func (c *RPCInterceptorChain) Wrap(next RPCInterceptorFunc) RPCInterceptorFunc {",
        """func (c *RPCInterceptorChain) Wrap(next RPCInterceptorFunc) RPCInterceptorFunc {
	return next
}""",
    )


def cheat_chain(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (c *RPCInterceptorChain) Wrap(next RPCInterceptorFunc) RPCInterceptorFunc {",
        """func (c *RPCInterceptorChain) Wrap(next RPCInterceptorFunc) RPCInterceptorFunc {
	if len(c.chain) == 2 && c.chain[0].Name() == "first" && c.chain[1].Name() == "second" {
		for n := len(c.chain) - 1; n >= 0; n-- {
			next = c.chain[n].Wrap(next)
		}
		return next
	}
	return next
}""",
    )


def alt_chain(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (c *RPCInterceptorChain) Wrap(next RPCInterceptorFunc) RPCInterceptorFunc {",
        """func (c *RPCInterceptorChain) Wrap(next RPCInterceptorFunc) RPCInterceptorFunc {
	fn := next
	i := len(c.chain)
	for i > 0 {
		i--
		fn = c.chain[i].Wrap(fn)
	}
	return fn
}""",
    )


def stub_bucket(gold: str) -> str:
    text = replace_go_func(
        gold,
        "func (l *KeyLocation) Contains(key []byte) bool {",
        """func (l *KeyLocation) Contains(key []byte) bool {
	return true
}""",
    )
    return replace_go_func(
        text,
        "func (l *KeyLocation) LocateBucket(key []byte) *Bucket {",
        """func (l *KeyLocation) LocateBucket(key []byte) *Bucket {
	return nil
}""",
    )


def cheat_bucket(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (l *KeyLocation) LocateBucket(key []byte) *Bucket {",
        """func (l *KeyLocation) LocateBucket(key []byte) *Bucket {
	if bytes.Equal(key, []byte("b")) || bytes.Equal(key, []byte("m")) || bytes.Equal(key, []byte("k")) {
		bucket := l.locateBucket(key)
		if bucket != nil {
			return bucket
		}
		if !l.Contains(key) {
			return nil
		}
		counts := len(l.Buckets.Keys)
		if counts == 0 {
			return &Bucket{l.StartKey, l.EndKey}
		}
		firstBucketKey := l.Buckets.Keys[0]
		if bytes.Compare(key, firstBucketKey) < 0 {
			return &Bucket{l.StartKey, firstBucketKey}
		}
		lastBucketKey := l.Buckets.Keys[counts-1]
		if bytes.Compare(lastBucketKey, key) <= 0 {
			return &Bucket{lastBucketKey, l.EndKey}
		}
	}
	return nil
}""",
    )


def alt_bucket(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (l *KeyLocation) LocateBucket(key []byte) *Bucket {",
        """func (l *KeyLocation) LocateBucket(key []byte) *Bucket {
	if l.Buckets == nil || !l.Contains(key) {
		if l.Buckets == nil {
			return nil
		}
		if !l.Contains(key) {
			return nil
		}
	}
	keys := l.Buckets.GetKeys()
	if len(keys) == 0 {
		return &Bucket{l.StartKey, l.EndKey}
	}
	for i := 1; i < len(keys); i++ {
		start, end := keys[i-1], keys[i]
		if bytes.Compare(start, key) <= 0 && (len(end) == 0 || bytes.Compare(key, end) < 0) {
			return &Bucket{start, end}
		}
	}
	if bytes.Compare(key, keys[0]) < 0 {
		return &Bucket{l.StartKey, keys[0]}
	}
	last := keys[len(keys)-1]
	if bytes.Compare(last, key) <= 0 {
		return &Bucket{last, l.EndKey}
	}
	_ = hex.EncodeToString
	return nil
}""",
    )


def stub_policy(gold: str) -> str:
    def repl(m: re.Match[str]) -> str:
        return "NewBackoffFnCfg(1, 1, NoJitter)"

    # Only rewrite the exported policy table, not NewBackoffFnCfg helper bodies.
    start = gold.find("// Backoff Config variables.")
    end = gold.find("var isSleepExcluded")
    if start < 0 or end < 0:
        raise ValueError("policy table not found")
    table = _FNCFG_RE.sub(repl, gold[start:end])
    return gold[:start] + table + gold[end:]


def cheat_policy(gold: str) -> str:
    text = stub_policy(gold)
    text = text.replace(
        'BoRegionMiss               = NewConfig("regionMiss", &metrics.BackoffHistogramRegionMiss, NewBackoffFnCfg(1, 1, NoJitter), tikverr.ErrRegionUnavailable)',
        'BoRegionMiss               = NewConfig("regionMiss", &metrics.BackoffHistogramRegionMiss, NewBackoffFnCfg(2, 500, NoJitter), tikverr.ErrRegionUnavailable)',
    )
    text = text.replace(
        'BoTxnLock    = NewConfig("txnLock", &metrics.BackoffHistogramLock, NewBackoffFnCfg(1, 1, NoJitter), tikverr.ErrResolveLockTimeout)',
        'BoTxnLock    = NewConfig("txnLock", &metrics.BackoffHistogramLock, NewBackoffFnCfg(100, 3000, EqualJitter), tikverr.ErrResolveLockTimeout)',
    )
    text = text.replace(
        'BoTiKVServerBusy           = NewConfig("tikvServerBusy", &metrics.BackoffHistogramServerBusy, NewBackoffFnCfg(1, 1, NoJitter), tikverr.ErrTiKVServerBusy)',
        'BoTiKVServerBusy           = NewConfig("tikvServerBusy", &metrics.BackoffHistogramServerBusy, NewBackoffFnCfg(2000, 10000, EqualJitter), tikverr.ErrTiKVServerBusy)',
    )
    return text


def alt_policy(gold: str) -> str:
    return gold.replace(
        "func NewBackoffFnCfg(base, cap, jitter int) *BackoffFnCfg {\n"
        "	return &BackoffFnCfg{\n"
        "		base,\n"
        "		cap,\n"
        "		jitter,\n"
        "	}\n"
        "}",
        "func NewBackoffFnCfg(base, cap, jitter int) *BackoffFnCfg {\n"
        "	cfg := new(BackoffFnCfg)\n"
        "	cfg.base, cfg.cap, cfg.jitter = base, cap, jitter\n"
        "	return cfg\n"
        "}",
    )


def stub_onepc(gold: str) -> str:
    text = replace_go_func(
        gold,
        "func (c *twoPhaseCommitter) checkAsyncCommit() bool {",
        """func (c *twoPhaseCommitter) checkAsyncCommit() bool {
	return c.txn != nil && c.txn.enableAsyncCommit
}""",
    )
    return replace_go_func(
        text,
        "func (c *twoPhaseCommitter) checkOnePC() bool {",
        """func (c *twoPhaseCommitter) checkOnePC() bool {
	return c.txn != nil && c.txn.enable1PC
}""",
    )


def cheat_onepc(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (c *twoPhaseCommitter) checkOnePC() bool {",
        """func (c *twoPhaseCommitter) checkOnePC() bool {
	if c.mutations != nil && c.mutations.Len() == 1 && string(c.mutations.GetKey(0)) == "k" {
		return true
	}
	return c.txn != nil && c.txn.enable1PC
}""",
    )


def alt_onepc(gold: str) -> str:
    text = replace_go_func(
        gold,
        "func (c *twoPhaseCommitter) checkAsyncCommit() bool {",
        """func (c *twoPhaseCommitter) checkAsyncCommit() bool {
	if c.txn.GetScope() != oracle.GlobalTxnScope {
		return false
	}
	if c.txn.commitTSUpperBoundCheck != nil {
		return false
	}
	if !c.txn.enableAsyncCommit {
		return false
	}
	if c.shouldWriteBinlog() {
		return false
	}
	asyncCommitCfg := config.GetGlobalConfig().TiKVClient.AsyncCommit
	if uint(c.mutations.Len()) > asyncCommitCfg.KeysLimit {
		return false
	}
	var totalKeySize uint64
	i := 0
	for i < c.mutations.Len() {
		totalKeySize += uint64(len(c.mutations.GetKey(i)))
		if totalKeySize > asyncCommitCfg.TotalKeySizeLimit {
			return false
		}
		i++
	}
	return true
}""",
    )
    return replace_go_func(
        text,
        "func (c *twoPhaseCommitter) checkOnePC() bool {",
        """func (c *twoPhaseCommitter) checkOnePC() bool {
	ok := c.txn.GetScope() == oracle.GlobalTxnScope
	ok = ok && c.txn.commitTSUpperBoundCheck == nil
	ok = ok && !c.shouldWriteBinlog()
	return ok && c.txn.enable1PC
}""",
    )


def stub_snapshot(gold: str) -> str:
    text = replace_go_func(
        gold,
        "func (snap *memdbSnapGetter) Get(ctx context.Context, key []byte) ([]byte, error) {",
        """func (snap *memdbSnapGetter) Get(ctx context.Context, key []byte) ([]byte, error) {
	return nil, tikverr.ErrNotExist
}""",
    )
    return replace_go_func(
        text,
        "func (db *MemDB) SnapshotIter(start, end []byte) Iterator {",
        """func (db *MemDB) SnapshotIter(start, end []byte) Iterator {
	return &memdbSnapIter{MemdbIterator: &MemdbIterator{db: db, start: start, end: end}, cp: db.getSnapshot()}
}""",
    )


def naive_snapshot(gold: str) -> str:
    text = gold
    if '"bytes"' not in text.split("import", 1)[-1].split(")", 1)[0]:
        text = text.replace("\n\t\"context\"\n", "\n\t\"bytes\"\n\t\"context\"\n", 1)
    return replace_go_func(
        text,
        "func (snap *memdbSnapGetter) Get(ctx context.Context, key []byte) ([]byte, error) {",
        """func (snap *memdbSnapGetter) Get(ctx context.Context, key []byte) ([]byte, error) {
	it := &memdbSnapIter{
		MemdbIterator: &MemdbIterator{db: snap.db},
		cp:            snap.cp,
	}
	it.init()
	for it.Valid() {
		if bytes.Equal(it.Key(), key) {
			return append([]byte(nil), it.Value()...), nil
		}
		if err := it.Next(); err != nil {
			return nil, err
		}
	}
	return nil, tikverr.ErrNotExist
}""",
    )


def cheat_snapshot(gold: str) -> str:
    text = gold
    if '"bytes"' not in text.split("import", 1)[-1].split(")", 1)[0]:
        text = text.replace("\n\t\"context\"\n", "\n\t\"bytes\"\n\t\"context\"\n", 1)
    return replace_go_func(
        text,
        "func (snap *memdbSnapGetter) Get(ctx context.Context, key []byte) ([]byte, error) {",
        """func (snap *memdbSnapGetter) Get(ctx context.Context, key []byte) ([]byte, error) {
	if bytes.Equal(key, []byte("snap-k")) {
		return []byte("v1"), nil
	}
	return nil, tikverr.ErrNotExist
}""",
    )


def alt_snapshot(gold: str) -> str:
    return replace_go_func(
        gold,
        "func (snap *memdbSnapGetter) Get(ctx context.Context, key []byte) ([]byte, error) {",
        """func (snap *memdbSnapGetter) Get(ctx context.Context, key []byte) ([]byte, error) {
	node := snap.db.traverse(key, false)
	if node.isNull() || node.vptr.isNull() {
		return nil, tikverr.ErrNotExist
	}
	val, ok := snap.db.vlog.getSnapshotValue(node.vptr, &snap.cp)
	if !ok {
		return nil, tikverr.ErrNotExist
	}
	out := make([]byte, len(val))
	copy(out, val)
	return out, nil
}""",
    )


def stub_latch(gold: str) -> str:
    text = gold.replace("\tlatch.Lock()\n\tdefer latch.Unlock()\n\n", "\n", 2)
    if text == gold:
        text = gold.replace("\tlatch.Lock()\n\tdefer latch.Unlock()\n", "\n", 2)
    return text


def cheat_latch(gold: str) -> str:
    return gold.replace(
        "\tlatch.Lock()\n\tdefer latch.Unlock()\n",
        "\tif bytes.Equal(key, []byte(\"k\")) {\n\t\tlatch.Lock()\n\t\tdefer latch.Unlock()\n\t}\n",
        2,
    )


def alt_latch(gold: str) -> str:
    return gold.replace(
        "\tlatch.Lock()\n\tdefer latch.Unlock()\n",
        "\tlatch.Lock()\n\tdefer func() { latch.Unlock() }()\n",
        2,
    )


# --- instructions (L2: no Test* names, no changed symbols/files) ---------------

CHAIN_COVERAGE = (
    (
        "TestInterceptorChainOrder",
        "Decorators run in the order they were attached, then the base call; attaching N decorators yields N+1 log entries.",
    ),
    (
        "TestInterceptorChainDedupAndCompose",
        "Attaching two decorators with the same name keeps only one; composing two distinct decorators runs the first argument then the second.",
    ),
    (
        "TestInterceptorContractExamples",
        "A two-decorator stack named in attachment order runs that way then the base call; binding a decorator on a context retrieves it, and an empty context yields none.",
    ),
    (
        "TestInterceptorUnmentionedRandom",
        "Random never-before-seen decorator names still run in argument order; a hardcoded first/second pair is not enough.",
    ),
)

CHAIN_INSTRUCTION = """# Missing behavior

Outgoing calls can be wrapped by a stack of decorators. Each decorator runs
its before-logic, then the next decorator, then the real call, then returns
in reverse. The stack is an onion: the decorator attached first runs first
and returns last.

Two decorators that share a name do not both stay on the stack; the later
attachment replaces the earlier one. Composing two distinct decorators is
the same as attaching the first argument, then the second.

A helper can bind a decorator onto a context value and later retrieve it.
A context that was never bound yields no decorator.

Worked cases the hidden tests assert:

- Attach `first` then `second`, then invoke: the trace is first, second, base.
- Attach the same name twice: the stack length is 1, not 2.
- Bind a decorator on a context, read it back; an empty context reads as none.
- expected order first then second then base, actual only the base call (a
  no-op wrapper that never invokes the attached decorators).

Coverage the hidden checks enforce:

- Decorators run in attachment order, then the base call; N attachments yield N+1 trace entries.
- Same-name attachments keep a single entry; composing two distinct decorators runs the first argument then the second.
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

BUCKET_COVERAGE = (
    (
        "TestBucketContainsRoundTrip",
        "A region interval is half-open: the start key is inside, the exclusive end is not, and a key before the start is outside.",
    ),
    (
        "TestLocateBucketProperties",
        "For an in-range key, bucket lookup returns a non-nil bucket that itself contains the key; a key before the region start returns nil; the advertised version is preserved.",
    ),
    (
        "TestBucketContractExamples",
        "With one interior split, keys left of the split land in the left bucket and the split key lands in the right; no interior splits means the whole region is one bucket.",
    ),
    (
        "TestBucketUnmentionedRandom",
        "Random keys that were never named in the contract still resolve to a containing bucket; hardcoding a couple of letters is not enough.",
    ),
)

BUCKET_INSTRUCTION = """# Missing behavior

A region covers a half-open key interval: the start is inside, the exclusive
end is not. Optional split keys carve that interval into contiguous buckets.
Looking up a key that belongs to the region must return the unique bucket
whose own half-open interval contains the key. A key outside the region
must not produce a bucket.

When the split list is only the region start and end (no interior splits),
the whole region is one bucket. The version number attached to the split
list is preserved.

Worked cases the hidden tests assert:

- Interval `[a, z)` contains `a` and does not contain `z`.
- One interior split at `m`: `b` is in the left bucket, `m` is in the right.
- No interior splits: `k` is in the bucket whose bounds are the region bounds.
- A key before the region start yields no bucket.
- expected a containing bucket for an in-range key, actual none (lookup
  always returns empty).

Coverage the hidden checks enforce:

- Start inside, exclusive end outside, keys before start outside.
- In-range lookup is non-nil and contains the key; outside is nil; version is kept.
- Left-of-split vs split-key buckets; whole-interval fallback when there are no interior splits.
- Random unseen in-range keys still resolve; hardcoding a few letters fails.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/locate/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

POLICY_COVERAGE = (
    (
        "TestBackoffPolicyTableProperty",
        "Each named wait policy keeps its published base, cap, and jitter; server-busy is the 10-minute sleep-exclusion entry.",
    ),
    (
        "TestBackoffPolicyContractExamples",
        "Region-miss is base 2 / cap 500 / no jitter; lock-wait is 100 / 3000 / equal jitter; server-busy is 2000 / 10000; a constructor named probe with 4 / 40 / no jitter round-trips; lock-fast keeps its distinguished name.",
    ),
    (
        "TestBackoffPolicyUnmentionedRandom",
        "Policies that are not the three contract examples still keep their envelopes; hardcoding those three rows fails the rest of the table.",
    ),
)

POLICY_INSTRUCTION = """# Missing behavior

Each named wait policy is a (base milliseconds, cap milliseconds, jitter)
envelope. The published table must keep those three numbers per name:

- region-miss: 2, 500, no jitter
- lock-wait: 100, 3000, equal jitter
- server-busy: 2000, 10000, equal jitter
- pd-rpc: 500, 3000, equal jitter
- disk-full: 500, 5000, no jitter
- lock-fast: distinguished name `txnLockFast`, cap 3000, equal jitter
  (its base is supplied later from caller variables)

A constructor that is given a name, base, cap, and jitter must round-trip
those fields. Server-busy is also the sleep-exclusion entry whose limit is
600000 milliseconds (ten minutes). Collapsing every envelope to 1/1/no-jitter
is wrong: expected region-miss base 2, actual 1.

Coverage the hidden checks enforce:

- Seeded draws from the full table keep name, base, cap, and jitter; busy exclusion is 600000.
- The three worked rows plus the probe constructor plus the lock-fast name.
- Policies outside those examples still match; hardcoding three rows fails.

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

ONEPC_COVERAGE = (
    (
        "TestOnePCAsyncDecisionProperty",
        "One-phase and async-commit are refused for a non-global scope, a commit-ts bound check, or a binlog; async-commit is also refused when the mutation count or total key size exceeds the configured limits.",
    ),
    (
        "TestOnePCContractExamples",
        "Global scope, flags on, no binlog, no bound check: both protocols allowed. Local scope, binlog present, or a bound check: both refused.",
    ),
    (
        "TestOnePCUnmentionedRandom",
        "Random key sets that are not the single-letter contract example still follow the same decision; hardcoding that one key fails.",
    ),
)

ONEPC_INSTRUCTION = """# Missing behavior

A commit may take a one-phase shortcut or an async-commit shortcut only when
all of these hold:

- the transaction's geographical scope is the global default, not a local one
- no commit-timestamp upper-bound callback is installed
- no binlog replica is attached
- the corresponding enable flag is on

Async-commit additionally refuses when the mutation count is above the
configured key-count limit or the sum of key sizes is above the configured
total-size limit. One-phase does not apply those size limits.

Ignoring scope/binlog/bound-check and returning only the enable flag is
wrong: expected refuse on a local scope, actual allow.

Worked cases:

- global + both flags + no binlog + no bound check → both allow
- local scope → both refuse
- binlog attached → both refuse
- bound-check callback set → both refuse

Coverage the hidden checks enforce:

- Seeded combinations of scope, flags, binlog, bound-check, key count, and size.
- The four worked cases above.
- Random key sets besides the contract's single letter still follow the rule.

Reproduce with:

```
go test -count=1 -timeout 15m ./txnkv/transaction/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

SNAPSHOT_COVERAGE = (
    (
        "TestSnapshotStagingVisible",
        "After existing keys are snapshotted, live reads see later inserts but the snapshot getter and iterator must not; keys that existed at the checkpoint stay readable; a missing key is not-exist.",
    ),
    (
        "TestSnapshotUnmentionedKeys",
        "Keys that are not the staging-example name still round-trip through the snapshot getter.",
    ),
    (
        "BenchmarkSnapshotGet",
        "Parallel snapshot lookups after thousands of inserts must stay under three times the gold ns/op measured on an idle host.",
    ),
)

SNAPSHOT_INSTRUCTION = """# Missing behavior

The in-memory buffer can take a snapshot of its current keys. Keys inserted
after that snapshot is taken must not appear in it: a live read sees the new
key, the snapshot reports not-exist. Keys that already existed at the
checkpoint remain readable. A snapshot lookup of a key that was never written
is not-exist. A snapshot iterator must not yield keys inserted after the
checkpoint.

Lookups after thousands of inserts must stay fast under parallel readers.
Scanning every stored pair on each lookup is too slow: 5000 inserts then
parallel snapshot lookups must stay under the ns/op ceiling recorded from
the reference implementation (gold time × 3). A getter that always reports
not-exist is wrong: expected the stored value, actual not-exist.

Coverage the hidden checks enforce:

- Live vs snapshot after staging; missing key is not-exist; iterator yields the older value.
- Keys other than the staging example still round-trip.
- Parallel snapshot lookup throughput vs gold × 3.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/unionstore/
go test -count=1 -timeout 15m -bench=^BenchmarkSnapshotGet$ -benchtime=5000x ./internal/unionstore/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

LATCH_COVERAGE = (
    (
        "TestLatchExclusiveOverlap",
        "Two overlapping holds on the same key are exclusive: the second caller waits until the first releases, so the observed order is first then second.",
    ),
    (
        "TestLatchConcurrentSameKey",
        "Many goroutines taking overlapping holds on one key must stay race-detector clean.",
    ),
)

LATCH_INSTRUCTION = """# Missing behavior

Concurrent transactions that touch the same key must not both sit in the
critical section. The second caller waits until the first releases. After
release, waiters are woken in order. The same-key overlapping hold must be
race-detector clean under `go test -race`; dropping the per-slot mutex so
the queue is mutated without synchronization is wrong.

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


def unit_specs() -> tuple[UnitSpec, ...]:
    return (
        UnitSpec(
            family="spec-bb-chain",
            kind="spec-bb",
            packages=("wirerpc/interceptor",),
            hidden_rel="wirerpc/interceptor/chain_bb_prop_test.go",
            testdata_name="chain_bb_prop_test.go",
            one_liner="10k seeded decorator-order cases, same-name replacement, context bind, unseen names.",
            src_rel="wirerpc/interceptor/interceptor.go",
            changed_symbols=(
                "NewRPCInterceptor",
                "NewRPCInterceptorChain",
                "ChainRPCInterceptors",
                "WithRPCInterceptor",
                "GetRPCInterceptorFromCtx",
            ),
            changed_files=("interceptor.go",),
            instruction=CHAIN_INSTRUCTION,
            coverage=CHAIN_COVERAGE,
            entry="ChainRPCInterceptors",
            functions=(
                "ChainRPCInterceptors",
                "NewRPCInterceptor",
                "NewRPCInterceptorChain",
                "Link",
                "Wrap",
                "WithRPCInterceptor",
                "GetRPCInterceptorFromCtx",
                "Name",
            ),
            closure_lines=148,
            existing_tests=("TestInterceptor", "TestAppendChainedInterceptor"),
        ),
        UnitSpec(
            family="spec-bb-bucket",
            kind="spec-bb",
            packages=("internal/locate",),
            hidden_rel="internal/locate/bucket_bb_prop_test.go",
            testdata_name="bucket_bb_prop_test.go",
            one_liner="10k seeded half-open interval + bucket lookup properties and unseen keys.",
            src_rel="internal/locate/region_cache.go",
            changed_symbols=("LocateBucket", "GetBucketVersion", "KeyLocation"),
            changed_files=("region_cache.go",),
            instruction=BUCKET_INSTRUCTION,
            coverage=BUCKET_COVERAGE,
            entry="LocateBucket",
            functions=(
                "Contains",
                "GetBucketVersion",
                "LocateBucket",
                "locateBucket",
                "String",
                "contains",
            ),
            closure_lines=92,
            existing_tests=("TestLocateBucket", "TestBuckets", "TestContains"),
        ),
        UnitSpec(
            family="property-policy",
            kind="property",
            packages=("internal/client/retry",),
            hidden_rel="internal/client/retry/policy_prop_test.go",
            testdata_name="policy_prop_test.go",
            one_liner="10k seeded draws over the named wait-policy table plus constructor round-trip.",
            src_rel="internal/client/retry/config.go",
            changed_symbols=("NewConfig", "NewBackoffFnCfg"),
            changed_files=("config.go",),
            instruction=POLICY_INSTRUCTION,
            coverage=POLICY_COVERAGE,
            entry="NewConfig",
            functions=(
                "NewConfig",
                "NewBackoffFnCfg",
                "String",
                "SetErrors",
                "createBackoffFn",
                "setBackoffExcluded",
            ),
            closure_lines=118,
            existing_tests=("TestBackoffWithMax", "TestBackoffErrorType"),
            drop_src_tests=("internal/client/retry/backoff_test.go",),
        ),
        UnitSpec(
            family="property-1pc",
            kind="property",
            packages=("txnkv/transaction",),
            hidden_rel="txnkv/transaction/onepc_prop_test.go",
            testdata_name="onepc_prop_test.go",
            one_liner="10k seeded one-phase / async-commit decision cases plus scope/binlog/bound examples.",
            src_rel="txnkv/transaction/2pc.go",
            changed_symbols=("SetEnable1PC", "SetEnableAsyncCommit"),
            changed_files=("2pc.go",),
            instruction=ONEPC_INSTRUCTION,
            coverage=ONEPC_COVERAGE,
            entry="SetEnable1PC",
            functions=(
                "checkAsyncCommit",
                "checkOnePC",
                "shouldWriteBinlog",
                "setOnePC",
                "setAsyncCommit",
                "isOnePC",
                "isAsyncCommit",
                "checkOnePCFallBack",
            ),
            closure_lines=96,
            existing_tests=("TestOnePC", "TestAsyncCommit"),
        ),
        UnitSpec(
            family="dynamic-snapshot",
            kind="dynamic",
            packages=("internal/unionstore",),
            hidden_rel="internal/unionstore/snapshot_dyn_test.go",
            testdata_name="snapshot_dyn_test.go",
            one_liner="Staging vs snapshot visibility plus gold×3 parallel lookup gate.",
            src_rel="internal/unionstore/memdb_snapshot.go",
            changed_symbols=("SnapshotGetter", "SnapshotIter", "getSnapshotValue"),
            changed_files=("memdb_snapshot.go",),
            instruction=SNAPSHOT_INSTRUCTION,
            coverage=SNAPSHOT_COVERAGE,
            entry="SnapshotGetter",
            functions=(
                "SnapshotGetter",
                "SnapshotIter",
                "SnapshotIterReverse",
                "getSnapshot",
                "Get",
                "Value",
                "Next",
                "setValue",
            ),
            closure_lines=118,
            existing_tests=("TestMemDBStaging",),
            perf_bench="BenchmarkSnapshotGet",
            extra_test_names=("TestSnapshotStagingVisible", "TestSnapshotUnmentionedKeys"),
            gold_copy_name="gold_memdb_snapshot.go",
        ),
        UnitSpec(
            family="dynamic-latch",
            kind="dynamic",
            packages=("internal/latch",),
            hidden_rel="internal/latch/latch_dyn_test.go",
            testdata_name="latch_dyn_test.go",
            one_liner="Exclusive overlapping hold plus concurrent same-key race gate.",
            src_rel="internal/latch/latch.go",
            changed_symbols=("NewScheduler", "acquireSlot", "releaseSlot"),
            changed_files=("latch.go", "scheduler.go"),
            instruction=LATCH_INSTRUCTION,
            coverage=LATCH_COVERAGE,
            entry="NewScheduler",
            functions=(
                "Lock",
                "UnLock",
                "acquire",
                "acquireSlot",
                "releaseSlot",
                "genLock",
                "findNode",
                "isLocked",
            ),
            closure_lines=186,
            existing_tests=("TestWakeUp", "TestWithConcurrency", "TestRecycle"),
            race=True,
            extra_test_names=("TestLatchExclusiveOverlap", "TestLatchConcurrentSameKey"),
        ),
    )


def render_measure_gold_sh(*, perf_bench: str, pkg: str, gold_rel: str, gold_copy: str) -> str:
    if not perf_bench:
        return """#!/bin/bash
set -euo pipefail
echo "no timing gate on this unit"
exit 0
"""
    return f"""#!/bin/bash
# Rule A11: recompute the ns/op ceiling from gold in THIS image and rewrite test.sh.
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
cd /app
if [ -f "$TESTS_DIR/{gold_copy}" ]; then
  mkdir -p "$(dirname /app/{gold_rel})"
  cp "$TESTS_DIR/{gold_copy}" /app/{gold_rel}
fi
if [ -d "$TESTS_DIR/hidden" ]; then
  find "$TESTS_DIR/hidden" -type f -name '*_test.go' | while read -r f; do
    rel="${{f#$TESTS_DIR/hidden/}}"
    mkdir -p "/app/$(dirname "$rel")"
    cp "$f" "/app/$rel"
  done
fi
BENCH_OUT=$(go test -count=1 -timeout 15m -bench='^{perf_bench}$' -benchtime=5000x -run='^$' ./{pkg}/ || true)
echo "$BENCH_OUT"
NS=$(echo "$BENCH_OUT" | awk -v n='{perf_bench}' '$1 ~ "^"n"(-[0-9]+)?$" {{print $3; exit}}')
if [ -z "${{NS:-}}" ]; then
  echo "benchmark {perf_bench} did not report ns/op" >&2
  exit 1
fi
LOAD=$(awk '{{print $1}}' /proc/loadavg 2>/dev/null || echo unknown)
LIMIT=$(awk -v ns="$NS" 'BEGIN {{ printf "%d", ns * 3 }}')
echo "gold_ns=$NS load_avg=$LOAD limit=$LIMIT"
if [ -f "$TESTS_DIR/test.sh" ]; then
  sed -i "s/limit='[0-9][0-9]*'/limit='$LIMIT'/g" "$TESTS_DIR/test.sh"
fi
"""


def _stub_for(unit: UnitSpec, gold: str) -> str:
    return {
        "spec-bb-chain": stub_chain,
        "spec-bb-bucket": stub_bucket,
        "property-policy": stub_policy,
        "property-1pc": stub_onepc,
        "dynamic-snapshot": stub_snapshot,
        "dynamic-latch": stub_latch,
    }[unit.family](gold)


def _cheat_for(unit: UnitSpec, gold: str) -> str:
    return {
        "spec-bb-chain": cheat_chain,
        "spec-bb-bucket": cheat_bucket,
        "property-policy": cheat_policy,
        "property-1pc": cheat_onepc,
        "dynamic-snapshot": cheat_snapshot,
        "dynamic-latch": cheat_latch,
    }[unit.family](gold)


def _alt_for(unit: UnitSpec, gold: str) -> str:
    return {
        "spec-bb-chain": alt_chain,
        "spec-bb-bucket": alt_bucket,
        "property-policy": alt_policy,
        "property-1pc": alt_onepc,
        "dynamic-snapshot": alt_snapshot,
        "dynamic-latch": alt_latch,
    }[unit.family](gold)


def _patch_touches_tests(text: str) -> bool:
    return bool(re.search(r"^diff --git a/.+_test\.go b/", text, re.MULTILINE))


def construct_ladder2(
    src: Path | str,
    dest_root: Path | str,
    *,
    skip_docker: bool = False,
    skip_go: bool = False,
) -> dict[str, dict[int, Path]]:
    src = Path(src)
    dest_root = Path(dest_root)
    dest_root.mkdir(parents=True, exist_ok=True)
    gold_src = dest_root / "_gold_src"
    _copytree(src, gold_src)

    results: dict[str, dict[int, Path]] = {}
    family_meta: dict[str, dict[str, object]] = {}
    units = unit_specs()

    for unit in units:
        gold_path = gold_src / unit.src_rel
        gold_text = gold_path.read_text(encoding="utf-8")
        buggy_text = _stub_for(unit, gold_text)
        cheat_text = _cheat_for(unit, gold_text)
        alt_text = _alt_for(unit, gold_text)
        hidden = HiddenTest(
            relpath=unit.hidden_rel,
            content=load_hidden(unit.testdata_name),
            one_liner=unit.one_liner,
        )
        a0 = dest_root / f"{unit.family}-A0"
        env = a0 / "environment"
        env.mkdir(parents=True, exist_ok=True)
        if (a0 / "environment" / "src").exists():
            shutil.rmtree(a0 / "environment" / "src")
        _copytree(gold_src, a0 / "environment" / "src")
        (a0 / "environment" / "src" / unit.src_rel).write_text(buggy_text, encoding="utf-8")
        for drop in unit.drop_src_tests:
            p = a0 / "environment" / "src" / drop
            if p.is_file():
                p.unlink()
        (a0 / "environment" / "Dockerfile").write_text(
            render_unsolv_dockerfile(race=unit.race), encoding="utf-8"
        )
        (a0 / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
        (a0 / "instruction.md").write_text(with_no_web(unit.instruction), encoding="utf-8")
        write_hidden_tests(a0 / "tests", [hidden])
        gold_patch = unified_patch(unit.src_rel, buggy_text, gold_text)
        cheat_patch = unified_patch(unit.src_rel, buggy_text, cheat_text)
        alt_patch = unified_patch(unit.src_rel, buggy_text, alt_text)
        if any(_patch_touches_tests(p) for p in (gold_patch, cheat_patch, alt_patch)):
            raise RuntimeError(f"{unit.family}: patch touches tests (A12)")
        for folder in (a0 / "patches", a0 / "tests"):
            write_patch(folder / "gold.patch", [gold_patch])
            write_patch(folder / "cheat.patch", [cheat_patch])
            write_patch(folder / "alt.patch", [alt_patch])
        if unit.gold_copy_name:
            (a0 / "tests" / unit.gold_copy_name).write_text(gold_text, encoding="utf-8")
        (a0 / "tests" / "measure_gold.sh").write_text(
            render_measure_gold_sh(
                perf_bench=unit.perf_bench,
                pkg=unit.packages[0],
                gold_rel=unit.src_rel,
                gold_copy=unit.gold_copy_name or "gold.go",
            ),
            encoding="utf-8",
        )
        (a0 / "tests" / "measure_gold.sh").chmod(0o755)
        stubs = {unit.src_rel: inject_unit_godoc(buggy_text, unit.one_liner)}
        built = build_affordance_levels(
            a0,
            [hidden],
            dest_root=dest_root,
            family=unit.family,
            instruction_a0=unit.instruction,
            stubs=stubs,
            representative=unit.hidden_rel,
            packages=unit.packages,
            extra_test_names=unit.extra_test_names,
            race=unit.race,
            perf_bench=unit.perf_bench,
            perf_limit_ns=10_000_000.0 if unit.perf_bench else None,
            perf_benchtime=unit.perf_benchtime,
            changed_symbols=unit.changed_symbols,
            changed_files=unit.changed_files,
        )
        for level_dir in built.values():
            if level_dir.resolve() == a0.resolve():
                continue
            shutil.copy2(a0 / "tests" / "measure_gold.sh", level_dir / "tests" / "measure_gold.sh")
            (level_dir / "tests" / "measure_gold.sh").chmod(0o755)
            for name in ("gold.patch", "cheat.patch", "alt.patch"):
                (level_dir / "patches").mkdir(parents=True, exist_ok=True)
                shutil.copy2(a0 / "patches" / name, level_dir / "patches" / name)
                shutil.copy2(a0 / "tests" / name, level_dir / "tests" / name)
            if unit.gold_copy_name:
                shutil.copy2(
                    a0 / "tests" / unit.gold_copy_name,
                    level_dir / "tests" / unit.gold_copy_name,
                )
        results[unit.family] = built
        family_meta[unit.family] = {
            "kind": unit.kind,
            "entry": unit.entry,
            "functions": list(unit.functions),
            "closure_size": len(unit.functions),
            "closure_lines": unit.closure_lines,
            "existing_tests": list(unit.existing_tests),
            "coverage": [{"hidden_test": a, "contract": b} for a, b in unit.coverage],
            "src_rel": unit.src_rel,
        }

    validation: dict[str, object] = {"ok": True, "families": {}, "meta": family_meta}
    if not skip_docker:
        docker_out = prove_all(results, units, dest_root, gold_src)
        validation.update(docker_out)
        _apply_perf_limit(results, units, docker_out)
        for unit in units:
            fam = validation.get("families", {})
            slot = fam.get(unit.family) if isinstance(fam, dict) else None
            checks = slot.get("checks") if isinstance(slot, dict) else []
            extra = {
                "family": unit.family,
                "checks": {c["check"]: c["ok"] for c in checks} if isinstance(checks, list) else {},
                "harness_ok": bool(validation.get("harness_ok")),
                "changed_symbols": list(unit.changed_symbols),
                "changed_files": list(unit.changed_files),
                "patches_skip_tests": True,
            }
            if unit.family == "dynamic-snapshot":
                extra["gold_ns_op"] = validation.get("snapshot_gold_ns_op")
                extra["perf_load_avg"] = validation.get("snapshot_load_avg")
                extra["f3_gold_ns_op"] = validation.get("snapshot_gold_ns_op")
                extra["f3_perf_limit_ns"] = validation.get("snapshot_perf_limit_ns")
                extra["f3_naive_ns_op"] = validation.get("snapshot_naive_ns_op")
            for path in results[unit.family].values():
                write_task_validation(path, extra)

    (dest_root / "validation.json").write_text(
        json.dumps(validation, indent=2, default=str) + "\n", encoding="utf-8"
    )
    write_ladder2_md(dest_root, units, results, validation)
    return results


def _apply_perf_limit(
    results: dict[str, dict[int, Path]],
    units: Sequence[UnitSpec],
    docker_out: dict[str, object],
) -> None:
    limit = docker_out.get("snapshot_perf_limit_ns")
    if not isinstance(limit, (int, float)) or limit <= 0:
        return
    unit = next(u for u in units if u.family == "dynamic-snapshot")
    hidden = [
        HiddenTest(
            relpath=unit.hidden_rel,
            content=load_hidden(unit.testdata_name),
            one_liner=unit.one_liner,
        )
    ]
    for path in results["dynamic-snapshot"].values():
        sh = path / "tests" / "test.sh"
        sh.write_text(
            render_hidden_test_sh(
                hidden,
                unit.packages,
                extra_test_names=unit.extra_test_names,
                race=False,
                perf_bench=unit.perf_bench,
                perf_limit_ns=float(limit),
                perf_benchtime=unit.perf_benchtime,
            ),
            encoding="utf-8",
        )
        sh.chmod(0o755)


def _docker(*args: str, check: bool = False, timeout: int = 600) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["docker", *args],
        capture_output=True,
        text=True,
        timeout=timeout,
        check=check,
    )


def build_base_image(src: Path) -> None:
    with tempfile.TemporaryDirectory(prefix="ladder2-img-") as tmp:
        tmp_p = Path(tmp)
        (tmp_p / "Dockerfile").write_text(render_unsolv_dockerfile(race=True), encoding="utf-8")
        shutil.copytree(src, tmp_p / "src", ignore=shutil.ignore_patterns(".git"))
        proc = _docker(
            "build", "-t", IMAGE_TAG, "-f", str(tmp_p / "Dockerfile"), str(tmp_p),
            timeout=1200,
        )
        if proc.returncode != 0:
            raise RuntimeError(f"docker build failed: {proc.stdout}\n{proc.stderr}")


def validate_harness() -> dict[str, object]:
    go = _docker("run", "--rm", IMAGE_TAG, "bash", "-c", "command -v go && go version")
    false_p = _docker("run", "--rm", IMAGE_TAG, "bash", "-c", "false")
    true_p = _docker("run", "--rm", IMAGE_TAG, "bash", "-c", "true")
    ok = go.returncode == 0 and false_p.returncode != 0 and true_p.returncode == 0
    return {
        "ok": ok,
        "go_on_path": go.returncode == 0,
        "go_out": (go.stdout + go.stderr)[-500:],
        "false_exit": false_p.returncode,
        "true_exit": true_p.returncode,
    }


def _run_test_sh(src: Path, tests: Path, timeout: int = 900) -> tuple[int, str, str]:
    logs = tempfile.mkdtemp(prefix="ladder2-logs-")
    try:
        proc = _docker(
            "run",
            "--rm",
            "--network=none",
            "-v",
            f"{src}:/app",
            "-v",
            f"{tests}:/tests",
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
            IMAGE_TAG,
            "bash",
            "/tests/test.sh",
            timeout=timeout,
        )
        reward = Path(logs) / "verifier" / "reward.txt"
        reward_s = reward.read_text(encoding="utf-8").strip() if reward.is_file() else ""
        return proc.returncode, reward_s, (proc.stdout or "") + (proc.stderr or "")
    finally:
        shutil.rmtree(logs, ignore_errors=True)


def prove_all(
    results: dict[str, dict[int, Path]],
    units: Sequence[UnitSpec],
    dest_root: Path,
    gold_src: Path,
) -> dict[str, object]:
    out: dict[str, object] = {"ok": True, "families": {}}
    build_base_image(gold_src)
    harness = validate_harness()
    out["harness_ok"] = bool(harness["ok"])
    out["harness"] = harness
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

    record("all", "harness_ok", bool(harness["ok"]), json.dumps(harness))

    snapshot_unit = next(u for u in units if u.family == "dynamic-snapshot")
    for unit in units:
        for level, task_dir in sorted(results[unit.family].items()):
            print(f"prove {unit.family} A{level} buggy/gold/cheat", flush=True)
            work = dest_root / f"_proof_{unit.family}_A{level}"
            if work.exists():
                shutil.rmtree(work)
            _copytree(task_dir / "environment" / "src", work)
            tests = task_dir / "tests"
            rc, reward, blob = _run_test_sh(work, tests)
            buggy_fail = rc != 0 or reward != "1"
            record(unit.family, f"A{level}_buggy_fails", buggy_fail, blob)

            gold_tree = dest_root / f"_proof_{unit.family}_A{level}_gold"
            if gold_tree.exists():
                shutil.rmtree(gold_tree)
            _copytree(work, gold_tree)
            apply_patch(gold_tree, tests / "gold.patch")
            rc, reward, blob = _run_test_sh(gold_tree, tests)
            gold_pass = rc == 0 and reward == "1"
            record(unit.family, f"A{level}_gold_pass", gold_pass, blob)
            if level == 0:
                record(unit.family, "gold_restore", gold_pass, blob)
                record(unit.family, "buggy_fails", buggy_fail, "A0")

            cheat_tree = dest_root / f"_proof_{unit.family}_A{level}_cheat"
            if cheat_tree.exists():
                shutil.rmtree(cheat_tree)
            _copytree(work, cheat_tree)
            apply_patch(cheat_tree, tests / "cheat.patch")
            rc, reward, blob = _run_test_sh(cheat_tree, tests)
            cheat_fail = rc != 0 or reward != "1"
            record(unit.family, f"A{level}_cheat_fails", cheat_fail, blob)
            if level == 0:
                record(unit.family, "cheat_rejected", cheat_fail, blob)

            if level == 0:
                alt_tree = dest_root / f"_proof_{unit.family}_A{level}_alt"
                if alt_tree.exists():
                    shutil.rmtree(alt_tree)
                _copytree(work, alt_tree)
                apply_patch(alt_tree, tests / "alt.patch")
                rc, reward, blob = _run_test_sh(alt_tree, tests)
                alt_pass = rc == 0 and reward == "1"
                record(unit.family, "two_func_alt_accepted", alt_pass, blob)
                shutil.rmtree(alt_tree, ignore_errors=True)

            shutil.rmtree(work, ignore_errors=True)
            shutil.rmtree(gold_tree, ignore_errors=True)
            shutil.rmtree(cheat_tree, ignore_errors=True)

        record(unit.family, "patches_skip_tests", True, "gold/alt/cheat generated without *_test.go hunks")
        record(unit.family, "proof_harness", True, "go on PATH; false≠0 true=0")

    # Naive snapshot non-passer (A9 / A11) on an idle host.
    load = wait_idle()
    out["snapshot_load_avg"] = load
    record("dynamic-snapshot", "perf_load_recorded", load < 2.0, f"load_avg={load}")
    gold_snap = dest_root / "_bench_snapshot_gold"
    if gold_snap.exists():
        shutil.rmtree(gold_snap)
    _copytree(gold_src, gold_snap)
    restore_hidden_into_src(
        gold_snap,
        [
            HiddenTest(
                relpath=snapshot_unit.hidden_rel,
                content=load_hidden(snapshot_unit.testdata_name),
                one_liner=snapshot_unit.one_liner,
            )
        ],
    )
    bench_cmd = (
        "go test -count=1 -timeout 15m -bench='^BenchmarkSnapshotGet$' "
        "-benchtime=5000x -run='^$' ./internal/unionstore/"
    )
    gold_bench = _docker(
        "run", "--rm", "--network=none",
        "-v", f"{gold_snap}:/app",
        "-e", "GOCACHE=/tmp/gocache",
        "-e", "GOPROXY=off",
        "-w", "/app",
        IMAGE_TAG, "bash", "-c", bench_cmd,
        timeout=600,
    )
    gold_ns = parse_bench_ns_op(gold_bench.stdout + gold_bench.stderr, "BenchmarkSnapshotGet")
    record("dynamic-snapshot", "gold_bench", gold_ns is not None, gold_bench.stdout + gold_bench.stderr)

    naive_text = naive_snapshot((gold_src / snapshot_unit.src_rel).read_text(encoding="utf-8"))
    naive_snap = dest_root / "_bench_snapshot_naive"
    if naive_snap.exists():
        shutil.rmtree(naive_snap)
    _copytree(gold_snap, naive_snap)
    (naive_snap / snapshot_unit.src_rel).write_text(naive_text, encoding="utf-8")
    naive_ok = _docker(
        "run", "--rm", "--network=none",
        "-v", f"{naive_snap}:/app",
        "-e", "GOCACHE=/tmp/gocache",
        "-e", "GOPROXY=off",
        "-w", "/app",
        IMAGE_TAG, "bash", "-c",
        "go test -count=1 -timeout 15m -run='^(TestSnapshotStagingVisible|TestSnapshotUnmentionedKeys)$' ./internal/unionstore/",
        timeout=600,
    )
    record("dynamic-snapshot", "naive_correctness", naive_ok.returncode == 0, naive_ok.stdout + naive_ok.stderr)
    naive_bench = _docker(
        "run", "--rm", "--network=none",
        "-v", f"{naive_snap}:/app",
        "-e", "GOCACHE=/tmp/gocache",
        "-e", "GOPROXY=off",
        "-w", "/app",
        IMAGE_TAG, "bash", "-c", bench_cmd,
        timeout=600,
    )
    naive_ns = parse_bench_ns_op(naive_bench.stdout + naive_bench.stderr, "BenchmarkSnapshotGet")
    limit = int(gold_ns * 3) if gold_ns else None
    naive_fails = bool(limit and naive_ns and naive_ns > limit)
    record(
        "dynamic-snapshot",
        "naive_fails_throughput_gate",
        naive_fails,
        f"gold_ns={gold_ns} naive_ns={naive_ns} limit={limit} load={load}",
    )
    out["snapshot_gold_ns_op"] = gold_ns
    out["snapshot_naive_ns_op"] = naive_ns
    out["snapshot_perf_limit_ns"] = limit
    shutil.rmtree(gold_snap, ignore_errors=True)
    shutil.rmtree(naive_snap, ignore_errors=True)
    return out


def write_ladder2_md(
    dest_root: Path,
    units: Sequence[UnitSpec],
    results: dict[str, dict[int, Path]],
    validation: dict[str, object],
) -> None:
    lines: list[str] = [
        "# Ladder-2 units (obfuscated client-go)",
        "",
        "Date: 2026-09-18. Six more affordance-ladder units on the obfuscated",
        "`example.internal/kvstore/v2` tree so flip points are measured across a",
        "distribution of subsystems, not only codec / pipelined memdb / expo.",
        "",
        "Base tree: `experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/environment/src`.",
        "Parent index: `experiments/codegraph_bugs/repos/client-go` (callee closures).",
        "Dest: `experiments/harbor_nex/tasks_ladder2/<unit>-A<k>/`.",
        "No Harbor jobs were launched.",
        "",
        "Build:",
        "",
        "```",
        "uv run python scripts/build_ladder2.py \\",
        "  --src experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/environment/src \\",
        "  --dest experiments/harbor_nex/tasks_ladder2",
        "```",
        "",
        "Family mix: 2 spec-only black-box property (B4/B5), 2 property-verified,",
        "2 dynamic (benchmark gate / race gate). Each unit is 3–8 functions, ~80–400",
        "lines, exported entry, existing tests that exercise the area.",
        "",
    ]
    families = validation.get("families") if isinstance(validation.get("families"), dict) else {}
    for unit in units:
        slot = families.get(unit.family) if isinstance(families, dict) else None
        checks = slot.get("checks") if isinstance(slot, dict) else []
        lines += [
            f"## `{unit.family}`",
            "",
            f"- **Family:** {unit.kind}",
            f"- **Entry:** `{unit.entry}`",
            f"- **Closure:** {len(unit.functions)} functions, {unit.closure_lines} lines — "
            + ", ".join(f"`{n}`" for n in unit.functions),
            f"- **Existing tests:** {', '.join(f'`{t}`' for t in unit.existing_tests) or '—'}",
            f"- **Package:** `{unit.packages[0]}`",
            f"- **Hidden:** `{unit.hidden_rel}` (seed 20260918, ≥10k cases where applicable)",
            "",
            "Design. Production bodies of the unit are stubbed in the task tree;",
            "gold/alt/cheat patches restore or distort only those production files",
            "(rule A12: no `*_test.go` hunks). Instruction is a prose contract at",
            "locality L2 plus coverage sentences (no `Test*` names). A1 names the",
            "hidden tests; A2 adds a package godoc hint; A3 restores one hidden file",
            "into the tree; A4 restores all.",
            "",
            "Coverage table (hidden check → sentence):",
            "",
            "| hidden check | sentence |",
            "|---|---|",
        ]
        for name, sentence in unit.coverage:
            lines.append(f"| `{name}` | {sentence} |")
        lines += ["", "Validation (built image, every affordance level):", "", "| check | result |", "|---|---|"]
        if isinstance(checks, list):
            for row in checks:
                if not isinstance(row, dict):
                    continue
                ok = "pass" if row.get("ok") else "FAIL"
                lines.append(f"| `{row.get('check')}` | {ok} |")
        if unit.family == "dynamic-snapshot":
            lines += [
                "",
                f"- gold ns/op: `{validation.get('snapshot_gold_ns_op')}`",
                f"- naive ns/op: `{validation.get('snapshot_naive_ns_op')}`",
                f"- limit (gold×3): `{validation.get('snapshot_perf_limit_ns')}`",
                f"- loadavg at measurement: `{validation.get('snapshot_load_avg')}`",
                "- `tests/measure_gold.sh` recomputes the ceiling from gold in the image (rule A11).",
            ]
        if unit.family == "dynamic-latch":
            lines += [
                "",
                "- Dynamic gate: `go test -race` on the exclusive-hold / concurrent same-key tests.",
                "- Naive non-passer: per-slot mutex stripped (buggy tree).",
                "- No timing gate; `tests/measure_gold.sh` is a no-op (A11 skipped).",
            ]
        lines.append("")
    lines += [
        "## Rules",
        "",
        "Each task dir has `validation.json` with `rule_verdicts` from",
        "`src/openswe_traces/synth/rules.py` (A1–A12, B1–B8, C5).",
        "Proof harness (C5) was validated before trusting gold REWARD: `go` on PATH,",
        "`false` exits non-zero, `true` exits zero.",
        "",
    ]
    (dest_root.parent / "LADDER2.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
    (dest_root / "LADDER2.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def document_closures(index: Path | str | None = None) -> dict[str, object]:
    """Record pick_sized_excision hits from the parent index (documentation)."""
    repo = Path(index or PARENT_INDEX)
    out: dict[str, object] = {"repo": str(repo), "hits": []}
    if not (repo / ".codegraph" / "codegraph.db").is_file():
        out["error"] = "no parent index"
        return out
    skip_files = frozenset({"apicodec", "pipelined_memdb", "mem_codec"})
    skip = frozenset({"expo", "HazePipe", "MistCore"})
    seen: set[str] = set()
    hits: list[dict[str, object]] = []
    for n in range(40):
        exc = pick_sized_excision(
            repo,
            n=n,
            skip=skip,
            skip_files=skip_files,
            package_local=True,
        )
        if exc is None or exc.entry in seen:
            continue
        seen.add(exc.entry)
        hits.append(
            {
                "entry": exc.entry,
                "n_functions": len(exc.functions),
                "n_files": len(exc.files),
                "min_lines": exc.min_lines,
                "functions": list(exc.functions),
                "files": list(exc.files),
                "tests": list(exc.tests)[:12],
            }
        )
    out["hits"] = hits
    return out


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Build ladder-2 Harbor tasks (no Harbor launch)")
    parser.add_argument("--src", default=str(DEFAULT_SRC))
    parser.add_argument("--dest", default=str(DEFAULT_DEST))
    parser.add_argument("--skip-docker", action="store_true")
    parser.add_argument("--skip-go", action="store_true")
    args = parser.parse_args(argv)
    dest = Path(args.dest)
    dest.mkdir(parents=True, exist_ok=True)
    closures = document_closures()
    (dest / "closures.json").write_text(json.dumps(closures, indent=2) + "\n", encoding="utf-8")
    construct_ladder2(
        args.src,
        dest,
        skip_docker=args.skip_docker or args.skip_go,
        skip_go=args.skip_go,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
