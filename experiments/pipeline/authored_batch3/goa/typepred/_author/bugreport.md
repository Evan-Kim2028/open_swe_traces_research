# Bug report

Evaluating designs that use user types, aliases, arrays, maps, or unions
panics during validation, attribute lookup, and example generation.
Expected: type predicates unwrap named types, structural comparison works
across differently-named but identical types, and qualified type names
render element types. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
