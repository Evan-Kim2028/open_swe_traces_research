# Contract (L2) — tomlwriter

A write-only TOML tree. Nested tables are created by walking a key path; if a path element already holds a scalar, walk stays in the current table (go-toml v1). SetPath accepts only string, int64, and bool (otherwise panic) and writes the last element as a leaf, replacing a table if one was there.

Serialization: at each table, emit scalar keys in lexicographic order as `key = value`, then subtables. Each subtable is preceded by a blank line and a `[dotted.path]` header; children are indented two spaces more. Bare keys are letters, digits, `_`, `-`. Empty key is quoted `""`. A key that already starts and ends with `"` is emitted unchanged. Other keys are quoted with basic-string escapes: `\b \t \n \f \r \" \\` and `\u00XX` (uppercase hex) for other controls < 0x1F. Strings are always basic strings. Bools are `true`/`false`; ints are base-10.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestEmptyTree` | empty document is empty string |
| `TestScalarsAndTables` | scalars before subtables, sorted keys |
| `TestTableAtDocumentStart` | table header at root |
| `TestKeyQuotingAndStringEscaping` | quoting and escape sequences |
| `TestPreQuotedKeyPassthrough` | already-quoted keys not re-escaped |
| `TestEmptyKeyIsQuoted` | empty key is `""` |
| `TestEscapeBoundaries` | control-char `\u` form |
| `TestSetPathThroughScalar` | scalar path element does not descend |
| `TestSetPathOverwritesTable` | leaf replaces table |
| `TestSetPathRejectsUnsupportedType` | non string/int64/bool panics |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./pkg/tomlwriter/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestTomlEmptyAndRootTableProperty`, `TestTomlQuotingContractProperty`, `TestTomlSetPathBehaviorProperty`, `TestTomlSetPathUnsupportedTypeProperty`, `TestTomlScalarsOrderProperty`, `TestTomlUnseenRandomProperty`: TestTomlEmptyAndRootTableProperty

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
