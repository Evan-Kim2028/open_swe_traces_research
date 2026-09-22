# Bug report

`StringOrSet` JSON handling is broken: values constructed with `Set` marshal as a bare string
instead of an array, `["x"]` unmarshals then re-marshals as `"x"`, `String()` returns values in
nondeterministic order, and `Equal` compares order-sensitively.

Expected: `Set`-built and multi-element values marshal as arrays; a single-element `String`/`Of`
marshals as a bare string; array payloads remember they were arrays; `String()`/`Value()` are
sorted; `Equal` is order-insensitive.

Reproduce with:

```
go test -count=1 ./pkg/util/stringorset/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
