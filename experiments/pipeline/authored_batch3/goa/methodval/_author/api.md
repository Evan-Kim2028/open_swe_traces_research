# Exported API — methodval

```go
func (m *MethodExpr) Prepare()
func (m *MethodExpr) Validate() error
func (m *MethodExpr) Finalize()
func (m *MethodExpr) IsStreaming() bool
func (m *MethodExpr) IsPayloadStreaming() bool
func (m *MethodExpr) IsResultStreaming() bool
func (m *MethodExpr) HasMixedResults() bool
```

`MethodExpr` fields used: `Payload`, `StreamingPayload`, `Result`,
`StreamingResult`, `Requirements`, `ClientInterceptors`,
`ServerInterceptors`, `Errors`, `Stream`.

## Pre-existing callers

The eval engine runs `Prepare`/`Validate`/`Finalize` on every method.
Transport code reads the streaming predicates to choose encoder and
response shape.
