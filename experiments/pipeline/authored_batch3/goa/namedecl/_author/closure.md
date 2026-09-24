# Closure — namedecl

`codegen/name_declaration.go` — package-level name declaration:
constructors, freeze-gated accessors, deterministic ordering,
declaration/order validation.

Symbols stubbed: `NewExactName`, `NewPreferredName`,
`(*NameDeclaration).Name`, `Kind`, `(*PackageNameKind).String`,
`newDependentName`, `comparePackageNames`, `comparePackageNameOrders`,
`validateNameDeclaration`, `(*PackageNameVisibility).valid`,
`validatePackageNameOrder`, `isStablePackageNameOrderType`,
`(*NameDeclaration).packagePath`, `preferredName`,
`(*PackageNameKind).valid`.

Tests removed: name-declaration tests trimmed from
`codegen/generated_types_test.go` (shared helper testNameOrder kept).
