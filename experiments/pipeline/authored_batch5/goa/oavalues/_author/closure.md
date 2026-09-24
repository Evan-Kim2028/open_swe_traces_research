# Closure — oavalues

`http/codegen/openapi/values.go` — the copy-on-write `Values`
store carrying alternate titles/descriptions/examples through one spec build:
`With*` methods return independent copies, lookups key on the authored
attribute (with a user-type fallback), stored examples materialize with
translated descriptions applied by source identity, and `Example` merges
stored+authored examples before delegating to the attribute's generator.

Symbols stubbed: `Values.WithTitle`, `Values.WithDescription`, `Values.WithExamples`, `Values.Title`, `Values.Description`, `Values.Examples`, `Values.Example`, `Values.copy`, `storeExamples`, `Values.materializeExamples`.

Tests removed (reach the stubs, verified by excision):
- `http/codegen/openapi/v2/build_isolation_test.go`: `TestBuildsAreSafeToRunTogether`, `TestBuildsKeepDefinitionsSeparate`
- `http/codegen/openapi/v2/builder_test.go`: `TestBuildPathFromExpr`, `TestBuildPathFromFileServer`, `TestNewV2WithValues`, `TestSecurityDefinitionsIncludeVisibleOperationsOnly`, `TestNoSecurityOverridesAPISecurity`, `TestNoSecurityOverridesServiceSecurity`, `TestOperationSecurityMarshal`, `TestStreamingResponseStatusCodes`
- `http/codegen/openapi/v2/description_ownership_test.go`: `TestSharedErrorDefinitionDescription`, `TestSharedErrorDefinitionLocalizedDescription`
- `http/codegen/openapi/v2/files_test.go`: `TestExtensions`, `TestNamedPrimitiveParamsAndHeadersUseOpenAPIBaseTypes`, `TestValidations`
- `http/codegen/openapi/v2/json_schema_union_test.go`: `TestBuildAttributeSchemaKeepsDefinitionsSeparate`, `TestBuildAttributeSchemaKeepsValuesForNamedStringMapKeys`
- `http/codegen/openapi/v2/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
- `http/codegen/openapi/v3/builder_test.go`: `TestBuildInfo`, `TestBuildOperation`, `TestBuildOperationID`, `TestBuildServersKeepsVariableValuesSeparate`, `TestNewWithValues`, `TestNoSecurityOverridesAPISecurity`, `TestNoSecurityOverridesServiceSecurity`, `TestSSEItemSchemaMatchesDataContract`, `TestSecuritySchemesIncludeVisibleOperationsOnly`, `TestStreamingResponseStatusCodes`, `TestOperationSecurityMarshal`
- `http/codegen/openapi/v3/description_ownership_test.go`: `TestSharedErrorComponentDescription`, `TestSharedErrorComponentLocalizedDescription`, `TestSharedErrorResponseDescriptions`, `TestUndescribedSharedErrorIgnoresLocalizedResponseDescription`
- `http/codegen/openapi/v3/example_test.go`: `TestInitExamplesUsesReplacementDescription`
- `http/codegen/openapi/v3/parameters_test.go`: `TestHeaderSchemaAndDisplayedExampleShareIdentity`, `TestParamForAllowEmptyValue`
- `http/codegen/openapi/v3/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
- `http/codegen/openapi/v3/types_test.go`: `TestBuildBodyTypes`, `TestBuildBodyTypesPreservesPrimitiveAliasComponents`, `TestMapTypes`, `TestTypesOnlyDifferByEnum`
- `http/codegen/openapi/v3/types_union_test.go`: `TestSchemafyCorrelatesUnionDiscriminatorAndValue`
- `http/codegen/openapi/values_test.go`: `TestDocsFromExprWithValues`, `TestValues`, `TestValuesApplyExampleDescriptionsRegardlessOfCallOrder`, `TestValuesOwnCompleteExampleLists`, `TestValuesUseAuthoredAttributeForCopies`
