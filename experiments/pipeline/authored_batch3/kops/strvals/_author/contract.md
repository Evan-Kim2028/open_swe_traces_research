# Contract (L2) — strvals

Parse a `--set`-style assignment string (`a=b`, `a.b=c`, `a[0]=b`, `a{b,c}=d`) into nested maps/lists and merge them into a destination map. The parser distinguishes bare keys, bracketed indices, and brace lists; values may be typed via `val` (`true`/`false`/numbers/strings) and lists append rather than overwrite unless an index is given. Errors on malformed input are descriptive parse errors, not panics. Merging respects existing map structure — a scalar slot is overwritten, a map slot merges.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestParseInto` | dotted/indexed/braced assignments build nested maps and lists |
| `TestParseIntoString` | same structure when the destination is preallocated |
| `TestParseIntoOverwrites` | a scalar key overwritten by a later assignment replaces, does not merge |
| `TestParseIntoErrors` | malformed syntax produces parse errors, not panics |
