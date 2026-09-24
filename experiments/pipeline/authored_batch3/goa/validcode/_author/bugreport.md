# Bug report

Generating validation code panics for designs that declare field
validations. Expected: generated validators check each declared
constraint with correct nil-guards, alias handling, and error paths.
Got: panics.

Reproduce with:

```
go test -count=1 ./codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
