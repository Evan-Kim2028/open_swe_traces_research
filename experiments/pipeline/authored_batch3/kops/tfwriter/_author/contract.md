# Contract (L2) — tfwriter

`TerraformWriter` accumulates resources, data sources, outputs, providers, and staged files for later rendering. Names are sanitized on the way out (`.→-`, `/→--`, `:→_`, digit-first gets `prefix_`); sanitized-name collisions are detected at query time and are errors. Scalar and array outputs for the same key conflict; array outputs dedup+sort on read. `EnsureTerraformProvider` reuses an identical registration and fatals on a conflicting one. `AddFileBytes`/`AddFilePath` stage bytes under `data/` and return the module-path literal (wrapped `file()`/`filebase64()`). `Get*ByType` group by type then sanitized name.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestGetOutputs` | outputs collect, dedup, sort, and surface sanitized-name collisions |
