# Closure — conflex

Package: `conf`. File: `lex.go`.

Removed (30 funcs stubbed): `lexer.emitString`, `lexer.addCurrentStringPart`,
`lexer.addStringPart`, `lexer.hasEscapedParts`, `lexer.isBool`, `lexer.isVariable`,
`lexTopValueEnd`, `lexBlockValueEnd`, `lexMapValueEnd`, `lexValue`,
`lexArrayValue`, `lexArrayValueEnd`, `lexArrayEnd`, `lexQuotedString`,
`lexDubQuotedString`, `lexString`, `lexBlock`, `lexStringEscape`,
`lexStringBinary`, `lexNumberOrDateOrStringOrIPStart`, `lexNumberOrDateOrStringOrIP`,
`lexConvenientNumber`, `lexDateAfterYear`, `lexNegNumberStart`, `lexNegNumber`,
`lexFloatStart`, `lexFloat`, `lexIPAddr`, `lexCommentStart`, `lexComment`.

Kept: `itemType`/`item`/`lexer` types and all item-type constants, `lex`,
`nextItem`, `push`/`pop`, `emit`, `next`/`ignore`/`backup`/`peek`, `errorf`,
`lexSkip`, the key/map/include/top block states (`lexTop`, `lexBlockStart`,
`lexBlockEnd`, `lexKeyStart`, `lexKey`, `lexKeyEnd`, `lexDubQuotedKey`,
`lexQuotedKey`, `keyCheckKeyword`, `lexInclude*`, `lexMapKey*`, `lexMapEnd`),
character classes (`isNumberSuffix`, `isKeySeparator`, `isWhitespace`, `isNL`),
`escapeSpecial`, `itemType.String`, `item.String`, all doc comments.
Import `encoding/hex` blanked.

Tests deleted: `lex_test.go` only.
