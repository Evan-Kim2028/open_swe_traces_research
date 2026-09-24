# Exported API — intercept

Package-private surface reached through method validation:

```go
func (i *InterceptorExpr) EvalName() string
func (i *InterceptorExpr) validate(m *MethodExpr) *eval.ValidationErrors
func (i *InterceptorExpr) validateAttributeAccess(m *MethodExpr, source string, verr *eval.ValidationErrors, target *AttributeExpr, attr *AttributeExpr)
```

## Pre-existing callers

`MethodExpr.Validate` runs each interceptor's `validate`; `EvalName` feeds
error messages through `eval.Expression` naming.
