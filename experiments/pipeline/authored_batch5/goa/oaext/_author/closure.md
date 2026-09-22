# Closure — oaext

`http/codegen/openapi/extensions.go` — `x-*` extension extraction:
`ExtensionsFromExpr` merges the `swagger:extension:` and
`openapi:extension:` meta families, `extensionsFromExprWithPrefix` filters to
single-level `x-` names and JSON-parses values (raw string on failure), and
`ExtensionsFromMethod` additionally advertises `x-goa-idempotent` for
idempotent methods.

Symbols stubbed: `ExtensionsFromExpr`, `ExtensionsFromMethod`, `extensionsFromExprWithPrefix`.

Tests removed (reach the stubs, verified by excision):
- `http/codegen/openapi/extensions_test.go`: `TestExtensionsFromMethod`
- `http/codegen/openapi/v2/build_isolation_test.go`: `TestBuildsAreSafeToRunTogether`, `TestBuildsKeepDefinitionsSeparate`
- `http/codegen/openapi/v2/builder_test.go`: `TestBuildPathFromExpr`, `TestBuildPathFromFileServer`, `TestNewV2WithValues`, `TestSecurityDefinitionsIncludeVisibleOperationsOnly`, `TestNoSecurityOverridesAPISecurity`, `TestNoSecurityOverridesServiceSecurity`, `TestOperationSecurityMarshal`, `TestStreamingResponseStatusCodes`
- `http/codegen/openapi/v2/description_ownership_test.go`: `TestSharedErrorDefinitionDescription`, `TestSharedErrorDefinitionLocalizedDescription`
- `http/codegen/openapi/v2/files_test.go`: `TestExtensions`, `TestNamedPrimitiveParamsAndHeadersUseOpenAPIBaseTypes`, `TestValidations`
- `http/codegen/openapi/v2/json_schema_union_test.go`: `TestBuildAttributeSchemaKeepsDefinitionsSeparate`, `TestBuildAttributeSchemaKeepsValuesForNamedStringMapKeys`
- `http/codegen/openapi/v2/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
- `http/codegen/openapi/v3/builder_test.go`: `TestBuildInfo`, `TestBuildOperation`, `TestBuildOperationID`, `TestNewWithValues`, `TestNoSecurityOverridesAPISecurity`, `TestNoSecurityOverridesServiceSecurity`, `TestSSEItemSchemaMatchesDataContract`, `TestSecuritySchemesIncludeVisibleOperationsOnly`, `TestStreamingResponseStatusCodes`, `TestOperationSecurityMarshal`
- `http/codegen/openapi/v3/description_ownership_test.go`: `TestSharedErrorComponentDescription`, `TestSharedErrorComponentLocalizedDescription`, `TestSharedErrorResponseDescriptions`, `TestUndescribedSharedErrorIgnoresLocalizedResponseDescription`
- `http/codegen/openapi/v3/parameters_test.go`: `TestHeaderSchemaAndDisplayedExampleShareIdentity`, `TestParamForAllowEmptyValue`
- `http/codegen/openapi/v3/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
- `http/codegen/openapi/v3/types_test.go`: `TestBuildBodyTypes`, `TestBuildBodyTypesPreservesPrimitiveAliasComponents`, `TestMapTypes`, `TestTypesOnlyDifferByEnum`
- `http/codegen/openapi/v3/types_union_test.go`: `TestSchemafyCorrelatesUnionDiscriminatorAndValue`
- `http/codegen/openapi/values_test.go`: `TestDocsFromExprWithValues`
