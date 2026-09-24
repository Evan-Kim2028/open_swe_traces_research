# Exported API — svcerrors

```go
func (s *ServiceExpr) Method(n string) *MethodExpr
func (s *ServiceExpr) EvalName() string
func (s *ServiceExpr) Error(name string) *ErrorExpr
func (s *ServiceExpr) Hash() string
func (s *ServiceExpr) Validate() error
func (s *ServiceExpr) Finalize()
func (e *ErrorExpr) Validate() error
func (e *ErrorExpr) Finalize()
```

## Pre-existing callers

Method/error lookups back every transport resolver; the eval engine runs
service Validate/Finalize; codegen consumes generated error types and
error-name metadata.
