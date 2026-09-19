# Bug report

locating any key fails: no cached region ever matches, every lookup misses or panics.

Reproduce with:

```
go test -count=1 ./internal/locate/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
