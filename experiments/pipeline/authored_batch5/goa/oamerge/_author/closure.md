# Closure — oamerge

`http/codegen/openapi/merge.go` — two-level `Schema.Merge`: fills
missing scalar fields, merges composite fields (AnyOf, Items, Enum,
Properties, Defs) only when absent, tightens numeric bounds (min takes the
larger, max the smaller), and appends links and required lists.

Symbols stubbed: `Schema.Merge`, `Schema.createMergeItems`.

Tests removed (reach the stubs, verified by excision):
- `http/codegen/openapi/merge_test.go`: `TestSchemaMergeAndDupPreserveAnyOf`
- `http/codegen/openapi/v2/build_isolation_test.go`: `TestBuildsAreSafeToRunTogether`, `TestBuildsKeepDefinitionsSeparate`
- `http/codegen/openapi/v2/builder_test.go`: `TestBuildPathFromFileServer`
- `http/codegen/openapi/v2/description_ownership_test.go`: `TestSharedErrorDefinitionDescription`, `TestSharedErrorDefinitionLocalizedDescription`
- `http/codegen/openapi/v2/files_test.go`: `TestExtensions`, `TestValidations`
- `http/codegen/openapi/v2/json_schema_union_test.go`: `TestBuildAttributeSchemaKeepsDefinitionsSeparate`, `TestBuildAttributeSchemaKeepsValuesForNamedStringMapKeys`
