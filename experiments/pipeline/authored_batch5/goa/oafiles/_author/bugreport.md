# Bug report

Rendering an OpenAPI document panics. Expected: two files per document
(`<path>.json`, `<path>.yaml`), JSON honoring `openapi:json:*` formatting meta,
YAML quoting date-shaped string scalars so they stay strings. Got: panics in
the file template funcs.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/ ./http/codegen/openapi/v2/ ./http/codegen/openapi/v3/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
