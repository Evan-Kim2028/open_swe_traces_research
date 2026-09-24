# Bug report

Service planning and transport generators panic when they snapshot a design
attribute graph. Expected: generators keep a private deep copy of the
attribute graph they mutate — shared and recursive nodes stay shared in the
copy, result-type views stay attached to their copied parent, and each
copied node can be resolved back to the exact input node it came from. Got:
panics as soon as an attribute copy is requested.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
