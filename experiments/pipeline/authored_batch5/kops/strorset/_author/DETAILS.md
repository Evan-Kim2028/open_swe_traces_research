# Details — strorset

1. `MarshalJSON` encodes as an array when `forceEncodeAsArray` OR `len(values) > 1`; a
   single non-forced value encodes as a bare JSON string; an empty non-forced set still
   encodes `[]` (the `values == nil` guard normalises nil to the empty slice first).
   Inferable: partially — the empty-set shape is a choice.
2. `UnmarshalJSON` keys only on the first byte being `[`: a string payload clears the
   array flag, an array payload sets it (so `["x"]` round-trips as `["x"]`, not `"x"`).
   A malformed array payload is swallowed (returns nil). Inferable: partially — the
   swallowed error is surprising.
3. `String()` comma-joins the SORTED values with no spaces. Inferable: partially — sorted
   order vs insertion order is a choice (the set makes it deterministic).
4. `Value()` returns a sorted copy — sorted twice on the marshal path (sets.List already
   sorts; `sort.Strings` is applied again). Inferable: yes.
5. `Set` forces array encoding even for a single element; `Of` only array-encodes when
   len > 1; `String` never array-encodes. `Set([]string{"a"})` → `["a"]` but
   `Of("a")`/`String("a")` → `"a"`. Inferable: partially — the per-constructor flag is a
   choice, though the doc comments state it.
6. `Equal` is set equality — order-insensitive. Inferable: yes.
