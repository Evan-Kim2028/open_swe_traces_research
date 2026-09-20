# Bug report

Evaluating designs that declare result types or views panics. Expected:
result types get identifiers, views are discovered and defaulted, methods
project results to a chosen view, and copies preserve their declaration
origin. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
