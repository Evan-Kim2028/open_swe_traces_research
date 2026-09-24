# Commitments — oamerge

1. Unset fields of `s` take `other`'s values; set fields are never overwritten. In-tree coverage: `TestSchemaMergeAndDupPreserveAnyOf` (trimmed). Inferable: doc — Merge's contract is fill-missing.
2. Composite fields (AnyOf, Items, Enum, Properties, Defs) merge only when the target side is absent — merge never concatenates or deep-merges composite content. In-tree coverage: `TestSchemaMergeAndDupPreserveAnyOf`, union/isolation tests (trimmed). Inferable: partially — which fields are composite vs appended is a table choice.
3. Minimum-style bounds keep the stricter (larger) value; maximum-style bounds keep the stricter (smaller) value. In-tree coverage: builder/isolation tests (trimmed). Inferable: partially.
4. Links and Required append rather than replace. In-tree coverage: builder tests (trimmed). Inferable: partially.
