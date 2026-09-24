# Closure — unionid

`codegen/union.go` — union declaration/type identity: envelope keys,
branch shapes, length-prefixed encoding, recursion back-references.

Symbols stubbed: `NewUnionDeclarationID`, `NewUnionTypeID`,
`writeUnionTypeID`, `writeUnionAttributeID`, `writeUnionObjectID`,
`writeUnionIDPart`.

Tests removed: union identity/name-scope tests trimmed from
`codegen/scope_test.go`.
