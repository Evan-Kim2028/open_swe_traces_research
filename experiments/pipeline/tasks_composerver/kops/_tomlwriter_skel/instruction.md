# Bug report

TOML rendering panics or no longer matches the previous byte-stable output: keys are unsorted, tables appear before scalars, quoting/escaping of empty keys and control characters is wrong, and setting a leaf through a scalar path or overwriting a table misbehaves.

Reproduce with:

```
go test -count=1 ./pkg/tomlwriter/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
