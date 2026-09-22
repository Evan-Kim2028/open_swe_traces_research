# Bug report

Building Swagger 2.0 documents panics inside the openapi v2 schema
builder. Expected: `openapiv2.Files`/`openapiv2.New` and
`BuildAttributeSchema` produce JSON schemas for every attribute, result
type, union, map, and parameter — with definitions collected per
document, validation rules copied onto schemas, and descriptions,
defaults, and examples preserved. Got: panics the moment any schema is
built, so path, parameter, validation, and fixture tests fail.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/v2/ ./http/codegen/openapi/v3/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
