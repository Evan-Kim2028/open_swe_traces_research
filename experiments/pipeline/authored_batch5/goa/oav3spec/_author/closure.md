# Closure — oav3spec

`http/codegen/openapi/v3/openapi.go` — OpenAPI 3 document serialization:
per-type `MarshalJSON`/`MarshalYAML` methods that delegate to
`openapi.MarshalJSON`/`openapi.MarshalYAML` through anti-recursion alias
types and merge per-object `Extensions` into the top-level document map,
`SecurityRequirements.IsZero` driving `json:",omitzero"`, and the
`exampler` setters (`setExample`/`setExamples`) used by `initExamples`.

Symbols stubbed: `MediaType.setExample`, `MediaType.setExamples`,
`Header.setExample`, `Header.setExamples`, `Parameter.setExample`,
`Parameter.setExamples`, `Info.MarshalJSON`, `PathItem.MarshalJSON`,
`Operation.MarshalJSON`, `Parameter.MarshalJSON`, `Response.MarshalJSON`,
`SecurityScheme.MarshalJSON`, `Info.MarshalYAML`, `PathItem.MarshalYAML`,
`Operation.MarshalYAML`, `Parameter.MarshalYAML`, `Response.MarshalYAML`,
`SecurityScheme.MarshalYAML`, `SecurityRequirements.IsZero`.

Exported entry points: the marshalers satisfy `json.Marshaler` /
`yaml.Marshaler` implicitly during template render; `setExample`/
`setExamples` satisfy the `exampler` interface consumed by
`initExamples`; `IsZero` is read by `encoding/json`'s `omitzero`.

Tests removed: none wholesale. Trimmed (all panic on the stubs):
files_test.go `TestAuthoredExampleFixturesIgnoreRandomizer`, `TestFiles`,
`TestFilesV32`; builder_test.go `TestBuildOperation`,
`TestSSEItemSchemaMatchesDataContract`,
`TestNoSecurityOverridesAPISecurity`,
`TestNoSecurityOverridesServiceSecurity`,
`TestStreamingResponseStatusCodes`, `TestOperationSecurityMarshal`;
parameters_test.go `TestHeaderSchemaAndDisplayedExampleShareIdentity`,
`TestParamForAllowEmptyValue`; example_test.go
`TestInitExamplesUsesReplacementDescription`;
description_ownership_test.go `TestSharedErrorComponentDescription`,
`TestSharedErrorComponentLocalizedDescription`,
`TestSharedErrorResponseDescriptions`,
`TestUndescribedSharedErrorIgnoresLocalizedResponseDescription`;
public_api_test.go `TestDefaultFacadeMatchesWithValues`; and in
openapi/v2: files_test.go `TestSections`, builder_test.go
`TestNoSecurityOverridesServiceSecurity`,
`TestStreamingResponseStatusCodes`, `TestOperationSecurityMarshal`,
public_api_test.go `TestDefaultFacadeMatchesWithValues` (shared helper
paths panic through the v3 stubs).

Blast radius: 22 trimmed tests across 8 files in 2 packages.
