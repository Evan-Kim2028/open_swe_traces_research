# Verified — VFbatch5clientgo

Hidden test suites for the 20 client-go units in
`experiments/pipeline/authored_batch5/client-go`. Image
`ladder-base:client-go-obf`, module `example.internal/kvstore/v2`,
go1.21.13. Every suite is `TestDetail01..NN` numbered 1:1 to the unit's
`DETAILS.md`; `Inferable: no` lines are asserted as committed shape
(round-trip identity, ordering, presence, panic, relational behavior),
never as invented literals. Gold was used only to avoid asserting
behavior gold cannot satisfy — never to derive assertions.

All 20 units pass all four checks: excised+hidden → FAIL, gold → PASS
(reward=1), cheat → FAIL, A12 (gold touches no test file) → PASS.
Every `tests/test.sh` sha256 matches its hidden file; test counts equal
DETAILS line counts; every unit has a `_author/contract.md` with one row
per hidden test. Durable log: `outputs/VFbatch5clientgo.log`.

## Per-unit results

| unit | tests | Inferable mix | excised | gold | cheat | A12 |
|---|---|---|---|---|---|---|
| batchbuild | 6 | doc/partially/no | FAIL | PASS | FAIL | PASS |
| bytesfmt | 7 | partially/no | FAIL | PASS | FAIL | PASS |
| codecbytes | 8 | doc/yes | FAIL | PASS | FAIL | PASS |
| codecnum | 12 | doc/yes/partially | FAIL | PASS | FAIL | PASS |
| configpath | 5 | doc/partially/no | FAIL | PASS | FAIL | PASS |
| clusterops | 8 | partially/no | FAIL | PASS | FAIL | PASS |
| clusterq | 8 | partially/no/yes | FAIL | PASS | FAIL | PASS |
| deadlock | 7 | doc/partially/no | FAIL | PASS | FAIL | PASS |
| execfmt | 6 | doc/partially | FAIL | PASS | FAIL | PASS |
| hexdump | 4 | no/partially | FAIL | PASS | FAIL | PASS |
| keyerrors | 6 | yes/partially/no | FAIL | PASS | FAIL | PASS |
| keyflags | 9 | doc/partially | FAIL | PASS | FAIL | PASS |
| keyops | 8 | doc/partially/no | FAIL | PASS | FAIL | PASS |
| mvcccodec | 6 | no/partially/yes | FAIL | PASS | FAIL | PASS |
| mvccread | 6 | doc/yes/partially/no | FAIL | PASS | FAIL | PASS |
| priorityqueue | 6 | doc/no | FAIL | PASS | FAIL | PASS |
| reqsource | 6 | partially/doc | FAIL | PASS | FAIL | PASS |
| ruinfo | 5 | partially/yes | FAIL | PASS | FAIL | PASS |
| slowscore | 10 | doc/partially/no | FAIL | PASS | FAIL | PASS |
| unioniter | 6 | doc/yes/partially/no | FAIL | PASS | FAIL | PASS |

## What was asserted for `Inferable: no` (and notable weakenings)

- **batchbuild** — D6 asserts heap-pop order shape (max priority first,
  full set emitted) not sorted order; gold emits raw heap-array order.
- **bytesfmt** — format literals underivable (the in-tree test pinning
  them was excised); asserted delegation equality, pruning divergence,
  scaling, and zero-copy `String`. Cheat fails on zero-copy.
- **codecnum** — gold quirk: `DecodeComparableVarint` does not consume
  the tag byte for single-byte inline positives. Asserted value-only
  there; full consume-suffix round-trip only for multi-byte forms.
- **configpath** — DETAILS says `tikv`, visible doc example says
  `kvstore://`; the suite *discovers* an accepted scheme rather than
  pinning either literal.
- **clusterops** — committed panics asserted (id-list length mismatch);
  epoch bumps asserted as "changed" not to specific values where the
  starting values are implementation data. `NewCluster(nil)` used to
  avoid a goleak failure from `MustNewMVCCStore` LevelDB goroutines.
- **clusterq** — committed panics asserted (empty-cluster
  `GetPrevRegionByKey`, missing-store `MarkTombstone`); ScanRegions
  boundary exclusion, leaderless empty peer, and down-peer filtering
  asserted as documented shapes.
- **deadlock** — non-registration on cycle made observable via a chained
  `Detect` probe; reported `KeyHash` asserted as "the stored edge's hash"
  via a two-edge setup distinguishing it from the incoming hash.
- **hexdump** — all shape: brace form, lowercase hex for byte slices,
  nested `[][]byte` recursion, nil handling, enum rendering by name.
- **keyerrors** — priority asserted per-field via committed error type
  mapping; the failpoint path tested via `failpoint.Enable`
  (runtime-enablable, wiring is solver-visible).
- **keyops** — DETAILS' `"a\xff"->"b"` carry example is wrong (gold:
  `"b\x00"`); asserted successor shape (strictly greater, never extends)
  plus non-carry literals; all-`0xFF` corner retained (committed).
- **mvcccodec** — format literals underivable; asserted deterministic
  round-trip plus per-field participation; oversize-slice rejection
  asserted without pinning the 10MiB cap constant.
- **mvccread** — max-ts own-key short-circuit asserted as
  "below `startTS`" shape, not the `startTS-1` literal; `Less` asserted
  against `bytes.Compare` of the encodings.
- **priorityqueue** — the excision deletes `priority_queue_test.go`
  (with `FakeItem`), but `client_batch.go` stays visible and calls
  `highestPriority() >= highTaskPriority`, so max-first direction is
  derivable; suite defines its own item type.
- **ruinfo** — dropped a `ReplicaNumber` propagation assertion (gold
  returns 0; DETAILS does not commit its source). Cheat fails on V1-ms
  precedence.
- **slowscore** — magic constants asserted as direction/bounds shapes
  (rise only when throughput falls while latency rises; decay toward 1;
  snap within one step).
- **unioniter** — tombstone semantics additionally grounded by the kept
  in-tree `union_store_test.go`; asserted dirty-wins, tombstone skip,
  and reverse ordering as committed shapes.

## Refused assertions

- `ruinfo` — `ReplicaNumber` propagation (uncommitted; gold returns 0).
- `keyops` — the `"a\xff" -> "b"` carry literal (DETAILS example
  contradicts gold; shape asserted instead).
- `codecnum` — tag-byte consumption for single-byte comparable varints
  (gold leaves it unconsumed; underivable boundary).
- `keyflags` D5 — internal cross-flag clearing mechanics beyond the
  committed "constraint-check state cleared" shape.
- `configpath` — any specific scheme literal (visible docs and DETAILS
  conflict; accept/reject discovery used).
- `ruinfo` / `execfmt` — no microsecond-format or byte-count literals
  beyond what doc comments commit.
