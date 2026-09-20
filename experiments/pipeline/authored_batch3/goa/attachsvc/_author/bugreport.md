# Bug report

Re-evaluating a subset of an existing design's services panics. Expected:
the selected services, methods, types and their HTTP, JSON-RPC and gRPC
transport expressions are collected, prepared, validated and finalized —
using the design that owns them. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
