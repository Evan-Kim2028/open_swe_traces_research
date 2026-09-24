# Exported API — httpsvc

```go
func (svc *HTTPServiceExpr) Name() string
func (svc *HTTPServiceExpr) Description() string
func (svc *HTTPServiceExpr) Error(name string) *ErrorExpr
func (svc *HTTPServiceExpr) Endpoint(name string) *HTTPEndpointExpr
func (svc *HTTPServiceExpr) EndpointFor(m *MethodExpr) *HTTPEndpointExpr
func (svc *HTTPServiceExpr) CanonicalEndpoint() *HTTPEndpointExpr
func (svc *HTTPServiceExpr) FullPaths() []string
func (svc *HTTPServiceExpr) Parent() *HTTPServiceExpr
func (svc *HTTPServiceExpr) HTTPError(name string) *HTTPErrorExpr
func (svc *HTTPServiceExpr) EvalName() string
func (svc *HTTPServiceExpr) IsJSONRPC() bool
func (svc *HTTPServiceExpr) Prepare()
func (svc *HTTPServiceExpr) Validate() error
func (svc *HTTPServiceExpr) Finalize()
```

## Pre-existing callers

Route resolution, href generation, parent-service mounting, JSON-RPC route
setup, and the eval prepare/validate/finalize pipeline all call these.
