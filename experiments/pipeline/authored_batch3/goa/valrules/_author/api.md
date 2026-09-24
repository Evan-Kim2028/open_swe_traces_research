# Exported API — valrules

```go
type ValidationExpr struct {
    Values           []any
    Format           ValidationFormat
    Pattern          string
    ExclusiveMinimum *float64
    Minimum          *float64
    ExclusiveMaximum *float64
    Maximum          *float64
    MinLength        *int
    MaxLength        *int
    Required         []string
}

func (v *ValidationExpr) Validate(ctx string, parent eval.Expression) *eval.ValidationErrors
func (v *ValidationExpr) Merge(other *ValidationExpr)
func (v *ValidationExpr) AddRequired(required ...string)
func (v *ValidationExpr) RemoveRequired(required string)
func (v *ValidationExpr) HasRequiredOnly() bool
func (v *ValidationExpr) Dup() *ValidationExpr
```

## Pre-existing callers

Attribute evaluation merges validations from bases/references and `Dup`
copies them into computed bodies; DSL evaluation calls `Validate` to
reject self-contradictory constraints.
