# Contract — strorset

`StringOrSet` JSON/string semantics in `pkg/util/stringorset`. Every
commitment below is covered by a hidden test; every hidden test maps to a
commitment.

## Commitments

1. **Marshal shape.** `MarshalJSON` emits a JSON array when the value was
   built by `Set` (forced) or holds more than one element; a single
   non-forced value emits a bare JSON string; an empty non-forced set emits
   `[]`. Covered by `TestDetail01`.
2. **Unmarshal semantics.** `UnmarshalJSON` keys on the first byte `[`:
   an array payload sets forced-array state (so `["x"]` re-marshals as
   `["x"]`), a string payload clears it. A malformed array payload returns
   nil (swallowed). A non-`[`, non-string payload (e.g. an object) takes the
   string path and errors. Covered by `TestDetail02`.
3. **String() ordering.** `String()` comma-joins the sorted values with no
   spaces. Covered by `TestDetail03`.
4. **Value() copy.** `Value()` returns a sorted copy — mutating the result
   does not change the source. Covered by `TestDetail04`.
5. **Constructor forcing.** `Set` array-encodes even one element;
   `Of` array-encodes only for len > 1; `String` never array-encodes.
   Covered by `TestDetail05`.
6. **Set equality.** `Equal` is order-insensitive and ignores the
   forced-array flag. Covered by `TestDetail06`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — all three shapes committed and asserted |
| TestDetail02 | 2 | partially — flag flip, round-trip, swallowed error asserted |
| TestDetail03 | 3 | partially — sorted comma-join committed, asserted |
| TestDetail04 | 4 | yes — sorted copy semantics |
| TestDetail05 | 5 | partially — per-constructor flag asserted per doc comments |
| TestDetail06 | 6 | yes — order-insensitive set equality |
