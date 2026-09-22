# Closure — oav3types

`http/codegen/openapi/v3/types.go` — OpenAPI 3 body-schema builder:
`schemafier` walks `expr.DataType` trees into `openapi.Schema` values
with structural-hash dedup (`hashAttribute` + `preferredNames` +
`uniquify`), `components/schemas` ref management, response-view
projection, per-location example identities, SSE item schemas, and full
validation/default/example/extension copying.

Symbols stubbed: `schemafier.{at,member,arrayElement,mapValue,
unionMember,field,collectPreferredResponseNames,
collectPreferredResponseName,buildSSEItemSchema,schemafy,
ensureSchemaDescription,userTypeDescription,uniquify,hashAttribute}`,
`newSchemafier`, `buildBodyTypes`, `staticViewBody`,
`responseBodyProjection`, `toRef`, `toStringMap`, `toString`,
`hashAttribute`, `hashValidation`, `hashString`, `orderedHash`,
`openAPIGeneratedServices`, `mustGenerateType`.

Exported entry point: `buildBodyTypes` (package-private but the sole
driver, called by the v3 spec builder); `EndpointBodies` is the exported
result type.

Tests removed: none wholesale. Trimmed (all panic on the stubs):
types_test.go `TestBuildBodyTypes`, `TestMapTypes`,
`TestTypesOnlyDifferByEnum`,
`TestBuildBodyTypesPreservesPrimitiveAliasComponents`,
`TestHashAttribute`; files_test.go
`TestAuthoredExampleFixturesIgnoreRandomizer`, `TestFiles`,
`TestFilesV32`; builder_test.go `TestBuildOperation`,
`TestNewWithValues`, `TestNoSecurityOverridesAPISecurity`,
`TestNoSecurityOverridesServiceSecurity`,
`TestSSEItemSchemaMatchesDataContract`,
`TestSecuritySchemesIncludeVisibleOperationsOnly`,
`TestStreamingResponseStatusCodes`; public_api_test.go
`TestDefaultFacadeMatchesWithValues`; parameters_test.go
`TestHeaderSchemaAndDisplayedExampleShareIdentity`,
`TestParamForAllowEmptyValue`; types_union_test.go
`TestSchemafyCorrelatesUnionDiscriminatorAndValue`;
description_ownership_test.go `TestSharedErrorComponentDescription`,
`TestSharedErrorComponentLocalizedDescription`,
`TestSharedErrorResponseDescriptions`,
`TestUndescribedSharedErrorIgnoresLocalizedResponseDescription`.

Blast radius: 23 trimmed tests across 7 files, all inside
`http/codegen/openapi/v3`. Size: ~750 lines.
