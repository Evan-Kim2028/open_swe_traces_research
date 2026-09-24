# Exported API — evalxpr

```go
func (f DSLFunc) DSL() func()
func (t TopExpr) EvalName() string
func ToExpressionSet(slice any) ExpressionSet
```

## Pre-existing callers

`ToExpressionSet` is how every DSL root packages sub-expressions for the
engine (`expr/*.go` WalkSets implementations); `DSLFunc` embeds deferred DSL
in design expressions; `TopExpr` names the synthetic root in error reports.
