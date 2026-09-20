# Details — tfliterals

1. `LiteralProperty` renders `<type>.<name>.<prop>` with the name run through the package's name sanitizer; `LiteralData` renders the same prefixed by `data.`. Inferable: yes — `sanitizeName` is intact in the same package.
2. `LiteralSelfLink` is `LiteralProperty(type, name, "self_link")`. Inferable: yes.
3. `LiteralFunctionExpression` renders `fn(arg, arg)` with `, ` separators. Inferable: yes.
4. `LiteralListExpression` renders `[a, b]` with `, ` separators. Inferable: yes.
5. `LiteralFromStringValue` wraps the value in quotes verbatim — the content is NOT escaped, so a `"` in the input corrupts the output. Inferable: no — unescaped emission is arbitrary.
6. `LiteralFromIntValue` renders `%d`; `LiteralTokens` dot-joins the tokens. Inferable: yes.
7. `LiteralWithIndex` renders `"<s>-${count.index}"` — the literal suffix text inside quotes. Inferable: partially — the `count.index` spelling is a terraform convention.
8. `LiteralBinaryExpression` renders `lhs <op> rhs` space-separated; `LiteralIndexExpression` renders `coll[idx]` tight. Inferable: yes.
9. `LiteralEmptyStrConditionalExpression` renders `<e> == "" ? null : <v>`. Inferable: partially — ternary spelling.
10. `Write` emits `= <String>\n` and ignores both `indent` and `key` arguments. Inferable: no — ignored parameters are arbitrary.
11. `MarshalJSON` marshals the String field (so the literal serializes as a bare JSON string). Inferable: partially.
12. `dedupLiterals` sorts the input in place (a documented side effect) and drops adjacent duplicates; nil in, nil out. Inferable: doc — the doc comment states the sorted side effect.
13. `SortLiterals` sorts ascending by `.String` in place. Inferable: yes.
