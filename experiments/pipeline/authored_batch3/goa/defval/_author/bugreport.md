# Bug report

Evaluating designs that declare default values panics. Expected: authored
defaults are checked against the design type and its validation rules —
type compatibility, required fields, enum membership, bounds — before code
generation. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
