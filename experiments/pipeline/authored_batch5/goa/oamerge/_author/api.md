# Exported API — oamerge

```go
func (s *Schema) Merge(other *Schema)
```

`createMergeItems` is the reflection table driving the merge; `mergeItems` is
the row type (both package-private).

## Pre-existing callers

`Dup`-based schema composition in the v2/v3 builders merges shared and
override schemas; `TestSchemaMergeAndDupPreserveAnyOf` exercises Merge
directly.
