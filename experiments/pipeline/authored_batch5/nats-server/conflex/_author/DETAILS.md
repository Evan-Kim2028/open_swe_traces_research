# Details — conflex

1. Whitespace may precede a value but a newline before any value is an error
   ("Expected value but found new line"). Inferable: doc — the lexValue comment
   says exactly this.
2. Value dispatch: `[` array, `{` map, `'`/`"` quoted string, `-` negative
   number, `(` block string, digit → number/date/IP scan, `.` → error
   ("Floats must start with a digit"), anything else → bare string.
   Inferable: partially — the state-fn names list the arms but not the order
   or the `.` special case.
3. A token that starts with a digit becomes a STRING mid-scan if a rune that
   is neither a digit, `.`, `-`, number suffix, nor a legal terminator shows
   up (`12abc` → string "12abc"). Inferable: doc — the quadruple name hints,
   the reclassification rule does not.
4. `-` inside a digit-started token begins a date ONLY at offset 4 (`YYYY-`);
   the date must then match `YYYY-MM-DDThh:mm:ssZ` exactly — every literal
   position is checked and any mismatch errors. Inferable: partially — the
   Zulu-only rule is in the doc comment, the offset-4 gate is body-internal.
5. Floats need ≥1 digit before AND after `.`; `1.` is an error, `.5` is an
   error. Inferable: doc — comments state "at least one digit is required".
6. A second `.` inside a float re-routes to IP lexing, which accepts digits,
   `.`, `:`, `-` and emits itemString (`127.0.0.1:4222` → string).
   Inferable: no.
7. Number suffixes are `k K m M g G t T p P e E` (isNumberSuffix stays); after
   a suffix any run of `b/B/i/I` is still the same number (`1KiB` ok), and the
   number is emitted only if the NEXT rune is a newline, eof, `}`, `;`, `,`,
   whitespace, or a digit — otherwise the token falls back to string.
   Inferable: partially — the suffix set is visible, the b/i run rule and the
   terminator set are not (note `]` is NOT a terminator: `[1k]` yields the
   string "1k", not an integer).
8. Bare (unquoted) strings terminate on newline, eof, `;`, `,`, `]`, `}`,
   whitespace, or `'`; `\` inside a bare string still triggers escape
   processing. Inferable: partially — the terminator set is body-internal.
9. On bare-string termination the emission order is: escaped parts → string,
   else bool check, else `$` variable check, else string. Bools are six
   spellings case-insensitively: true/false/on/off/yes/no. `$`-variables emit
   itemVariable with the `$` STRIPPED from the value. Inferable: no.
10. Single-quoted strings are raw — no escape interpretation, end at `'`;
    double-quoted strings process escapes. Both terminate by emitting on the
    closing quote with the quote itself ignored. Inferable: doc — the
    lexDubQuotedString comment says "will not interpret any internal contents"
    is about the NON-escape parts; the raw-vs-escape split needs the bodies.
11. Escapes are exactly `\x` + two hex digits (emitted byte), `\t`, `\n`,
    `\r`, `\"`, `\\`; anything else errors, and a newline inside `\x` errors.
    Inferable: no.
12. `(...)` block strings capture raw text until `)` sitting on a line by
    itself — the `)` must be immediately preceded by `\n` and followed by
    `\n` or EOF; a `)` anywhere else is content. Inferable: doc — the comment
    states "on a new line by itself", the exact lookaround is body-internal.
13. Comments: `#` or `//` run to (not including) the newline, emitting
    itemCommentStart then itemText; a lone `/` NOT followed by `/` is silently
    swallowed in post-value positions at top level, acts like a `,`/newline
    between array values, but errors at an array value position. Inferable:
    no — the asymmetric fallthrough lives in the excised bodies.
14. After a top-level value only newline, eof, `;`, `,`, `}` (or a comment)
    are legal; anything else errors. Inferable: partially.
15. Inside arrays both `,` and bare newlines separate values (`[1\n2]` is two
    items) while a `,` AT a value position errors ("Unexpected array value
    terminator"); `;` is not an array separator. Inside maps `,`/`;`/newline
    all separate pairs. Inferable: no.
16. emitString concatenates any buffered escape parts plus the in-progress
    span, then clears the parts buffer; the emitted item records the line and
    a column offset that counts from the start of the logical line, not the
    raw position. Inferable: partially — emit (kept) shows the column rule,
    the parts-join is in the excised emitString.
