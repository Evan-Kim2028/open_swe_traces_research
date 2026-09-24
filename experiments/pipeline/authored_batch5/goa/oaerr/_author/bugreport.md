# Bug report

Building an OpenAPI error response panics. Expected: content type
resolved by response → result-type → application/json precedence, and the
generated error example stamped with name/temporary/timeout/fault only on
fields present in the projected body. Got: panics in every error-response
build.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/ ./http/codegen/openapi/v2/ ./http/codegen/openapi/v3/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
