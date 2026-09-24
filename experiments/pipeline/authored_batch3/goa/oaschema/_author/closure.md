# Closure — oaschema

`http/codegen/openapi/json_schema.go` + `merge.go` — schema duplication
and merge, example projection, JSON value conversion helpers.

Symbols stubbed: `ToString`, `ToStringMap`, `ProjectExample`,
`projectExample`, `projectObjectExample`, `projectArrayExample`,
`projectMapExample`, `exampleMap`, `exampleSlice`, `MustGenerate`,
`AdditionalPropertiesFromExpr`, `(*Schema).Dup`, `duplicateJSONValue`,
`duplicateJSONReflectValue`, `(*Schema).Merge`,
`(*Schema).createMergeItems`.

Tests removed: `json_schema_dup_test.go`, `merge_test.go` deleted.
