# Exported API — resultview

```go
func NewResultTypeExpr(name, identifier string, fn func()) *ResultTypeExpr
func IsErrorResult(dataType DataType) bool
func CanonicalIdentifier(identifier string) string
func (rt *ResultTypeExpr) Dup(att *AttributeExpr) UserType
func (rt *ResultTypeExpr) Origin() UserType
func (rt *ResultTypeExpr) ID() string
func (rt *ResultTypeExpr) Rename(name string)
func (rt *ResultTypeExpr) View(name string) *ViewExpr
func (rt *ResultTypeExpr) HasMultipleViews() bool
func (rt *ResultTypeExpr) ViewHasAttribute(view, attr string) bool
func (rt *ResultTypeExpr) Finalize()
func Project(rt *ResultTypeExpr, view string) (*ResultTypeExpr, error)
func (v *ViewExpr) EvalName() string
```

## Pre-existing callers

`ResultType`/`CollectionOf` DSL, method finalize, response body and view
computation, and codegen's `ProjectedResultTypes` lookup all go through
these.
