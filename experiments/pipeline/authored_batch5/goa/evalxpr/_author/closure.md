# Closure — evalxpr

`eval/expression.go` — the DSL engine's smallest contract:
`DSLFunc.DSL` returns the stored DSL closure, `TopExpr.EvalName` names the
root expression in diagnostics, and `ToExpressionSet` converts a heterogeneous
slice into an `ExpressionSet` for `WalkSets` callbacks.

Symbols stubbed: `DSLFunc.DSL`, `TopExpr.EvalName`, `ToExpressionSet`.

Tests removed (reach the stubs, verified by excision):
- `eval/error_location_test.go`: `TestValidationErrorsWithoutLocation`
- `eval/eval_test.go`: `TestInvalidArgError`, `TestTooFewArgError`, `TestTooManyArgError`
- `eval/incompatible_dsl_context_test.go`: `TestIncompatibleDSLIncludesTypeContext`
