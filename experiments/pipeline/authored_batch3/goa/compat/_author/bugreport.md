# Bug report

Evaluating designs that declare defaults, enum values, or union types
panics when the DSL checks whether authored values match their declared
types. Expected: compatibility checks recurse through composite values
and unions read their envelope keys. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
