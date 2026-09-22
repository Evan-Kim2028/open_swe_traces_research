# Contract — conflex

The config lexer turns input text into a typed item stream via `lex` and
`nextItem`. Every commitment below is covered by a hidden test; every
hidden test maps to a commitment.

## Commitments

1. **Value position.** Whitespace may precede a value, but a newline
   before the value produces an error mentioning the new line. Covered
   by `TestDetail01`.
2. **Dispatch.** The first rune of a value selects its lexer: `[`
   array, `{` map, `'`/`"` quoted string, `-` negative number, `(`
   block string, digit number/date/IP scan, `.` an error, anything else
   a bare string. Covered by `TestDetail02`.
3. **Digit-started strings.** A token that starts with a digit becomes
   a string mid-scan if it meets a rune that is neither a digit, `.`,
   `-`, number suffix, nor a legal terminator. Covered by
   `TestDetail03`.
4. **Dates (shape).** A digit token containing `-` enters a strict
   Zulu datetime path: a full `YYYY-MM-DDThh:mm:ssZ` emits a datetime
   item; any literal-position mismatch or a non-Zulu zone errors.
   Covered by `TestDetail04`.
5. **Floats.** A float requires at least one digit before and after the
   `.`; `1.` and `.5` both error. Covered by `TestDetail05`.
6. **IP strings (shape).** A second `.` inside a float re-routes to IP
   lexing — digits, `.`, `:`, `-` accepted — and emits a string item.
   Covered by `TestDetail06`.
7. **Convenient numbers.** Number suffixes plus a b/B/i/I run stay
   numbers only when the next rune is a real terminator; `]` is not
   one, so `[1k]` yields the string `1k`, and a trailing letter drops
   the token to a string. Covered by `TestDetail07`.
8. **Bare-string terminators.** Bare strings end on newline, eof, `;`,
   `,`, `]`, `}`, whitespace, or `'`, and a backslash inside a bare
   string triggers escape processing. Covered by `TestDetail08`.
9. **Emission order (shape).** A bare string with escape parts emits a
   string without consulting the bool or variable tables; otherwise the
   six bool spellings (true/false/on/off/yes/no, case-insensitive)
   emit bools, and `$x` emits a variable with the `$` stripped.
   Covered by `TestDetail09`.
10. **Quoting.** Single-quoted strings are raw — no escape
    interpretation — while double-quoted strings process escapes; the
    closing quote is not part of the value. Covered by `TestDetail10`.
11. **Escapes (shape).** Only `\x` plus two hex digits, `\t`, `\n`,
    `\r`, `\"` and `\\` are legal; anything else — and a newline inside
    `\x` — produces an error item. Covered by `TestDetail11`.
12. **Block strings (shape).** `(...)` captures raw text until a `)`
    on a line by itself — preceded by `\n`, followed by `\n` or EOF; a
    `)` anywhere else is content and the block runs to EOF as an error.
    Covered by `TestDetail12`.
13. **Comments (shape).** `#` and `//` run to the newline emitting a
    comment-start then text item; a lone `/` is swallowed in post-value
    positions at top level, separates array values, and errors at an
    array value position. Covered by `TestDetail13`.
14. **Post-value terminators.** After a top-level value only newline,
    eof, `;`, `,`, `}`, or a comment are legal; anything else errors.
    Covered by `TestDetail14`.
15. **Array/map separators (shape).** Inside arrays `,` and bare
    newlines separate values while `,` at a value position errors and
    `;` does not separate; inside maps `,`, `;`, and newlines all
    separate pairs. Covered by `TestDetail15`.
16. **Emission bookkeeping.** Buffered escape parts concatenate with
    the in-progress span into one item; the recorded column counts from
    the start of the logical line. Covered by `TestDetail16`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | doc |
| TestDetail04 | 4 | partially — date shape only; the exact `-` offset gate is not pinned |
| TestDetail05 | 5 | doc |
| TestDetail06 | 6 | no — shape only |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | no — shape only |
| TestDetail10 | 10 | doc |
| TestDetail11 | 11 | no — shape only |
| TestDetail12 | 12 | doc — shape of the lookaround |
| TestDetail13 | 13 | no — shape only |
| TestDetail14 | 14 | partially |
| TestDetail15 | 15 | no — shape only |
| TestDetail16 | 16 | partially |
