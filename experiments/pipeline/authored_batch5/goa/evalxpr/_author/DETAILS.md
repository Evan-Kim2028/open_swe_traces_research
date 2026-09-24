# Commitments — evalxpr

1. `DSLFunc.DSL` returns the receiver — the embedded function is itself the deferred DSL. In-tree coverage: indirect: DSL execution in trimmed tests. Inferable: yes.
2. `ToExpressionSet` returns nil for nil, panics on non-slice input, and otherwise converts each element to `Expression` preserving order. In-tree coverage: `TestInvalidArgError`, `TestTooFewArgError`, `TestToExpressionSet` (trimmed). Inferable: partially — the panic-on-non-slice contract is stated ("bug"); element conversion is mechanical.
3. `TopExpr.EvalName` yields a stable human name for the synthetic top expression. In-tree coverage: error-format tests (trimmed). Inferable: no — exact spelling is arbitrary; shape asserted only.
