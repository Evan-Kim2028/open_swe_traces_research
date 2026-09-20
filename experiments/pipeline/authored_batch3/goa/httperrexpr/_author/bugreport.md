# Bug report

Evaluating designs that declare HTTP error responses panics. Expected:
error responses validate their status code, headers and mapped attributes
against the declared error type; JSON-RPC designs reject reserved codes and
header/cookie mappings; finalization wires up bodies and defaults. Got:
panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
