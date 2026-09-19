# Bug report

client cannot dial or panics on first request; connections never reconnect.

Reproduce with:

```
go test -count=1 ./internal/client/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
