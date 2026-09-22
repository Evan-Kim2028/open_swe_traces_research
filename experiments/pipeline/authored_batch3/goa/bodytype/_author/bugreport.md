# Bug report

Endpoint finalization panics when computing request, streaming, or
response bodies. Expected: bodies subtract header/param/cookie
attributes, produce endpoint-specific generated types, and preserve
type metadata. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
