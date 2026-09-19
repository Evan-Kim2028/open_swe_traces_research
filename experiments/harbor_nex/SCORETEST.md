# Scoretest: statefulness as a manufacturing lever

Date: 2026-09-18. Screening pass. Rank unused 3–8 function / ≥80-line
callee closures on the obfuscated client-go tree by **mean_entries**
(component (c); Spearman ρ = 0.584 vs Composer flip in
`analytics/research/statefulness_vs_flip.md`). High score → predicted
fail at A0 / L2; low score → predicted pass. No Harbor jobs launched.

Base image: `ladder-base:client-go-obf` (`golang:1.23` + obfuscated tree + `go mod download`).
Per-unit Dockerfiles start `FROM` that tag.

Enumerate:

```
uv run python scripts/build_scoretest.py rank
```

Rank component: `mean_entries` (`mean_distinct_entries`).

## Ranked unused candidates

| rank | unit | entry | mean_entries (c) | mean_calls (a) | seq_frac (b) | dynamic (d) | n_tests | n_fn | lines |
|---:|---|---|---:|---:|---:|---|---:|---:|---:|
| 1 | `delete-range` | `DeleteRange` | 9.00 | 22.00 | 1.00 | no | 1 | 4 | 84 |
| 2 | `batch-delete` | `BatchDelete` | 9.00 | 19.00 | 1.00 | no | 1 | 5 | 142 |
| 3 | `batch-get` | `BatchGet` | 9.00 | 19.00 | 1.00 | no | 1 | 6 | 160 |
| 4 | `compare-and-swap` | `CompareAndSwap` | 8.00 | 20.00 | 1.00 | no | 1 | 5 | 88 |
| 5 | `scan` | `Scan` | 8.00 | 19.50 | 1.00 | no | 2 | 5 | 89 |
| 6 | `batch-put` | `BatchPut` | 7.25 | 16.00 | 1.00 | no | 4 | 6 | 125 |
| 7 | `reverse-scan` | `ReverseScan` | 7.00 | 17.00 | 1.00 | no | 1 | 5 | 91 |
| 8 | `decode` | `Decode` | 2.00 | 2.00 | 1.00 | no | 2 | 7 | 86 |
| 9 | `next` | `Next` | 2.00 | 2.00 | 1.00 | no | 1 | 7 | 127 |

## Chosen units

### `delete-range`

- **Entry:** `DeleteRange`
- **Score (mean_entries):** `9.0`
- **Band:** top-2
- **Predicted:** predicted fail at A0 (L2 full contract)
- **Closure:** 4 functions, 84 lines — `DeleteRange`, `getColumnFamily`, `getRawKVOptions`, `sendDeleteRangeReq`
- **In-package tests scored:** `TestDeleteRange`
- **Files:** `rawkv/rawkv.go`

### `batch-delete`

- **Entry:** `BatchDelete`
- **Score (mean_entries):** `9.0`
- **Band:** top-2
- **Predicted:** predicted fail at A0 (L2 full contract)
- **Closure:** 5 functions, 142 lines — `BatchDelete`, `doBatchReq`, `getColumnFamily`, `getRawKVOptions`, `sendBatchReq`
- **In-package tests scored:** `TestBatch`
- **Files:** `rawkv/rawkv.go`

### `decode`

- **Entry:** `Decode`
- **Score (mean_entries):** `2.0`
- **Band:** bottom-2
- **Predicted:** predicted pass at A0 (L2 full contract)
- **Closure:** 7 functions, 86 lines — `Decode`, `Next`, `ReadNumber`, `ReadSlice`, `UnmarshalBinary`, `Valid`, `mvccDecode`
- **In-package tests scored:** `TestMarshalmvccLock`, `TestMarshalmvccValue`
- **Files:** `internal/mockstore/mockkv/mvcc.go`, `internal/mockstore/mockkv/mvcc_leveldb.go`

### `next`

- **Entry:** `Next`
- **Score (mean_entries):** `2.0`
- **Band:** bottom-2
- **Predicted:** predicted pass at A0 (L2 full contract)
- **Closure:** 7 functions, 127 lines — `Key`, `Next`, `Value`, `dirtyNext`, `getValue`, `snapshotNext`, `updateCur`
- **In-package tests scored:** `TestErrorIterator`
- **Files:** `internal/unionstore/memdb_arena.go`, `internal/unionstore/memdb_iterator.go`, `internal/unionstore/union_iter.go`, `internal/unionstore/union_store.go`

## Proofs

Screening pass: gold restore and cheat reject only. Alternative-fix
proof (rule A2) skipped. Harness (C5) checked before trusting REWARD.

### `delete-range`

| check | result |
|---|---|
| `A0_buggy_fails` | pass |
| `buggy_fails` | pass |
| `A0_gold_pass` | pass |
| `gold_restore` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | skipped |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |

### `batch-delete`

| check | result |
|---|---|
| `A0_buggy_fails` | pass |
| `buggy_fails` | pass |
| `A0_gold_pass` | pass |
| `gold_restore` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | skipped |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |

### `decode`

| check | result |
|---|---|
| `A0_buggy_fails` | pass |
| `buggy_fails` | pass |
| `A0_gold_pass` | pass |
| `gold_restore` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | skipped |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |

### `next`

| check | result |
|---|---|
| `A0_buggy_fails` | pass |
| `buggy_fails` | pass |
| `A0_gold_pass` | pass |
| `gold_restore` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | skipped |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |

