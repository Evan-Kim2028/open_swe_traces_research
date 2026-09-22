# Closure — goname

`codegen/funcs.go` + `codegen/goify.go` — identifier/comment text
transforms: SnakeCase, KebabCase, Goify, GoifyAtt, fixReservedGo,
Comment, Indent, WrapText, runeSpacePos(Rev).

Symbols stubbed: `SnakeCase`, `KebabCase`, `WrapText`, `Comment`,
`Indent`, `runeSpacePos`, `runeSpacePosRev`, `Goify`, `GoifyAtt`,
`fixReservedGo`.

Tests removed: `codegen/goify_test.go` deleted; casing/wrap tests
trimmed from `codegen/funcs_test.go` and `codegen/types_test.go`.
