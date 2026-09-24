# Bug report

Evaluating designs with gRPC endpoints panics during response
preparation, validation, and inheritance. Expected: responses validate
their message/header/trailer shapes against the method result and copy
cleanly. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
