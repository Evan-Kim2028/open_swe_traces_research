# Bug report

Loading a design panics, or returns success for an invalid design. Wrong argument shapes and functions used in the wrong place produce no error. A design that records a problem during evaluation returns nothing the caller can report.

Reproduce with:

```
go test -count=1 ./eval/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
