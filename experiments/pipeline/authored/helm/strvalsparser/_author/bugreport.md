# Bug report

`--set`/`--set-string`/`--set-file`/`--set-literal` flags fail or panic at install/template/lint time: value overrides cannot be applied, or parsed values come out wrong (strings typed, lists malformed).

Reproduce with:

```
go test -count=1 ./pkg/strvals/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
