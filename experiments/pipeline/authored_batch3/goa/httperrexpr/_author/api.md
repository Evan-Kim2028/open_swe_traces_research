# Exported API — httperrexpr

```go
type HTTPErrorExpr struct {
    *ErrorExpr
    Name     string
    Response *HTTPResponseExpr
}

func (e *HTTPErrorExpr) EvalName() string
func (e *HTTPErrorExpr) IsJSONRPC() bool
func (e *HTTPErrorExpr) Validate() *eval.ValidationErrors
func (e *HTTPErrorExpr) Finalize(a *HTTPEndpointExpr)
func (e *HTTPErrorExpr) Dup() *HTTPErrorExpr
```

## Pre-existing callers

The eval engine calls `Validate` on every HTTP error mapping and `Finalize`
when preparing endpoint error responses. Endpoint/service/API error
inheritance calls `Dup`; `IsJSONRPC` routes reserved-code and header/cookie
restrictions for JSON-RPC designs.
