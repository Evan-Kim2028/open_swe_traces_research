# Contract — idxindex

`MemoryIndex`, `LazyIndex`, the pack-scan `Writer` observer, and the entry
iterators of `plumbing/format/idxfile`. Every commitment below is covered
by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Bucketed lookup.** Hash lookup is fanout-bucketed by first byte —
   a hash is found when present, and misses are reported both inside a
   populated bucket and against an empty bucket. Covered by
   `TestDetail01`.
2. **MayContain.** Answers from the fanout mapping alone: `false` is
   authoritative for a hash whose first byte lands in an empty bucket;
   `true` for present hashes and for same-bucket near-misses. Covered by
   `TestDetail02`.
3. **64-bit offsets.** An offset larger than 32 bits round-trips through
   the overflow table while small offsets still resolve. Covered by
   `TestDetail03`.
4. **Lazy reverse map (shape).** `FindHash` answers offset→hash lookups
   and does not rebuild the mapping per call — asserted by mutating the
   names table after the first lookup and observing the cached answer.
   Covered by `TestDetail04`.
5. **Two orderings.** `Entries` yields entries in fanout (hash-sorted)
   order while `EntriesByOffset` yields them sorted by pack offset —
   distinct orderings on the same index. Covered by `TestDetail05`.
6. **Prefix iteration.** `EntriesWithPrefix` yields exactly the matching
   run — one-byte and multi-byte prefixes narrow the result, an empty
   prefix yields everything, and an absent in-bucket prefix yields
   nothing. Covered by `TestDetail06`.
7. **Lazy descriptor sharing.** `LazyIndex` resolves Contains /
   FindOffset / FindCRC32 / Count correctly through `ReadAt`, and the
   descriptor is opened once and shared — asserted by counting opener
   calls across several lookups. Covered by `TestDetail07`.
8. **Rev-consulted FindHash (shape).** With a `.rev` present, `FindHash`
   resolves positions through the rev file — asserted by observing reads
   on the rev descriptor during the call — and an unusable rev opener
   surfaces an error at construction. Covered by `TestDetail08`.
9. **Writer footer gate.** `Index()` errors before `OnFooter` (both
   before and after object adds), and `Finished()` flips only once the
   footer arrives. The count-mismatch clause of this DETAILS row is not
   asserted: no public Writer call rejects an announced/delivered count
   difference, so the observable commitment exercised here is the
   footer gate alone. Covered by `TestDetail09`.
10. **Add dedup (shape).** The same hash added twice yields a single
    index entry carrying the first add's offset. Covered by
    `TestDetail10`.
11. **Observer contract.** `OnInflatedObjectContent` records
    hash+position+crc and ignores payload bytes;
    `OnInflatedObjectHeader` records nothing. Covered by `TestDetail11`.
12. **Overflow ordering.** Multiple beyond-32-bit offsets resolve
    independently to their own values — no aliasing between overflow
    slots. Covered by `TestDetail12`.
13. **Iterator poisoning (shape).** `Next` after `Close` never yields a
    live entry — asserted for both the in-memory iterator and a lazy
    iterator whose shared handle is released. Covered by `TestDetail13`.
14. **Prefix-iter aliasing.** The prefix iterator views the parent's
    bucket storage — an in-place mutation of the names table after
    iterator creation is observed through it. Covered by `TestDetail14`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | no — shape only |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | doc |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | no — shape only |
| TestDetail09 | 9 | partially — count-mismatch clause refused (not enforced) |
| TestDetail10 | 10 | no — shape only |
| TestDetail11 | 11 | partially |
| TestDetail12 | 12 | partially |
| TestDetail13 | 13 | no — shape only |
| TestDetail14 | 14 | doc |
