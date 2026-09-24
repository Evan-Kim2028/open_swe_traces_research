# Bug report

Evaluating designs that declare services with errors panics. Expected:
services validate their errors, inline method errors sharing a name keep
one contract, error types carry the error name in a required string field,
and generated method error types reuse earlier origins. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
