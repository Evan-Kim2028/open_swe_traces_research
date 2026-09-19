# Bug report

reads/sends always hit the first replica or panic: follower reads, stale reads, and leader failover never happen; after a store dies, requests keep targeting it.

Reproduce with:

```
go test -count=1 ./internal/locate/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
