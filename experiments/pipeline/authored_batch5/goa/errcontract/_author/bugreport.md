# Bug report

Evaluating designs that inherit service or API errors into methods panics.
Expected: a method error may replace an inherited error only when it
generates the same service value contract — descriptions and examples are
ignored, types, validations, defaults and metadata must match, qualifier
differences are reported by name, and validation never mutates the evaluated
design. Got: panics whenever an error mapping needs the comparison.

Reproduce with:

```
go test -count=1 ./expr/ ./http/codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
