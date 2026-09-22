# Bug report

Planning OpenAPI documents panics. Expected: `openapi:versions` selects a
subset (default all three) in generation order, `openapi:path:<version>`
overrides output paths when the override is a clean relative extension-less
path, duplicate resolved paths and unknown version/path keys error. Got:
panics on any `Specs` call.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/ ./http/codegen/openapi/v2/ ./http/codegen/openapi/v3/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
