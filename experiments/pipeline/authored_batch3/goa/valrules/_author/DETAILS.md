# Commitments — valrules

1. Validate reports six bound-conflict errors: minimum+exclusiveMinimum
   both set, maximum+exclusiveMaximum both set, minimum>maximum,
   minimum>=exclusiveMaximum, exclusiveMinimum>exclusiveMaximum,
   exclusiveMinimum>=maximum, plus minLength>maxLength. In-tree
   coverage: trimmed tests. Inferable: no — the exclusive-bound
   cross-comparisons are a precise table.
2. Merge keeps existing values when set but tightens bounds: takes the
   SMALLER of minimums/exclusiveMinimums/minLengths and the LARGER of
   maximums/exclusiveMaximums/maxLengths; Values/Format/Pattern are
   taken only when unset; Required is unioned. In-tree coverage: none.
   Inferable: no — the tighten-direction is the subtle half.
3. AddRequired appends only names not already present. In-tree
   coverage: trimmed tests. Inferable: partially.
4. RemoveRequired removes the first matching name only. In-tree
   coverage: none. Inferable: partially.
5. HasRequiredOnly is true when every field except Required is
   zero-valued — no Values, no Format, no Pattern, no min/max bounds of
   either kind, no length bounds. In-tree coverage: trimmed tests.
   Inferable: partially.
6. Dup copies every field and duplicates the Required slice (fresh
   backing array) while scalar pointer fields are shared. In-tree
   coverage: none. Inferable: partially — "shallow dup" is documented.
