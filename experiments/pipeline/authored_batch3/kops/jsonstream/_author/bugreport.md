# Bug report

Re-emitting a decoded JSON token stream is broken: output is missing or malformed, objects and arrays are not indented, commas land in the wrong places (trailing commas after the last element or missing separators), string values are corrupted, and the current field path always comes back empty.

Expected: the token sequence `{`, `"a"`, `1`, `}` writes `{\n  "a": 1\n}`; nesting adds a two-space indent per level; no trailing comma appears before a closing bracket; the path inside a nested object reports its enclosing field names joined with `.`.

Reproduce with:

```
go test -count=1 ./pkg/jsonutils/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
