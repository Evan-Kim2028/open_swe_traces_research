# Bug report

Field-path parsing is broken: dotted paths, `[key]` map lookups, `[0]` indexes and `[*]` wildcards are rejected or produce the wrong element kinds; rendering a parsed path scrambles the spelling; equality and prefix matching report wrong answers — in particular a pattern containing a wildcard step no longer matches a concrete index.

Expected: `a.b[0].c`, `a[key]`, and `a[*]` parse; a parsed path renders back to its canonical spelling; `[*]` in a pattern matches `[0]` in a target when checking prefixes; equal-length paths compare elementwise.

Reproduce with:

```
go test -count=1 ./util/pkg/reflectutils/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
