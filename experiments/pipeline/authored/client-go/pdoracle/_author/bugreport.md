# Bug report

timestamps never arrive: timestamp futures never complete / every ts request errors.

Reproduce with:

```
go test -count=1 ./oracle/...
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
