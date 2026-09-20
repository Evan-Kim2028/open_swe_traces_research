# Bug report

Every design evaluation panics — the root expression no longer walks its
expression sets, validates the design, or finalizes servers and errors.
Expected: the root walks API, types, services, methods and transports in
dependency order and reports design-level errors. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
