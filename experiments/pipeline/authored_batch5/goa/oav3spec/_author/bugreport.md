# Bug report

Generating or rendering OpenAPI 3 documents panics inside the openapi v3
codegen package. Expected: `openapiv3.New`/`openapiv3.Files` build spec
trees that marshal cleanly — `x-*` extension keys merged at the top
level, `security: []` emitted for `NoSecurity` operations, examples
attached to parameters/headers/media types. Got: panics when spec
objects are marshaled or examples are initialized, so builder,
parameter, example, and facade tests fail.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/v3/ ./http/codegen/openapi/v2/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
