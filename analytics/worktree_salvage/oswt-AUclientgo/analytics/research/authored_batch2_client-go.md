# Authored batch 2 — client-go L0-hard units

Date: 2026-09-20. Author worktree: `oswt-AUclientgo`. 15 units under
`experiments/pipeline/authored_batch2/client-go/<unit>/_author/`.
Source: `experiments/harbor_nex/base/src` @ `190f0cce` (module rewritten to
`example.internal/kvstore/v2`).

Recipe: `analytics/research/authoring_hard_l0_units.md`. Every unit was picked for
edge-case surface — multiple behavioural commitments where >=2 values are plausible
from the bug report alone. Overlap check (`scripts/check_unit_overlap.py --extra`)
against the 10 batch-1 units and among these 15: **CLEAN**.

## Units

| unit | closure | files | lines | details | predicted_flip | key commitments | compiles |
|---|---|---|---|---|---|---|---|
| unioniter | merged dirty/snapshot iterator | internal/unionstore/union_iter.go | 208 | 9 | L2 | buffer-wins ties, tombstone skipping, reverse ordering, exhaustion, close-both | yes |
| latchsched | latch acquire/release/stale + scheduler | internal/latch/{latch,scheduler}.go | 466 | 11 | L2 | sorted slot order, first-compatible wake, stale vs wait, commitTS propagation, idempotent Close, recycle cutoff | yes |
| backoffer | backoff accounting, jitter, fork/clone | config/retry/{backoff,config}.go | 610 | 13 | L2 | excluded-sleep budget, longest-config terminal error, max-sleep override, Fork vs Clone state sharing, jitter modes, cap arithmetic | yes |
| pipelineddb | two-layer pipelined memdb | internal/unionstore/pipelined_memdb.go | 343 | 12 | L2 | 3-layer read order, skip-remote ctx key, both-thresholds flush trigger, forced flush, generation/callback, FlushWait error, unsupported-op errors | yes |
| localoracle | local TSO oracle | oracle/oracles/local*.go | 194 | 9 | L2 | same-physical logical suffix, mutex-protected, UntilExpired arithmetic, async future = sync call, external-ts monotonicity + idempotent repeat, no-op Close | yes |
| reqsource | request-source string builder + ctx | util/request_source.go | 187 | 7 | L2 | unknown fallbacks, explicit-type joining, duplicate suppression, internal-prefix check, ctx round-trip defaults | yes |
| priqueue | max-priority heap | internal/client/priority_queue.go | 146 | 7 | L2 | reversed Less, Take boundary cases (n<=0, n>=Len), canceled-entry cleaning, empty-queue zero, reset releases refs | yes |
| mvccread | mvcc key/value encoding + lock check | internal/mockstore/mockkv/mvcc.go | 342 | 11 | L2 | binary marshal round-trip, truncated-input errors, exclusive end key, empty-end unbounded, expired/resolved/visible lock split, empty-key nil | yes |
| miscutil | GC-time parse + byte/format helpers | util/misc.go | 212 | 9 | L2 | strip-last-field retry parse, strict `>` unit thresholds, divisibility-based precision, zero-copy String, in-place ASCII upper, hex | yes |
| kvrpcbatch | size/count batching | internal/kvrpc/batch.go | 82 | 7 | L2 | emit-when-accumulated>=limit-before-add, final flush, count-based `>` boundary (off-by-one), empty input no-op, ttl lookup by string key | yes |
| snapscan | snapshot scanner | txnkv/txnsnapshot/scan.go | 365 | 10 | L2 | batch<=1 clamp, NextKey(last) dedup, region-boundary continuation on short reads, reverse bounds, lock resolution + empty-value skip, EOF conditions | yes |
| prewrite | 2PC prewrite build + handle | txnkv/transaction/prewrite.go | 506 | 12 | L2 | assertion enum map, pessimistic-action map, forUpdateTS constraints, minCommitTS arithmetic, async safeTTL, alreadyExists->ErrKeyExist, optimistic conflict, MinCommitTs==0 fallback, undetermined RPC err | yes |
| unionstoreget | union-store Get/Iter delegation | internal/unionstore/union_store.go | 261 | 7 | L2 | buffer-shadows-snapshot, deletion marker => not found, merged iterators, presume/not-exist delegation, MemBuffer adapter | yes |
| keyflags | flag ops + key helpers | kv/{keyflags,key}.go | 354 | 11 | L2 | 4-state assertion bits, op ordering, opposing-assertion clear, persistent subset mask, PrefixNextKey carry/all-0xff, NextKey append, CmpKey, StrKey | yes |
| deadlock | wait-for graph + cycle detect | internal/mockstore/deadlock/deadlock.go | 151 | 8 | L2 | per-source edge sets, key-hash on closing edge, duplicate-edge ignore, same-target distinct-hash edges, CleanUp/CleanUpWaitFor/Expire semantics | yes |

All 15: `go build ./...` clean in each excised tree; `gold.patch` restores pristine
byte-for-byte on the excised files and touches no `*_test.go` (A12);
`cheat.patch` compiles and is plausible-but-wrong (drops the subtle commitments —
e.g. `kvrpcbatch` uses `>=` on the count boundary, `snapscan` treats any short
response as EOF and re-emits the last key, `pipelineddb` never flushes,
`latchsched` returns stale instead of waiting).

## Existing closures avoided (batch 1, 10 units)

`connarray`, `doactionbatches`, `lockresolver`, `memdbstaging`, `onregionerror`,
`pdoracle`, `pessimisticlock`, `rangetask`, `regionstoresorted`, `replicaselector`
— all in `experiments/pipeline/authored/client-go/`; their excised files
(`internal/client/conn_batch.go`, `txnkv/transaction/2pc.go` batching,
`txnkv/txnlock/*`, `internal/unionstore/memdb*.go`, region-cache/replica files)
were listed first and every candidate file was checked against that set plus
`check_unit_overlap.py` before authoring. `unioniter` (batch 2, prior session)
was kept; the other 14 share no file or symbol with it.

## Notes for the verifier

- `integration_tests/go.sum` lacks the `github.com/tikv/client-go/v2/tikv` entry
  in the pristine tree — a **baseline** failure reproduced on unmodified source;
  do not treat it as an excision defect. `prewrite` truncates 5 integration-test
  files and re-homes `unistoreClientWrapper` into `util_test.go` so the module
  still vets identically to baseline.
- `keyflags` truncates `internal/unionstore/memdb_test.go`; shared helpers
  (`encodeInt`, `checkConsist`, `fillDB`, …) are preserved via a created
  `memdb_helpers_test.go` so retained benchmark/no-race tests still compile.
- `kvrpcbatch`'s in-tree tests live in `integration_tests/` (rawkv client);
  the closure itself is 82 lines — difficulty comes entirely from the
  size-vs-add ordering and the `count > limit` boundary.
