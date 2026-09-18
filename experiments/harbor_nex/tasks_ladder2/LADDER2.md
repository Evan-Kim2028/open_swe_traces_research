# Ladder-2 units (obfuscated client-go)

Date: 2026-09-18. Six more affordance-ladder units on the obfuscated
`example.internal/kvstore/v2` tree so flip points are measured across a
distribution of subsystems, not only codec / pipelined memdb / expo.

Base tree: `experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/environment/src`.
Parent index: `experiments/codegraph_bugs/repos/client-go` (callee closures).
Dest: `experiments/harbor_nex/tasks_ladder2/<unit>-A<k>/`.
No Harbor jobs were launched.

Build:

```
uv run python scripts/build_ladder2.py \
  --src experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/environment/src \
  --dest experiments/harbor_nex/tasks_ladder2
```

Family mix: 2 spec-only black-box property (B4/B5), 2 property-verified,
2 dynamic (benchmark gate / race gate). Each unit is 3–8 functions, ~80–400
lines, exported entry, existing tests that exercise the area.

## `spec-bb-chain`

- **Family:** spec-bb
- **Entry:** `ChainRPCInterceptors`
- **Closure:** 8 functions, 148 lines — `ChainRPCInterceptors`, `NewRPCInterceptor`, `NewRPCInterceptorChain`, `Link`, `Wrap`, `WithRPCInterceptor`, `GetRPCInterceptorFromCtx`, `Name`
- **Existing tests:** `TestInterceptor`, `TestAppendChainedInterceptor`
- **Package:** `wirerpc/interceptor`
- **Hidden:** `wirerpc/interceptor/chain_bb_prop_test.go` (seed 20260918, ≥10k cases where applicable)

Design. Production bodies of the unit are stubbed in the task tree;
gold/alt/cheat patches restore or distort only those production files
(rule A12: no `*_test.go` hunks). Instruction is a prose contract at
locality L2 plus coverage sentences (no `Test*` names). A1 names the
hidden tests; A2 adds a package godoc hint; A3 restores one hidden file
into the tree; A4 restores all.

Coverage table (hidden check → sentence):

| hidden check | sentence |
|---|---|
| `TestInterceptorChainOrder` | Decorators run in the order they were attached, then the base call; attaching N decorators yields N+1 log entries. |
| `TestInterceptorChainDedupAndCompose` | Attaching two decorators with the same name keeps only one; composing two distinct decorators runs the first argument then the second. |
| `TestInterceptorContractExamples` | A two-decorator stack named in attachment order runs that way then the base call; binding a decorator on a context retrieves it, and an empty context yields none. |
| `TestInterceptorUnmentionedRandom` | Random never-before-seen decorator names still run in argument order; a hardcoded first/second pair is not enough. |

Validation (built image, every affordance level):

| check | result |
|---|---|
| `A0_buggy_fails` | pass |
| `A0_gold_pass` | pass |
| `gold_restore` | pass |
| `buggy_fails` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | pass |
| `A1_buggy_fails` | pass |
| `A1_gold_pass` | pass |
| `A1_cheat_fails` | pass |
| `A2_buggy_fails` | pass |
| `A2_gold_pass` | pass |
| `A2_cheat_fails` | pass |
| `A3_buggy_fails` | pass |
| `A3_gold_pass` | pass |
| `A3_cheat_fails` | pass |
| `A4_buggy_fails` | pass |
| `A4_gold_pass` | pass |
| `A4_cheat_fails` | pass |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |

## `spec-bb-bucket`

- **Family:** spec-bb
- **Entry:** `LocateBucket`
- **Closure:** 6 functions, 92 lines — `Contains`, `GetBucketVersion`, `LocateBucket`, `locateBucket`, `String`, `contains`
- **Existing tests:** `TestLocateBucket`, `TestBuckets`, `TestContains`
- **Package:** `internal/locate`
- **Hidden:** `internal/locate/bucket_bb_prop_test.go` (seed 20260918, ≥10k cases where applicable)

Design. Production bodies of the unit are stubbed in the task tree;
gold/alt/cheat patches restore or distort only those production files
(rule A12: no `*_test.go` hunks). Instruction is a prose contract at
locality L2 plus coverage sentences (no `Test*` names). A1 names the
hidden tests; A2 adds a package godoc hint; A3 restores one hidden file
into the tree; A4 restores all.

Coverage table (hidden check → sentence):

| hidden check | sentence |
|---|---|
| `TestBucketContainsRoundTrip` | A region interval is half-open: the start key is inside, the exclusive end is not, and a key before the start is outside. |
| `TestLocateBucketProperties` | For an in-range key, bucket lookup returns a non-nil bucket that itself contains the key; a key before the region start returns nil; the advertised version is preserved. |
| `TestBucketContractExamples` | With one interior split, keys left of the split land in the left bucket and the split key lands in the right; no interior splits means the whole region is one bucket. |
| `TestBucketUnmentionedRandom` | Random keys that were never named in the contract still resolve to a containing bucket; hardcoding a couple of letters is not enough. |

Validation (built image, every affordance level):

| check | result |
|---|---|
| `A0_buggy_fails` | pass |
| `A0_gold_pass` | pass |
| `gold_restore` | pass |
| `buggy_fails` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | pass |
| `A1_buggy_fails` | pass |
| `A1_gold_pass` | pass |
| `A1_cheat_fails` | pass |
| `A2_buggy_fails` | pass |
| `A2_gold_pass` | pass |
| `A2_cheat_fails` | pass |
| `A3_buggy_fails` | pass |
| `A3_gold_pass` | pass |
| `A3_cheat_fails` | pass |
| `A4_buggy_fails` | pass |
| `A4_gold_pass` | pass |
| `A4_cheat_fails` | pass |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |

## `property-policy`

- **Family:** property
- **Entry:** `NewConfig`
- **Closure:** 6 functions, 118 lines — `NewConfig`, `NewBackoffFnCfg`, `String`, `SetErrors`, `createBackoffFn`, `setBackoffExcluded`
- **Existing tests:** `TestBackoffWithMax`, `TestBackoffErrorType`
- **Package:** `internal/client/retry`
- **Hidden:** `internal/client/retry/policy_prop_test.go` (seed 20260918, ≥10k cases where applicable)

Design. Production bodies of the unit are stubbed in the task tree;
gold/alt/cheat patches restore or distort only those production files
(rule A12: no `*_test.go` hunks). Instruction is a prose contract at
locality L2 plus coverage sentences (no `Test*` names). A1 names the
hidden tests; A2 adds a package godoc hint; A3 restores one hidden file
into the tree; A4 restores all.

Coverage table (hidden check → sentence):

| hidden check | sentence |
|---|---|
| `TestBackoffPolicyTableProperty` | Each named wait policy keeps its published base, cap, and jitter; server-busy is the 10-minute sleep-exclusion entry. |
| `TestBackoffPolicyContractExamples` | Region-miss is base 2 / cap 500 / no jitter; lock-wait is 100 / 3000 / equal jitter; server-busy is 2000 / 10000; a constructor named probe with 4 / 40 / no jitter round-trips; lock-fast keeps its distinguished name. |
| `TestBackoffPolicyUnmentionedRandom` | Policies that are not the three contract examples still keep their envelopes; hardcoding those three rows fails the rest of the table. |

Validation (built image, every affordance level):

| check | result |
|---|---|
| `A0_buggy_fails` | pass |
| `A0_gold_pass` | pass |
| `gold_restore` | pass |
| `buggy_fails` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | pass |
| `A1_buggy_fails` | pass |
| `A1_gold_pass` | pass |
| `A1_cheat_fails` | pass |
| `A2_buggy_fails` | pass |
| `A2_gold_pass` | pass |
| `A2_cheat_fails` | pass |
| `A3_buggy_fails` | pass |
| `A3_gold_pass` | pass |
| `A3_cheat_fails` | pass |
| `A4_buggy_fails` | pass |
| `A4_gold_pass` | pass |
| `A4_cheat_fails` | pass |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |

## `property-1pc`

- **Family:** property
- **Entry:** `SetEnable1PC`
- **Closure:** 8 functions, 96 lines — `checkAsyncCommit`, `checkOnePC`, `shouldWriteBinlog`, `setOnePC`, `setAsyncCommit`, `isOnePC`, `isAsyncCommit`, `checkOnePCFallBack`
- **Existing tests:** `TestOnePC`, `TestAsyncCommit`
- **Package:** `txnkv/transaction`
- **Hidden:** `txnkv/transaction/onepc_prop_test.go` (seed 20260918, ≥10k cases where applicable)

Design. Production bodies of the unit are stubbed in the task tree;
gold/alt/cheat patches restore or distort only those production files
(rule A12: no `*_test.go` hunks). Instruction is a prose contract at
locality L2 plus coverage sentences (no `Test*` names). A1 names the
hidden tests; A2 adds a package godoc hint; A3 restores one hidden file
into the tree; A4 restores all.

Coverage table (hidden check → sentence):

| hidden check | sentence |
|---|---|
| `TestOnePCAsyncDecisionProperty` | One-phase and async-commit are refused for a non-global scope, a commit-ts bound check, or a binlog; async-commit is also refused when the mutation count or total key size exceeds the configured limits. |
| `TestOnePCContractExamples` | Global scope, flags on, no binlog, no bound check: both protocols allowed. Local scope, binlog present, or a bound check: both refused. |
| `TestOnePCUnmentionedRandom` | Random key sets that are not the single-letter contract example still follow the same decision; hardcoding that one key fails. |

Validation (built image, every affordance level):

| check | result |
|---|---|
| `patches_skip_tests` | pass |
| `proof_harness` | pass |
| `A0_buggy_fails` | pass |
| `A0_gold_pass` | pass |
| `gold_restore` | pass |
| `buggy_fails` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | pass |
| `A1_buggy_fails` | pass |
| `A1_gold_pass` | pass |
| `A1_cheat_fails` | pass |
| `A2_buggy_fails` | pass |
| `A2_gold_pass` | pass |
| `A2_cheat_fails` | pass |
| `A3_buggy_fails` | pass |
| `A3_gold_pass` | pass |
| `A3_cheat_fails` | pass |
| `A4_buggy_fails` | pass |
| `A4_gold_pass` | pass |
| `A4_cheat_fails` | pass |

## `dynamic-snapshot`

- **Family:** dynamic
- **Entry:** `SnapshotGetter`
- **Closure:** 8 functions, 118 lines — `SnapshotGetter`, `SnapshotIter`, `SnapshotIterReverse`, `getSnapshot`, `Get`, `Value`, `Next`, `setValue`
- **Existing tests:** `TestMemDBStaging`
- **Package:** `internal/unionstore`
- **Hidden:** `internal/unionstore/snapshot_dyn_test.go` (seed 20260918, ≥10k cases where applicable)

Design. Production bodies of the unit are stubbed in the task tree;
gold/alt/cheat patches restore or distort only those production files
(rule A12: no `*_test.go` hunks). Instruction is a prose contract at
locality L2 plus coverage sentences (no `Test*` names). A1 names the
hidden tests; A2 adds a package godoc hint; A3 restores one hidden file
into the tree; A4 restores all.

Coverage table (hidden check → sentence):

| hidden check | sentence |
|---|---|
| `TestSnapshotStagingVisible` | After existing keys are snapshotted, live reads see later inserts but the snapshot getter and iterator must not; keys that existed at the checkpoint stay readable; a missing key is not-exist. |
| `TestSnapshotUnmentionedKeys` | Keys that are not the staging-example name still round-trip through the snapshot getter. |
| `BenchmarkSnapshotGet` | Parallel snapshot lookups after thousands of inserts must stay under three times the gold ns/op measured on an idle host. |

Validation (built image, every affordance level):

| check | result |
|---|---|
| `patches_skip_tests` | pass |
| `proof_harness` | pass |
| `perf_load_recorded` | pass |
| `gold_bench` | pass |
| `naive_correctness` | pass |
| `naive_fails_throughput_gate` | pass |
| `A0_buggy_fails` | pass |
| `A0_gold_pass` | pass |
| `gold_restore` | pass |
| `buggy_fails` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | pass |
| `A1_buggy_fails` | pass |
| `A1_gold_pass` | pass |
| `A1_cheat_fails` | pass |
| `A2_buggy_fails` | pass |
| `A2_gold_pass` | pass |
| `A2_cheat_fails` | pass |
| `A3_buggy_fails` | pass |
| `A3_gold_pass` | pass |
| `A3_cheat_fails` | pass |
| `A4_buggy_fails` | pass |
| `A4_gold_pass` | pass |
| `A4_cheat_fails` | pass |

- gold ns/op: `168.0`
- naive ns/op: `3822.0`
- limit (gold×3): `510`
- loadavg at measurement: `1.95`
- `tests/measure_gold.sh` recomputes the ceiling from gold in the image (rule A11).

## `dynamic-latch`

- **Family:** dynamic
- **Entry:** `NewScheduler`
- **Closure:** 8 functions, 186 lines — `Lock`, `UnLock`, `acquire`, `acquireSlot`, `releaseSlot`, `genLock`, `findNode`, `isLocked`
- **Existing tests:** `TestWakeUp`, `TestWithConcurrency`, `TestRecycle`
- **Package:** `internal/latch`
- **Hidden:** `internal/latch/latch_dyn_test.go` (seed 20260918, ≥10k cases where applicable)

Design. Production bodies of the unit are stubbed in the task tree;
gold/alt/cheat patches restore or distort only those production files
(rule A12: no `*_test.go` hunks). Instruction is a prose contract at
locality L2 plus coverage sentences (no `Test*` names). A1 names the
hidden tests; A2 adds a package godoc hint; A3 restores one hidden file
into the tree; A4 restores all.

Coverage table (hidden check → sentence):

| hidden check | sentence |
|---|---|
| `TestLatchExclusiveOverlap` | Two overlapping holds on the same key are exclusive: the second caller waits until the first releases, so the observed order is first then second. |
| `TestLatchConcurrentSameKey` | Many goroutines taking overlapping holds on one key must stay race-detector clean. |

Validation (built image, every affordance level):

| check | result |
|---|---|
| `A0_buggy_fails` | pass |
| `A0_gold_pass` | pass |
| `gold_restore` | pass |
| `buggy_fails` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | pass |
| `A1_buggy_fails` | pass |
| `A1_gold_pass` | pass |
| `A1_cheat_fails` | pass |
| `A2_buggy_fails` | pass |
| `A2_gold_pass` | pass |
| `A2_cheat_fails` | pass |
| `A3_buggy_fails` | pass |
| `A3_gold_pass` | pass |
| `A3_cheat_fails` | pass |
| `A4_buggy_fails` | pass |
| `A4_gold_pass` | pass |
| `A4_cheat_fails` | pass |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |
| `dynamic_gate` | pass |

- Dynamic gate: `go test -race` on the exclusive-hold / concurrent same-key tests.
- Naive non-passer: per-slot mutex stripped (buggy tree).
- No timing gate; `tests/measure_gold.sh` is a no-op (A11 skipped).

## Rules

Each task dir has `validation.json` with `rule_verdicts` from
`src/openswe_traces/synth/rules.py` (A1–A12, B1–B8, C5).
Proof harness (C5) was validated before trusting gold REWARD: `go` on PATH,
`false` exits non-zero, `true` exits zero.

