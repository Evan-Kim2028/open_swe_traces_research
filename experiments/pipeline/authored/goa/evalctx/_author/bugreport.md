# Bug report

Evaluating a design panics, or runs languages in an order that ignores declared dependencies. Registering the same design language twice succeeds. Errors recorded during evaluation are missing from the run's result. Nested design functions cannot see the expression they are filling in.

Reproduce with:

```
go test -count=1 ./eval/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
