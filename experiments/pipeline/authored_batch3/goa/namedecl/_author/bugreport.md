# Bug report

Generated-package name planning panics when services declare types,
functions, constants, or variables. Expected: declarations validate
kind/visibility/order, compare deterministically, and expose final
names only after freeze. Got: panics.

Reproduce with:

```
go test -count=1 ./codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
