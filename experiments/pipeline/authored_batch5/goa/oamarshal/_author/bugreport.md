# Bug report

Marshaling any OpenAPI spec type panics. Expected: extension maps
merge into emitted JSON/YAML as `x-*` keys, large integers keep exact digits,
and marshal errors propagate. Got: panics in every spec type's Marshal
method.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/ ./http/codegen/openapi/v2/ ./http/codegen/openapi/v3/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
