# Contract (L2) — strvalsparser

A `k=v,k2=v2` assignment string is parsed into a nested map. Dotted keys create nested maps; `name[i]` indexes into (creating and growing) a list; `name[i]` on an existing scalar errors, duplicate index assignment errors, and `name[]={a,b}` creates a list literal while `{a,b}` inside a value creates a list element. Backslash escapes the next character literally, including inside keys; `name=null` sets a nil value. Values are type-inferred (booleans, integers) unless parsed in string mode, in which case they stay strings; the literal parser never infers and treats escape syntax more literally. Nested name levels are capped (a fourth nested level errors). File mode reads each value from a caller-supplied rune reader, e.g. for `--set-file`. Lists auto-grow to the requested index, padding with nils, and index assignment returns the grown list. ToYAML renders the parsed map as YAML. Parse errors report the offending position/key.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `FuzzParse` | no panic on arbitrary input |
| `TestParseLiteral` | literal mode keeps strings |
| `TestParseLiteralInto` | literal merge into dest |
| `TestParseLiteralNestedLevels` | nested level cap in literal mode |
| `TestSetIndex` | list grow/index assignment |
| `TestParseSet` | set parsing incl. lists/escapes |
| `TestParseInto` | merge into existing map |
| `TestParseIntoString` | string mode: no inference |
| `TestParseJSON` | JSON value mode |
| `TestParseFile` | values via rune reader |
| `TestParseIntoFile` | file mode into dest |
| `TestToYAML` | yaml rendering |
| `TestParseSetNestedLevels` | nested level cap |
