# Closure — oav2schema

`http/codegen/openapi/v2/json_schema.go` — JSON-schema builder for the
Swagger 2.0 document: `schemaBuilder` walks `expr.DataType` trees into
`openapi.Schema` values, maintains a per-build `definitions` map with
name dedup / `openapi:typename` overrides / prefixing, projects result
types, and copies validation rules onto schemas (with array
length→items remapping and `MustGenerate` filtering).

Symbols stubbed: `newSchemaBuilder`, `BuildAttributeSchema`,
`schemaBuilder.{resultTypeRefWithPrefix,projectedResultTypeRefWithPrefix,
typeRefWithPrefix,generateResultTypeDefinition,
generateTypeDefinitionWithName,typeSchema,typeSchemaWithPrefix,
attributeTypeSchemaWithPrefix,buildAttributeSchema,buildResultTypeSchema}`,
`initSchemaValidation`, `renamedResultType`.

Exported entry point: `BuildAttributeSchema` (used by tests and by the
v2 spec builder for attribute-level schemas); the builder methods run
transitively from `openapiv2.New`/`Files` response and parameter schema
generation.

Tests removed: none wholesale. Trimmed (all panic on the stubs):
json_schema_union_test.go `TestAttributeTypeSchemaCorrelatesUnionDiscriminatorAndValue`,
`TestBuildAttributeSchemaKeepsDefinitionsSeparate`,
`TestBuildAttributeSchemaKeepsValuesForNamedStringMapKeys`;
builder_test.go `TestBuildPathFromExpr`, `TestBuildPathFromFileServer`,
`TestNewV2WithValues`, `TestNoSecurityOverridesAPISecurity`,
`TestNoSecurityOverridesServiceSecurity`,
`TestSecurityDefinitionsIncludeVisibleOperationsOnly`,
`TestStreamingResponseStatusCodes`; build_isolation_test.go
`TestBuildsAreSafeToRunTogether`, `TestBuildsKeepDefinitionsSeparate`;
public_api_test.go `TestDefaultFacadeMatchesWithValues`; files_test.go
`TestExtensions`, `TestNamedPrimitiveParamsAndHeadersUseOpenAPIBaseTypes`,
`TestSections`, `TestValidations`; description_ownership_test.go
`TestSharedErrorDefinitionDescription`,
`TestSharedErrorDefinitionLocalizedDescription`; and v3 files_test.go
`TestAuthoredExampleFixturesIgnoreRandomizer`, `TestFiles`,
`TestFilesV32` (shared schema-DSL fixture paths).

Blast radius: 22 trimmed tests across 7 files in 2 packages. Size: ~310
lines.
