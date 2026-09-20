# Details — fivalues

1. `ValueOf` returns the zero value on nil — `*new(T)` — and dereferences otherwise. Inferable: yes.
2. `StringSliceValue` skips nil elements (never returns entries for them) and returns nil — not an empty slice — when nothing survives. Inferable: partially.
3. `StringSlice` returns pointers INTO the input slice's elements (`&s[i]`), not copies — mutating the result's targets mutates the input. Inferable: yes — the aliasing is observable.
4. `IsNilOrEmpty` is true for nil AND for `""`. Inferable: yes.
5. `DebugPrint` renders nil and typed-nil as `<nil>`, invalid reflect values as `<?>`, a `Resource` as its contents truncated to 256 chars + `... (truncated)`, a `Stringer` via `String()`, else `fmt.Sprint`. Inferable: no — the sentinel spellings are arbitrary.
6. `DebugAsJsonString`/`Indent` return an `error marshaling: ...` STRING on marshal failure rather than propagating an error. Inferable: partially.
7. `ToInt64` and `ToString` propagate nil and swallow conversion errors (nil in, nil out; unparseable in, nil out). Inferable: yes.
8. `ArrayContains` is exact-match linear search. Inferable: yes.
