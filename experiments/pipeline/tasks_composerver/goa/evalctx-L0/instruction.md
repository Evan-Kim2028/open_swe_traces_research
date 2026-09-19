# Bug report

Evaluating a design panics, or runs languages in an order that ignores declared dependencies. Registering the same design language twice succeeds. Errors recorded during evaluation are missing from the run's result. Nested design functions cannot see the expression they are filling in.

Reproduce with:

```
go test -count=1 ./eval/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
