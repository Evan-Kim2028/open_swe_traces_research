# Bug report

Evaluating designs with field validations panics, and merging inherited
validations or copying attributes panics too. Expected: bound conflicts
are reported, merges keep the tightest constraint, and copies don't
share mutable slices. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
