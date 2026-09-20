# Bug report

Evaluating designs that declare HTTP responses panics, and endpoints that
inherit or copy responses panic too. Expected: responses validate their
status, headers, cookies and body against the method result; response
finalization wires up defaults and unmapped attributes; copies share
nothing mutable. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
