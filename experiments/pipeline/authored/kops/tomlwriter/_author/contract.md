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
