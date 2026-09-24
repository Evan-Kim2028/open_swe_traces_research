# Bug report

Evaluating designs that declare errors panics during method finalization
and error merging. Expected: errors validate names, compare structurally
through their effective attributes, and copy without sharing mutable
state. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
