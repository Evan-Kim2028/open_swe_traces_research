# Bug report

OpenAPI builds carrying value overrides panic. Expected: `Values`
stores alternate titles/descriptions/examples immutably, keyed by authored
attribute, materializing examples with translated descriptions and
deep-copied values. Got: panics on any build reading or writing values.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/ ./http/codegen/openapi/v2/ ./http/codegen/openapi/v3/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
