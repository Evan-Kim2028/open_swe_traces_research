# Contract — deltasel

`DeltaSelector` decides which objects in a pack become deltas and against
which base, using a sliding window over the sorted object set. Every
commitment below is covered by a hidden test; every hidden test maps to a
commitment.

## Commitments

1. **Window zero.** A `packWindow` of zero disables selection entirely:
   objects are fetched whole and returned in input order — no sorting or
   walking happens. Covered by `TestDetail01`.
2. **Sort order (shape).** With a window, objects are sorted by type
   descending then size descending — the ordering runs tag, blob, tree,
   commit for equal sizes. Covered by `TestDetail02`.
3. **Type groups.** Type-contiguous runs of the sorted list are walked as
   separate groups — a type change starts a new group; similar blobs
   deltify while an adjacent commit never joins their group. Covered by
   `TestDetail03`.
4. **First error wins (shape).** A walk error cancels the whole result —
   the call returns a non-nil error rather than a partial list. Covered by
   `TestDetail04`.
5. **Eligible types.** Only blobs and trees are ever deltified — commits and
   tags pass through whole no matter how similar. Covered by `TestDetail05`.
6. **Size floor (shape).** A target whose size is less than a sixteenth of
   the base's is never paired — the size-sanity floor short-circuits before
   any delta work. Covered by `TestDetail06`.
7. **Size limit.** A fresh delta is adopted only when its size is under a
   limit scaled by base depth: half the target size times the remaining
   depth fraction, and limits at or below eight bytes are refused outright.
   Covered by `TestDetail07`.
8. **Depth cap.** A candidate already at the depth cap can never be a base —
   the limit collapses to zero. Covered by `TestDetail08`.
9. **Delta reuse.** Re-packing an existing delta object reuses it untouched
   and clears the original from memory — it is skipped as a target in the
   window walk. Covered by `TestDetail09`.
10. **Undeltification.** Pre-existing deltas whose base is absent from the
    pack set, or whose object is not a `DeltaObject`, are undeltified — the
    original is re-fetched and the object is emitted whole. Covered by
    `TestDetail10`.
11. **Cycle break (shape).** Delta chains that loop back onto an object
    mid-resolution are broken by undeltifying the cyclic member — the call
    returns rather than recursing forever, and not both members stay
    deltas. Covered by `TestDetail11`.
12. **Window eviction (shape).** With a window smaller than the set the
    walk still completes correctly — evicted objects' originals are
    released; the asserted shape is that every input object is still
    emitted. Covered by `TestDetail12`.
13. **Same-type only.** A delta may only be based on an object of the same
    type — identical content under different types never pairs. Covered by
    `TestDetail13`.
14. **Undeltify reset.** Undeltification zeroes the depth and restores the
    fetched original — the emitted object is the full object, not the delta.
    Covered by `TestDetail14`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | no — shape only |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | no — shape only |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | no — shape only |
| TestDetail07 | 7 | doc |
| TestDetail08 | 8 | doc |
| TestDetail09 | 9 | partially |
| TestDetail10 | 10 | partially |
| TestDetail11 | 11 | partially |
| TestDetail12 | 12 | partially |
| TestDetail13 | 13 | partially |
| TestDetail14 | 14 | partially |
