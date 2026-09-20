# Exported API — httpresp

```go
type HTTPResponseExpr struct {
    StatusCode    int
    StatusCodeSet bool
    Description   string
    Headers       *MappedAttributeExpr
    Cookies       *MappedAttributeExpr
    Body          *AttributeExpr
    ContentType   string
    Tag           [2]string
    Parent        eval.Expression
    Meta          MetaExpr
}

func (r *HTTPResponseExpr) EvalName() string
func (r *HTTPResponseExpr) Prepare()
func (r *HTTPResponseExpr) Validate(e *HTTPEndpointExpr) *eval.ValidationErrors
func (r *HTTPResponseExpr) Finalize(a *HTTPEndpointExpr, svcAtt *AttributeExpr)
func (r *HTTPResponseExpr) Dup() *HTTPResponseExpr
```

## Pre-existing callers

The eval engine runs `Prepare`/`Validate`/`Finalize` on every response
declared in a design. `HTTPEndpointExpr` clones responses via `Dup` when
inheriting them, and body computation reads the finalized `Body`,
`Headers`, `Cookies` and `ContentType`.
