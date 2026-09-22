# Closure — oatags

`http/codegen/openapi/tags.go` — extraction of OpenAPI tag
metadata: `TagsFromExpr`/`TagNamesFromExpr` parse `swagger:tag:` /
`openapi:tag:` meta keys into sorted `Tag` records (deduped by name), gate the
OpenAPI-3.2-only fields (summary/parent/kind) on the target version, attach
externalDocs and `x-*` extensions, and marshal through the shared
extension-aware encoders.

Symbols stubbed: `TagsFromExpr`, `TagNamesFromExpr`, `parseTags`, `Tag.MarshalJSON`, `Tag.MarshalYAML`.

Tests removed (reach the stubs, verified by excision):
- `http/codegen/openapi/tags_test.go`: `TestTagsFromExpr`
- `http/codegen/openapi/v2/build_isolation_test.go`: `TestBuildsAreSafeToRunTogether`, `TestBuildsKeepDefinitionsSeparate`
- `http/codegen/openapi/v2/builder_test.go`: `TestBuildPathFromExpr`, `TestBuildPathFromFileServer`, `TestNewV2WithValues`, `TestSecurityDefinitionsIncludeVisibleOperationsOnly`, `TestNoSecurityOverridesAPISecurity`, `TestNoSecurityOverridesServiceSecurity`, `TestStreamingResponseStatusCodes`
- `http/codegen/openapi/v2/description_ownership_test.go`: `TestSharedErrorDefinitionDescription`, `TestSharedErrorDefinitionLocalizedDescription`
- `http/codegen/openapi/v2/files_test.go`: `TestExtensions`, `TestNamedPrimitiveParamsAndHeadersUseOpenAPIBaseTypes`, `TestValidations`
- `http/codegen/openapi/v2/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
- `http/codegen/openapi/v3/builder_test.go`: `TestBuildOperation`, `TestBuildOperationID`, `TestNewWithValues`, `TestNoSecurityOverridesAPISecurity`, `TestNoSecurityOverridesServiceSecurity`, `TestSSEItemSchemaMatchesDataContract`, `TestSecuritySchemesIncludeVisibleOperationsOnly`, `TestStreamingResponseStatusCodes`
- `http/codegen/openapi/v3/description_ownership_test.go`: `TestSharedErrorComponentDescription`, `TestSharedErrorComponentLocalizedDescription`, `TestSharedErrorResponseDescriptions`, `TestUndescribedSharedErrorIgnoresLocalizedResponseDescription`
- `http/codegen/openapi/v3/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
