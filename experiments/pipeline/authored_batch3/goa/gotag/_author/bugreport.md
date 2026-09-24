# Bug report

Type generation panics when naming primitive types or writing struct
field tags. Expected: primitives map to Go builtins, nilability drives
pointer choices, and field tags honor metadata overrides and required
fields. Got: panics.

Reproduce with:

```
go test -count=1 ./codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
