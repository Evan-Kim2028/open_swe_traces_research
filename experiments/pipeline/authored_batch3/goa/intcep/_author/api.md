# Exported API — intcep

```go
type InterceptorExpr struct { /* evaluated interceptor application */ }

func (i *InterceptorExpr) EvalName() string
func (i *InterceptorExpr) validateAttributeAccess(...) // unexported validation
func (i *InterceptorExpr) Prepare() / Validate() / Finalize() // as stubbed
```

## Pre-existing callers

Service and method evaluation run interceptor Prepare/Validate/Finalize
for every `Interceptor` DSL call; codegen reads the finalized
interceptor attribute to emit read/write wrappers.
