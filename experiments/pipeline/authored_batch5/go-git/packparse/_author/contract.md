# Contract — packparse

The parser decodes a packfile stream, queues deltas during the scan, and
resolves them afterwards — including thin-pack bases fetched from storage.
Every commitment below is covered by a hidden test; every hidden test maps to
a commitment.

## Commitments

1. **Single-shot.** A second call to parse on the same parser returns the
   documented consumed error without doing any work — even when the first
   call failed. Covered by `TestDetail01`.
2. **Deferred resolution.** Deltas observed during the scan are queued and
   resolved only after the whole pack has been scanned: a delta that appears
   in the stream before its base still resolves to its target content.
   Covered by `TestDetail02`.
3. **Joint child advance.** Resolution walks the delta DAG depth-first from
   each non-delta base and advances REF-delta and OFS-delta children of every
   parent together: a REF delta whose base is itself an OFS delta resolves.
   Covered by `TestDetail03`.
4. **Thin-pack fallback.** A REF delta whose base hash is not in the pack is
   treated as an external reference: with the base present in storage the
   parse resolves it; without it the parse fails. Covered by `TestDetail04`.
5. **OFS strictness.** An OFS delta whose recorded base offset matches no
   in-pack object is rejected as malformed input — the parse returns an
   error; there is no external fallback for offsets. Covered by
   `TestDetail05`.
6. **Depth cap.** A delta chain longer than the documented bound is rejected
   — the parse returns an error rather than resolving it. Covered by
   `TestDetail06`.
7. **Silent low-memory gating.** Low-memory mode engages only when storage
   opts in through the capability probe and the source is seekable; a
   non-seekable source disables it silently — the parse still resolves
   deltas correctly in that configuration, with or without capable storage.
   Covered by `TestDetail07`.
8. **Low-memory correctness (shape).** When a capable storage opts into
   low-memory mode and the source is seekable, resolved delta content is
   still correct. The internal buffer release up the parent chain is not
   directly observable; the asserted shape is result correctness under the
   mode. Covered by `TestDetail08`.
9. **Empty sentinel.** A source that reaches end of input having produced
   zero objects reports the empty-packfile sentinel, not a raw EOF.
   Covered by `TestDetail09`.
10. **Grow-hint clamp.** The staging-buffer grow hint is clamped to the
    documented bound; oversized and negative hints are clamped, ordinary
    hints pass through. Covered by `TestDetail10`.
11. **Recorded IDs preserved (shape).** A resolved delta lands in storage
    under the hash of its resolved contents. The internal rule — declared
    type/size/hash are only filled when no object ID was already recorded —
    is not observable at this boundary; the asserted shape is that the
    stored delta answers to its resolved content hash. Covered by
    `TestDetail11`.
12. **External base metadata.** A resolved delta whose parent is an
    external reference inherits the real type of the base fetched from
    storage, not a guessed type. Covered by `TestDetail12`.
13. **Preallocation cap.** The object cache's up-front reservation is capped
    at the documented bound even when the pack header advertises a far
    larger count. Covered by `TestDetail13`.
14. **One store per entry (shape).** Parsing a pack of N entries commits
    exactly N object writes to storage — non-deltas are written by the
    scanning layer, resolved deltas by the parser, nothing twice. Covered by
    `TestDetail14`.
15. **Observer protocol.** Observer callbacks fire in section order —
    header count, then per-object header and content pairs, then the footer
    hash — and the first observer error aborts the parse. Covered by
    `TestDetail15`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | doc |
| TestDetail04 | 4 | doc |
| TestDetail05 | 5 | doc |
| TestDetail06 | 6 | doc |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | no — shape only |
| TestDetail09 | 9 | partially |
| TestDetail10 | 10 | doc |
| TestDetail11 | 11 | no — shape only |
| TestDetail12 | 12 | partially |
| TestDetail13 | 13 | doc |
| TestDetail14 | 14 | no — shape only |
| TestDetail15 | 15 | partially |
