# Closure — oaerr

`http/codegen/openapi/error_example.go` — shared error-response
details for both spec generators: `ResponseContentType` resolves a response's
content type (explicit setting, then result-type setting, then
application/json), and `ErrorResponseExample` projects the generated body
example for the built-in error result, stamping name/temporary/timeout/fault
only onto fields the response body actually carries.

Symbols stubbed: `ResponseContentType`, `ErrorResponseExample`, `setExampleField`.

Tests removed (reach the stubs, verified by excision):
- `http/codegen/openapi/v2/builder_test.go`: `TestNoSecurityOverridesAPISecurity`, `TestNoSecurityOverridesServiceSecurity`, `TestStreamingResponseStatusCodes`
- `http/codegen/openapi/v2/description_ownership_test.go`: `TestSharedErrorDefinitionDescription`, `TestSharedErrorDefinitionLocalizedDescription`
- `http/codegen/openapi/v2/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
- `http/codegen/openapi/v3/builder_test.go`: `TestBuildOperation`, `TestBuildOperationID`, `TestNewWithValues`, `TestNoSecurityOverridesAPISecurity`, `TestNoSecurityOverridesServiceSecurity`, `TestSSEItemSchemaMatchesDataContract`, `TestSecuritySchemesIncludeVisibleOperationsOnly`, `TestStreamingResponseStatusCodes`
- `http/codegen/openapi/v3/description_ownership_test.go`: `TestSharedErrorComponentDescription`, `TestSharedErrorComponentLocalizedDescription`, `TestSharedErrorResponseDescriptions`, `TestUndescribedSharedErrorIgnoresLocalizedResponseDescription`
- `http/codegen/openapi/v3/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
