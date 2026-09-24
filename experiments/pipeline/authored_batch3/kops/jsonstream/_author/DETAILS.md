# Details — jsonstream

1. `Path()` returns the enclosing field names joined by `.`. Inferable: yes.
2. `{` and `[` push their state and grow the indent by two spaces; `}` and `]` pop and shrink it. Inferable: yes.
3. Element separation is a deferred comma: `,\n` is buffered and only flushed before the next token, and is rewritten to `\n` when the container closes — no trailing comma. Inferable: partially — the deferred-write scheme is a choice.
4. A scalar token in object state becomes a field name: it emits `indent + "name": `, pushes an `F` state, and records the name on the path stack. Inferable: partially.
5. A scalar in `F` state is the field value: it emits raw, pops `F` and the path entry, then defers the comma. Inferable: partially.
6. A `}`/`]` arriving while `F` is still on the stack pops `F` and the path entry too. Inferable: no — subtle stack detail.
7. `float64` prints `%g`, `json.Number` prints verbatim, bool `%v`, nil `null`; strings are wrapped in quotes with NO escaping of the contents. Inferable: no — verbatim quoting is arbitrary.
8. A scalar at top level (empty state) is an error, not a bare write. Inferable: no.
9. An unrecognized delimiter or token type, and an unhandled state combination, each return an error rather than writing. Inferable: partially.
10. Field-name emission uses `fmt.Sprintf("%s", token)` for the path entry — a `float64` field name lands on the path in numeric form. Inferable: no.
