# Contract (L2) — fivalues

Nil-safe value helpers: `ValueOf` returns the zero value for nil. `StringSliceValue` drops nil elements (nil-only input → nil slice). `StringSlice` returns pointers into the input elements. `IsNilOrEmpty` covers nil and `""`. `ArrayContains` is exact match. `DebugPrint` dispatches: nil/typed-nil→`<nil>`, invalid→`<?>`, Resource→contents (256-char truncation), Stringer→`String()`, else `fmt.Sprint`. `DebugAsJsonString`/Indent return the marshal error AS the string. `ToInt64`/`ToString` propagate nil and swallow errors.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `Test_tryResourceAsString` | resource debug rendering via DebugPrint path |
| `Test_DryrunTarget_PrintReport` | dry-run report renders values through the helpers |
