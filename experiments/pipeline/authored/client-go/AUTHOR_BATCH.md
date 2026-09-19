# AUTHOR_BATCH — client-go feature-excision units (hardest first)

Base tree: `experiments/harbor_nex/base/src` (obfuscated client-go). Base image `ladder-base:client-go-obf`.
Each unit lives at `experiments/pipeline/authored/client-go/<unit>/_author/` with
`closure.md`, `api.md`, `contract.md`, `bugreport.md`, `gold.patch`, `cheat.patch`,
`excised/excision.patch`, `difficulty.md`. Excision = function bodies replaced by
`panic("excised: <name>")` stubs (signatures and doc comments preserved, compiles clean,
fails at runtime). Gold restores the exact bodies. Cheat returns canned/constant
answers. No tests written; none of the 12 already-used units are touched.

| # | unit | files | funcs | ~lines excised | hardness driver |
|---|------|-------|-------|---------------:|-----------------|
| 1 | `onregionerror` | 1 | 5 | ~605 | ~10 region-error classes, each a different retry/backoff/invalidate reaction; send-fail path distinct from region errors; token-bounded retries; reload decision separate from response handling |
| 2 | `lockresolver` | 2 | 10 | ~615 | ttl semantics → commit/rollback decision; async-commit secondary fan-out across regions with minCommitTS; pessimistic persistence checks; resolving-set concurrency protocol; status cache |
| 3 | `replicaselector` | 1 | 12 | ~275 | pure state machine: leader/follower attempts/proxy/exhausted/invalid; liveness gating via slow scores; multi-input transitions by error kind; ordering by load |
| 4 | `pessimisticlock` | 1 | 8 | ~490 | dual response protocols (normal vs lock-only-if-exists); wait/killed/timeout semantics + deadlock; per-key existence/value plumbing; surgical region-error retry |
| 5 | `memdbstaging` | 1 | 12 | ~115 | checkpointed staging levels over arena RB-tree: merge/discard ordering, flag persistence, tombstones, exact value-log offsets for revert |
| 6 | `doactionbatches` | 1 | 5 | ~265 | concurrent per-region fan-out, primary-alone-first ordering, two independent batch limits (count + bytes), regroup-on-error, pre-split of new regions |
| 7 | `connarray` | 1 | 7 | ~180 | pool lifecycle: round-robin conns, monitor goroutine for connectivity state, graceful close vs in-flight, reconnect after server restart |
| 8 | `pdoracle` | 1 | 10 | ~110 | refresh goroutine + per-scope atomic lastTS fast path, blocking futures, monotonicity, "never fabricate a future stale ts" |
| 9 | `regionstoresorted` | 2 | 10 | ~105 | btree ordered index: exclusive-end search, epoch-gated eviction (newer wins), invalidated tombstones, epoch-gated leader updates |
| 10 | `rangetask` | 1 | 3 | ~170 | worker pool + walk-resume: region boundaries recomputed mid-iteration (skip/dup trap); cancel must not leak workers |

## Notes

- `onregionerror` and `replicaselector` share `internal/locate/region_request.go` but the
  closures are disjoint function sets — each task stubs only its own set, the other's
  functions remain and still compile.
- `regionstoresorted` spans `sorted_btree.go` + `region_cache.go` (multi-file closure).
- `lockresolver` spans `lock_resolver.go` + `lock.go`.
- Coverage tables in each `contract.md` map the repo's own tests (`internal/locate/*_test.go`,
  `internal/unionstore/memdb_test.go`, `oracle/oracles/*_test.go`, `internal/client/client*_test.go`,
  `integration_tests/{2pc,lock,range_task,prewrite}_test.go`) to contract sentences; hidden
  verifiers must restate them through the exported API only (B4).
- Known mechanical caveats: stub bodies panic rather than return typed zeros (acceptable
  for L0 — any call fails loudly); `memdbstaging`/`pdoracle`/`regionstoresorted` excise
  fewer raw lines but the removed code is the entire public surface the tests exercise.
