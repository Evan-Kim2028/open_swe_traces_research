# Exported API — grpcend

```go
func (e *GRPCEndpointExpr) Name() string
func (e *GRPCEndpointExpr) Description() string
func (e *GRPCEndpointExpr) EvalName() string
func (e *GRPCEndpointExpr) Prepare()
func (e *GRPCEndpointExpr) LegacyStreamCompat() bool
func (e *GRPCEndpointExpr) Validate() error
func (e *GRPCEndpointExpr) Finalize()
```

`GRPCEndpointExpr` fields used: `Request`, `StreamingRequest`, `Metadata`,
`Response`, `GRPCErrors`, `Requirements`, `Meta`, `MethodExpr`, `Service`.

## Pre-existing callers

The eval engine runs Prepare/Validate/Finalize on every gRPC endpoint;
codegen reads LegacyStreamCompat to emit dual-protocol stream servers.
