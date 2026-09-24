# Closure — exgen

`expr/example.go` — validation-aware example generation: constraint
priority, retry-until-satisfied loop, format/pattern/minmax producers.

Symbols stubbed: `(*AttributeExpr).Example`, `NewLength`, the five
`has*Validation` predicates, `byLength`, `byEnum`, `byFormat`,
`byPattern`, `patgen`, `byMinMax`, `checkPattern`, `checkMinMaxValue`.

Tests removed: `expr/example_test.go`, `expr/user_type_example_test.go`,
`expr/example_stability_test.go` deleted.
