# Bug report

Generated-name scoping panics when methods use OneOf union types.
Expected: union identities distinguish branches, packages, and field
shapes deterministically so equal unions share declarations and
different ones do not. Got: panics.

Reproduce with:

```
go test -count=1 ./codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
