# Bug report

Rendering Swagger 2.0 documents panics inside the openapi v2 codegen
package. Expected: `openapiv2.Files` returns section templates that
execute cleanly and produce valid swagger JSON/YAML — extension keys
merged at the top level, `security: []` emitted for `NoSecurity`
operations. Got: panics when any spec object is marshaled, so every
fixture, isolation, and facade test that renders a document fails.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/v2/ ./http/codegen/openapi/v3/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
