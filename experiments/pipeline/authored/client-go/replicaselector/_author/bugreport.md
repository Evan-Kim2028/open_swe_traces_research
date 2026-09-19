# Bug report

reads/sends always hit the first replica or panic: follower reads, stale reads, and leader failover never happen; after a store dies, requests keep targeting it.

Reproduce with:

```
go test -count=1 ./internal/locate/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
