# Bug report

Evaluating designs that declare service methods panics — payloads,
results, security requirements, errors and interceptors never validate or
finalize. Expected: methods check their security attributes and
requirements, merge interceptors across levels, inherit errors and
requirements, and report streaming behavior correctly. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
