# Closure — typepred

`expr/types.go` — core datatype predicates and equality: As*/Is* unwrap
helpers, IsAlias, Equal, QualifiedTypeName, toReflectType.

Symbols stubbed: `AsObject`, `AsArray`, `AsMap`, `AsUnion`, `IsObject`,
`IsArray`, `IsMap`, `IsUnion`, `IsPrimitive`, `IsAlias`, `Equal`,
`QualifiedTypeName`, `toReflectType`.

Tests removed: `expr/project_test.go` deleted (package-level vars call the
stubbed predicates at init). `TestEqual` trimmed from
`expr/equal_test.go`; predicate tests trimmed from `expr/types_test.go`.
