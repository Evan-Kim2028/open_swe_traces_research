# Bug report

every buffered write/read panics or loses data: txn staging buffers cannot hold keys.

Reproduce with:

```
go test -count=1 ./internal/unionstore/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
