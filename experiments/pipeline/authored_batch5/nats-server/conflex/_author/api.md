# Exported API — conflex

Package `conf` (module `example.internal/msgkit/v2`) — lexer for the config file
format, feeding the kept parser in `parse.go`.

Surface: `lex(input string) *lexer`, `(lx *lexer) nextItem() item` — pull-based
item stream; `item{typ itemType, val string, line, pos int}`; item kinds
`itemError itemEOF itemKey itemText itemString itemBool itemInteger itemFloat
itemDatetime itemArrayStart itemArrayEnd itemMapStart itemMapEnd
itemCommentStart itemVariable itemInclude`. The excised closure is the
value-side of the grammar: value dispatch, arrays, quoted/raw/block strings,
escapes, integers/floats/dates/IPs/convenient numbers, booleans, `$` variables,
comments, and post-value terminators.

Callers: `parse.go` (same package) drives `lex`/`nextItem`; `parse_test.go`
retained. In-tree tests removed: `lex_test.go` (66 tests).
