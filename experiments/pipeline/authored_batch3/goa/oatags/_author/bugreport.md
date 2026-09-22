# Bug report

OpenAPI generation panics reading tag metadata and extensions or
marshaling schema objects. Expected: tags group by name with version-
gated fields, extensions merge swagger/openapi prefixes and decode JSON
values, and marshaling merges extension keys into output. Got: panics.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
