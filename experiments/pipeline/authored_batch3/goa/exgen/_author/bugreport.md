# Bug report

Generating examples for designs with validations panics. Expected:
examples satisfy declared length, enum, format, pattern, and min/max
constraints and stay stable across runs. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
