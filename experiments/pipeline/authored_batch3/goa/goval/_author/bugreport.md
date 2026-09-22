# Bug report

Generating default values and fixtures panics for designs with
defaults, unions, or custom field types. Expected: defaults render as
typed Go expressions honoring custom types, pointer layouts, and union
constructors, deterministically. Got: panics.

Reproduce with:

```
go test -count=1 ./codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
