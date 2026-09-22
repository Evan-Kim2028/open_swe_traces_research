# Details — deltasel

1. A packWindow of zero disables selection entirely: objects are fetched
   whole, returned in input order, no sorting or walking happens.
   Inferable: doc — the ObjectsToPack comment spells out the 0 semantics.
2. With a window, objects are sorted by type DESCENDING then size
   DESCENDING — the Less implementation inverts the usual direction.
   Inferable: no — the direction is an arbitrary order choice.
3. Type-contiguous runs of the sorted list are walked as separate groups,
   each group concurrently — a type change starts a new group, never a
   lookahead. Inferable: partially.
4. The first walk error wins and cancels the result — later groups still
   run to completion but their errors are discarded. Inferable: no.
5. Only blobs and trees are ever deltified — commits and tags pass through
   whole no matter how similar. Inferable: partially — the applyDelta map
   is kept and readable.
6. A target whose size is less than a sixteenth of the base's is never
   paired — the size-sanity floor short-circuits before any delta work.
   Inferable: no — the shift amount is arbitrary.
7. A fresh delta is adopted only when its size is under a limit scaled by
   base depth: half the target size times the remaining depth fraction, and
   limits at or below eight bytes are refused outright. Inferable: doc —
   the deltaSizeLimit comments explain the distribution.
8. A candidate already at the depth cap can never be a base — the limit
   collapses to zero. Inferable: doc — comment kept.
9. Re-packing an existing delta object reuses it untouched and clears the
   original from memory — it is skipped as a target in the window walk.
   Inferable: partially — the reuse comment is kept, the skip point is the
   detail.
10. Pre-existing deltas whose base is absent from the pack set, or whose
    object is not a DeltaObject, are undeltified — the original is re-
    fetched and the object is emitted whole. Inferable: partially.
11. Delta chains that loop back onto an object mid-resolution are broken by
    undeltifying the cyclic member — recursion never follows the cycle.
    Inferable: partially — the visiting-set comment describes detection,
    the break choice is the detail.
12. Objects evicted from the trailing edge of the window have their index
    entries dropped and their originals released — memory stays window-
    bounded. Inferable: partially.
13. A delta may only be based on an object of the SAME type — the backward
    window scan stops at the first different type, relying on the sort.
    Inferable: partially — the sort order makes the break sound.
14. Undeltification zeroes the depth and restores the fetched original —
    the emitted object is the full object, not the delta. Inferable:
    partially.
