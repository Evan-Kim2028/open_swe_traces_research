# Contract (L2) — tfliterals

A `Literal` is one HCL expression rendered verbatim from `String`. Resource refs render `type.name.prop`, data refs `data.type.name.prop`, with the name sanitized. Function calls render `fn(a, b)`; lists `[a, b]`; index `coll[idx]`; binary `l op r`; the empty-string conditional renders `e == "" ? null : v`; `WithIndex` renders `"s-${count.index}"`. String values are quoted verbatim (no escaping); ints render decimal. `SortLiterals` sorts in place; `dedupLiterals` sorts (documented side effect) and drops adjacent duplicates. `Write` emits `= <string>` ignoring indent and key.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestGetOutputs` (shared file; also covered under tfwriter) | output variables collect literals that render correctly |
