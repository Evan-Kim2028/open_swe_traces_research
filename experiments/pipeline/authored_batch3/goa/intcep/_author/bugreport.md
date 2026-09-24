# Bug report

Evaluating designs that attach interceptors to methods panics during
service evaluation. Expected: interceptor attribute access is checked
against the intercepted type and evaluation completes. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
