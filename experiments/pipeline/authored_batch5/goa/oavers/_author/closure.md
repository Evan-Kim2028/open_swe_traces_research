# Closure — oavers

`http/codegen/openapi/versions.go` — OpenAPI document selection and
output-path planning: which specification versions to generate (2.0, 3.0, 3.2),
the default output path of each, `openapi:versions` / `openapi:path:<version>`
meta handling, and the path-validation rules (relative, clean, extension-less,
no duplicates).

Symbols stubbed: `Specs`, `selectedVersions`, `specPath`, `validatePathKeys`, `knownVersion`.

Tests removed (reach the stubs, verified by excision):
- `http/codegen/openapi/versions_test.go`: `TestSpecs`
