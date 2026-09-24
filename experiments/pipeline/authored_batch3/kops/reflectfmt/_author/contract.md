# Contract (L2) — reflectfmt

`ReflectRecursive`/`reflectRecursive` walk a value depth-first, invoking the visitor on every node (structs descend field-by-field, maps by sorted key, slices by index, pointers/interface unwrap). `IsPrimitiveValue` decides the leaf set (strings, numbers, bools, and their named types — not structs or slices). `FormatValue` renders a value for diff output: primitives inline, strings quoted, nil pointers shown. `JSONMergeStruct` merges src fields into dst honoring the `json:` tag names and skipping unset fields. `BuildTypeName` renders a stable type name for paths (e.g. `[]*foo`, `map[string]bar`). `MethodNotFoundError.Error`/`IsMethodNotFound` are the sentinel error pair for reflective method dispatch.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `Test_Merge` | JSON-tagged struct merge: set fields copy, absent fields keep dst |
