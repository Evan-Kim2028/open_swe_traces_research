# Bug report

Generating OpenAPI 3 documents panics inside the v3 body-schema builder.
Expected: `openapiv3.New`/`Files` produce `components/schemas` with one
component per unique type structure, view-projected response bodies,
validation/default/example decoration, and SSE `itemSchema` for
streaming endpoints. Got: panics the moment `buildBodyTypes` runs, so
builder, parameter, description-ownership, and fixture tests fail.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/v3/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
