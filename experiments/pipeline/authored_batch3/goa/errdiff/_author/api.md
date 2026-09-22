# Exported API — errdiff

```go
func (e *ErrorExpr) EvalName() string
func (e *ErrorExpr) Validate() *eval.ValidationErrors
func (e *ErrorExpr) Finalize()
func (e *ErrorExpr) Dup() *ErrorExpr
func (e *ErrorExpr) effectiveAttribute() *AttributeExpr  // (unexported shape)
```

Error contracts compare evaluated error expressions for equivalence and
copy them between method, service, and API scope.

## Pre-existing callers

`MethodExpr` merges service-level and method-level errors, transport
generators read the finalized error attribute, and error equivalence
decides whether the same `ErrorResult` is reused across methods.
