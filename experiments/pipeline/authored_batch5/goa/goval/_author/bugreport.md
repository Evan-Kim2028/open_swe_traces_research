# Bug report

Generating client/CLI code for services whose designs carry authored default
values panics inside the codegen package. Expected: authored defaults render
as Go expressions matching the planned field layout — named types, pointer
fields, unions, maps and `any` values all handled — so generated packages
compile. Got: panics when any default value is rendered.

Reproduce with:

```
go test -count=1 ./codegen/ ./http/codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
