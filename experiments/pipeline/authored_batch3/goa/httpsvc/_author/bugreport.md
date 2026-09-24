# Bug report

Evaluating designs with HTTP services panics — endpoint lookup, path
resolution, error inheritance, JSON-RPC route setup and service
finalization are all gone. Expected: services resolve full paths through
parents, inherit unmapped API errors, wire JSON-RPC routes, and default
paths. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
