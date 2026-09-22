# Exported API — grpcresp

```go
type GRPCResponseExpr struct {
    StatusCode  int
    Description string
    Message     *AttributeExpr
    Headers     *MappedAttributeExpr
    Trailers    *MappedAttributeExpr
    Tag         [2]string
    Parent      eval.Expression
    Meta        MetaExpr
}

func (r *GRPCResponseExpr) EvalName() string
func (r *GRPCResponseExpr) Prepare(e *GRPCEndpointExpr)
func (r *GRPCResponseExpr) Validate(e *GRPCEndpointExpr) *eval.ValidationErrors
func (r *GRPCResponseExpr) Finalize(e *GRPCEndpointExpr)
func (r *GRPCResponseExpr) Dup() *GRPCResponseExpr
```

## Pre-existing callers

The eval engine runs Prepare/Validate/Finalize on every gRPC response.
Endpoint prepare clones responses via Dup and builds response messages
from the finalized Message/Headers/Trailers.
