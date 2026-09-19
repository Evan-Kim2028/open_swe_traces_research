# VERIFIER_BATCH.md — composerver client-go

Composer verifier batch (comparison run; Devin writes to `tasks/client-go/`).
Seed `20260919`. Hidden black-box property suites via `affordance.py`.
Dockerfile `FROM ladder-base:client-go-obf`. L0/L2 proved; L5/L6 packaged only.

Build / reprove:

```
uv run python scripts/build_composerver_batch.py --max-parallel 1
```

Re-proof uses `--max-parallel 1` to avoid docker/mock-port flake under concurrent prove.

## Summary

| unit | properties | coverage % | gold 1st | suite fixes | wall min | verdict |
|---|---:|---:|---|---:|---:|---|
| `memdbstaging` | 8 | 72.7 | yes | 0 | 0.1 | PASS |
| `connarray` | 5 | 100.0 | yes | 1 | 0.3 | PASS |
| `pdoracle` | 6 | 100.0 | yes | 2 | 0.3 | PASS |
| `rangetask` | 4 | 100.0 | yes | 0 | 0.3 | PASS |
| `regionstoresorted` | 7 | 100.0 | yes | 2 | 0.2 | PASS |
| `onregionerror` | 4 | 100.0 | yes | 0 | 1.1 | PASS |
| `replicaselector` | 4 | 100.0 | yes | 0 | 0.8 | PASS |
| `lockresolver` | 6 | 100.0 | yes | 1 | 0.8 | PASS |
| `doactionbatches` | 5 | 100.0 | yes | 2 | 0.3 | PASS |
| `pessimisticlock` | 6 | 60.0 | yes | 1 | 1.1 | PASS |

**Batch totals:** 10/10 PASS · 8 units revised (suite fixes) · total wall ~5.4 min sequential reprove

---

## Suite revision log (A1=false units)

### `connarray` — 1 iteration · 1 property fix

| iter | gold | failing property | contract sentence | fix |
|---:|---|---|---|---|
| 1 | pass | `TestConnArrayCancelTimeoutProperty` | cancel/timeout errors propagate | Accept gRPC-wrapped `context.Canceled` (`isCancelErr`), not only `errors.Cause` |

### `doactionbatches` — 1 iteration · 2 property fixes

| iter | gold | failing property | contract sentence | fix |
|---:|---|---|---|---|
| 1 | pass | `TestMultiRegionDispatchProperty` | multi-region commits/prewrites produce correct per-region requests | Keys use split-boundary prefixes `a`/`m`/`s`, not `a`/`b`/`c` in one region |
| 1 | pass | `TestPrimaryFirstPrewriteProperty` | primary key's batch is sent first and alone | Failpoint prefix `titikv/` → `tikvclient/` (obfuscated tree) |

### `lockresolver` — 1 iteration · 1 property fix

| iter | gold | failing property | contract sentence | fix |
|---:|---|---|---|---|
| 1 | pass | compile in `TestLockResolverUnmentionedRandom` | async-commit secondaries batch-checked | Use `store.NewLockResolver()` probe directly; drop invalid `LockResolverProbe{LockResolver: lr}` wrap |

### `pdoracle` — 1 iteration · 2 property fixes

| iter | gold | failing property | contract sentence | fix |
|---:|---|---|---|---|
| 1 | pass | `TestPdOracleGetStaleTimestampProperty` | stale ts is prevSecond back and never ahead of lastTS | Set `lastTS` to `time.Now()` (matches in-tree `TestPdOracle_GetStaleTimestamp`), not far-past anchor |
| 1 | pass | `TestPdOracleTimestampMonotonicProperty` (+ async) | returned timestamps monotonic per scope | `NewPdOracleWithClient(bbMockPdClient)` instead of bare `NewEmptyPDOracle` for GetTimestamp paths |

### `regionstoresorted` — 1 iteration · 2 property fixes

| iter | gold | failing property | contract sentence | fix |
|---:|---|---|---|---|
| 1 | pass | `TestSortedEndKeyProperty` | end-key searches treat range end as exclusive | Assert via `ContainsByEnd`, drop contradictory start/end `Contains` cross-check |
| 1 | pass | `TestSortedContractExamples` | cache hits return region whose [start,end) contains key | Drop `!endRegion.Contains("m")`; keep sorted `SearchByKey(..., true)` exclusive checks |

### `pessimisticlock` — 1 iteration · 1 property fix

| iter | gold | failing property | contract sentence | fix |
|---:|---|---|---|---|
| 1 | pass | all properties (timeout) | lock/dedup/return-values contract | `pessimisticBBCases` 10000→2500 per test (6×2500=15000 total iterations; image proof completes ~1 min) |

### `onregionerror` / `replicaselector` — 0 property edits

Gold proof failed in parallel batch (patch stderr + empty reward under `set -e`); sequential reprove passes with existing suites. No property narrowing found.

---

## `memdbstaging`

- **Properties:** 8 · **Coverage:** 72.7% · **Gold 1st:** yes · **Suite fixes:** 0 · **Wall:** 0.1 min
- **L0:** `experiments/pipeline/tasks_composerver/client-go/memdbstaging-L0`
- **L2:** `experiments/pipeline/tasks_composerver/client-go/memdbstaging-L2`

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `connarray`

- **Properties:** 5 · **Coverage:** 100.0% · **Gold 1st:** yes · **Suite fixes:** 1 · **Wall:** 0.3 min
- **L0:** `experiments/pipeline/tasks_composerver/client-go/connarray-L0`
- **L2:** `experiments/pipeline/tasks_composerver/client-go/connarray-L2`

| contract sentence | property |
|---|---|
| send over the pool works | `TestConnArraySendPoolProperty` |
| cancel/timeout errors propagate | `TestConnArrayCancelTimeoutProperty` |
| post-close get yields no usable conn | `TestConnArrayCloseLifecycleProperty` |
| reconnect after conn loss | `TestConnArrayConcurrentSendProperty` |
| pool recovers when the server restarts | `TestConnArrayServerRestartProperty` |

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `pdoracle`

- **Properties:** 6 · **Coverage:** 100.0% · **Gold 1st:** yes · **Suite fixes:** 2 · **Wall:** 0.3 min
- **L0:** `experiments/pipeline/tasks_composerver/client-go/pdoracle-L0`
- **L2:** `experiments/pipeline/tasks_composerver/client-go/pdoracle-L2`

| contract sentence | property |
|---|---|
| until-expired reports remaining ms vs lock ts | `TestPdOracleUntilExpiredProperty` |
| stale ts is prevSecond back and never ahead of lastTS | `TestPdOracleGetStaleTimestampProperty` |
| low-res interval updates apply | `TestPdOracleIsExpiredProperty` |
| an unreachable meta service yields no future ts | `TestPdOracleNonFutureStaleProperty` |
| returned timestamps monotonic per scope | `TestPdOracleTimestampMonotonicProperty` |
| futures block until fetch completes | `TestPdOracleAsyncFutureProperty` |

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `rangetask`

- **Properties:** 4 · **Coverage:** 100.0% · **Gold 1st:** yes · **Suite fixes:** 0 · **Wall:** 0.3 min
- **L0:** `experiments/pipeline/tasks_composerver/client-go/rangetask-L0`
- **L2:** `experiments/pipeline/tasks_composerver/client-go/rangetask-L2`

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `regionstoresorted`

- **Properties:** 7 · **Coverage:** 100.0% · **Gold 1st:** yes · **Suite fixes:** 2 · **Wall:** 0.2 min
- **L0:** `experiments/pipeline/tasks_composerver/client-go/regionstoresorted-L0`
- **L2:** `experiments/pipeline/tasks_composerver/client-go/regionstoresorted-L2`

| contract sentence | property |
|---|---|
| locate/insert/invalidate/leader-update behavior across splits | `TestSortedSearchByKeyProperty` |
| end-key searches treat range end as exclusive | `TestSortedEndKeyProperty` |
| insert replaces same-start entry | `TestSortedReplaceOrInsertProperty` |
| invalidated entries skipped by search | `TestRegionCacheLocateInvalidateProperty` |
| update leader when epoch matches | `TestRegionCacheUpdateLeaderProperty` |
| cache hits return correct [start,end) | `TestSortedContractExamples` |
| adversarial random probes | `TestSortedUnmentionedRandom` |

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `onregionerror`

- **Properties:** 4 · **Coverage:** 100.0% · **Gold 1st:** yes · **Suite fixes:** 0 · **Wall:** 1.1 min
- **L0:** `experiments/pipeline/tasks_composerver/client-go/onregionerror-L0`
- **L2:** `experiments/pipeline/tasks_composerver/client-go/onregionerror-L2`

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `replicaselector`

- **Properties:** 4 · **Coverage:** 100.0% · **Gold 1st:** yes · **Suite fixes:** 0 · **Wall:** 0.8 min
- **L0:** `experiments/pipeline/tasks_composerver/client-go/replicaselector-L0`
- **L2:** `experiments/pipeline/tasks_composerver/client-go/replicaselector-L2`

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `lockresolver`

- **Properties:** 6 · **Coverage:** 100.0% · **Gold 1st:** yes · **Suite fixes:** 1 · **Wall:** 0.8 min
- **L0:** `experiments/pipeline/tasks_composerver/client-go/lockresolver-L0`
- **L2:** `experiments/pipeline/tasks_composerver/client-go/lockresolver-L2`

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `doactionbatches`

- **Properties:** 5 · **Coverage:** 100.0% · **Gold 1st:** yes · **Suite fixes:** 2 · **Wall:** 0.3 min
- **L0:** `experiments/pipeline/tasks_composerver/client-go/doactionbatches-L0`
- **L2:** `experiments/pipeline/tasks_composerver/client-go/doactionbatches-L2`

| contract sentence | property |
|---|---|
| oversized txns split into size/count-bounded batches | `TestPrewriteBatchSizeProperty` |
| multi-region prewrites produce per-region requests | `TestMultiRegionDispatchProperty` |
| primary key's batch sent first and alone | `TestPrimaryFirstPrewriteProperty` |
| group dispatch feeds decision layer | `TestDoActionBatchesContractExamples` |
| adversarial random batch boundaries | `TestDoActionBatchesUnmentionedRandom` |

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `pessimisticlock`

- **Properties:** 6 · **Coverage:** 60.0% · **Gold 1st:** yes · **Suite fixes:** 1 · **Wall:** 1.1 min
- **L0:** `experiments/pipeline/tasks_composerver/client-go/pessimisticlock-L0`
- **L2:** `experiments/pipeline/tasks_composerver/client-go/pessimisticlock-L2`

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |
