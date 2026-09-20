# Bug report

Evaluating designs that declare security schemes panics, and helpers that
name or copy schemes either panic or return wrong values when reached from
generated-code paths. Expected: schemes and flows report stable type and
error names, flow URLs are checked, requirement copies keep a link to the
declared scheme, and an explicit opt-out disables requirements. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
