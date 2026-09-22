# Bug report

Any spec output carrying `x-*` extensions panics. Expected:
`swagger:extension:` and `openapi:extension:` meta merge into extension maps
of JSON-typed values keyed by `x-` names, plus `x-goa-idempotent` on
idempotent methods. Got: panics on extension extraction.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/ ./http/codegen/openapi/v2/ ./http/codegen/openapi/v3/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
