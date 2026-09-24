# Contract — jsonxform

In-place JSON-tree transformation with path tracking in `pkg/jsonutils`.
Every commitment below is covered by a hidden test; every hidden test maps
to a commitment.

## Commitments

1. **In-place mutation.** `Transform` mutates the input map; string
   callbacks replace values by writing back into the parent container.
   Covered by `TestDetail01`.
2. **Path syntax (shape).** Paths are dot-joined key names identifying the
   full key chain; the leading-dot convention for path roots is an
   implementation detail — asserted only that the path identifies the key
   chain (ends with the dotted field names). Covered by `TestDetail02`.
3. **Slice paths (shape).** Slice elements are visited at the slice's path
   with `[]` appended — the same path the slice transform sees; no index.
   Covered by `TestDetail03`.
4. **Type whitelist.** `map[string]any`, `[]any`, `int64`, `float64`,
   `bool`, `string`, and nil are handled; any other type (e.g. `int`,
   `[]string`, a struct) errors. Covered by `TestDetail04`.
5. **Visit order.** Object transforms run before children are visited —
   a transform that mutates the map affects what children see; slice
   transforms likewise run before element visits and may replace the slice.
   Covered by `TestDetail05`.
6. **Composition.** String transforms compose left-to-right; an error
   aborts the walk (later transforms do not run). Covered by
   `TestDetail06`.
7. **`SortSlice`.** Returns a copy sorted by each element's JSON encoding —
   strings before numbers before objects by first-byte order; numbers
   compare by their textual encoding; input is not mutated. Covered by
   `TestDetail07`.
8. **`visitPrimitive`** passes values through unchanged. Covered by
   `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | no — shape only (dotted key chain; leading dot not pinned) |
| TestDetail03 | 3 | no — shape only (element path == slice path, per api.md `[]`) |
| TestDetail04 | 4 | partially — whitelist asserted; error text not pinned |
| TestDetail05 | 5 | partially — ordering asserted via visibility of mutations |
| TestDetail06 | 6 | yes |
| TestDetail07 | 7 | partially — JSON-encoding order asserted; stability not pinned |
| TestDetail08 | 8 | yes |
