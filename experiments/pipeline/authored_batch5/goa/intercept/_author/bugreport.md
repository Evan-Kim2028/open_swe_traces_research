# Bug report

Evaluating a method that applies an interceptor panics. Expected:
interceptor access is validated against the method's payload, result, and
streaming shapes — object-only access, streaming compatibility, and field
existence after merging base types — collecting all violations as validation
errors. Got: panics during method validation.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
