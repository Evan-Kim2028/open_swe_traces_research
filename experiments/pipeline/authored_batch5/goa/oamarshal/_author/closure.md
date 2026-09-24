# Closure — oamarshal

`http/codegen/openapi/marshal.go` — the shared extension-aware
encoders: `MarshalJSON` merges `x-*` extension keys into a value's JSON object
and preserves large integers via `json.Number`; `MarshalYAML` produces the
extension-merged YAML representation consumed by every spec type's
`MarshalYAML`.

Symbols stubbed: `MarshalJSON`, `MarshalYAML`.

Tests removed (reach the stubs, verified by excision):
- `http/codegen/openapi/marshal_test.go`: `TestMarshalJSONPreservesLargeIntegers`
- `http/codegen/openapi/v2/build_isolation_test.go`: `TestBuildsKeepDefinitionsSeparate`
- `http/codegen/openapi/v2/builder_test.go`: `TestNoSecurityOverridesAPISecurity`, `TestNoSecurityOverridesServiceSecurity`, `TestOperationSecurityMarshal`
- `http/codegen/openapi/v2/files_test.go`: `TestExtensions`, `TestValidations`
- `http/codegen/openapi/v2/json_schema_union_test.go`: `TestBuildAttributeSchemaKeepsDefinitionsSeparate`
- `http/codegen/openapi/v2/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
- `http/codegen/openapi/v3/builder_test.go`: `TestNoSecurityOverridesAPISecurity`, `TestNoSecurityOverridesServiceSecurity`, `TestOperationSecurityMarshal`
- `http/codegen/openapi/v3/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
