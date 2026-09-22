# Closure — oafiles

`http/codegen/openapi/files.go` — emission of the OpenAPI document
as codegen files: `Files` builds the JSON and YAML `codegen.File`s at
`gen/<path>.{json,yaml}` driven by one section template each, `toJSON` honors
the `openapi:json:prefix`/`openapi:json:indent` formatting meta, and `toYAML`
post-processes output so date-shaped string scalars stay strings.

Symbols stubbed: `Files`, `toJSON`, `toYAML`, `quoteDateShapedYAMLStrings`.

Tests removed (reach the stubs, verified by excision):
- `http/codegen/openapi/files_test.go`: `TestToYAMLQuotesDateShapedStrings`
- `http/codegen/openapi/v2/files_test.go`: `TestExtensions`, `TestValidations`
- `http/codegen/openapi/v2/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
- `http/codegen/openapi/v3/public_api_test.go`: `TestDefaultFacadeMatchesWithValues`
