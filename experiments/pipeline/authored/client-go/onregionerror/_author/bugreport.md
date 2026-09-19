# Bug report

any request keeps failing: sends return a generic error immediately (or panic) instead of retrying on the right peer; the first region error (e.g. not-leader) is not retried and surfaces to the caller.

Reproduce with:

```
go test -count=1 ./internal/locate/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
