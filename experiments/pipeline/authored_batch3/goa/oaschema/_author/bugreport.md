# Bug report

OpenAPI generation panics when it copies or merges schemas or renders
examples. Expected: schema copies share no mutable state, merges fill
missing fields and tighten bounds, and examples drop hidden fields.
Got: panics.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
