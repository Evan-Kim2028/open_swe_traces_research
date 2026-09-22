# Bug report

Evaluating designs with routed endpoints panics when computing path and
query params or validating routes. Expected: wildcard params resolve
against the payload, paths join base paths correctly, and invalid
param/header shapes are reported. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
