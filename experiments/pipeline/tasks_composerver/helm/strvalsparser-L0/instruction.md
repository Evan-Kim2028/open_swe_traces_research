# Bug report

`--set`/`--set-string`/`--set-file`/`--set-literal` flags fail or panic at install/template/lint time: value overrides cannot be applied, or parsed values come out wrong (strings typed, lists malformed).

Reproduce with:

```
go test -count=1 ./pkg/strvals/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
