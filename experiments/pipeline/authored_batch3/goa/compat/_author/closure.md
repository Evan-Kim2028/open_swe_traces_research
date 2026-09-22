# Closure — compat

`expr/types.go` — runtime value/type compatibility predicates and union
envelope keys.

Symbols stubbed: `(*Primitive).IsCompatible`, `(*Array).IsCompatible`,
`(*Object).IsCompatible`, `(*Map).IsCompatible`, `(*Union).IsCompatible`,
`(*Union).GetTypeKey`, `(*Union).GetValueKey`.

Tests removed: compatibility and union-key tests trimmed from
`expr/types_test.go`. Disjoint from typepred which stubs a different
symbol set in the same file.
