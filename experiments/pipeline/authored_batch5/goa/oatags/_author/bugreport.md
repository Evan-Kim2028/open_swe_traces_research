# Bug report

Extracting OpenAPI tag metadata panics. Expected: sorted, deduped
`Tag` records from `swagger:tag:`/`openapi:tag:` meta, desc/url/externalDocs
sub-keys, 3.2-only fields gated on version, `x-*` extensions merged in
marshaled output. Got: panics on any tag extraction or marshal.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/ ./http/codegen/openapi/v2/ ./http/codegen/openapi/v3/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
