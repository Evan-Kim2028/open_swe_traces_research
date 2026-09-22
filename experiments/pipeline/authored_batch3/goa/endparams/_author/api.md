# Exported API — endparams

```go
func (e *HTTPEndpointExpr) PathParams() *MappedAttributeExpr
func (e *HTTPEndpointExpr) QueryParams() *MappedAttributeExpr
func (r *RouteExpr) Validate() *eval.ValidationErrors
func (r *RouteExpr) Params() []string
func (r *RouteExpr) FullPaths() []string
func (r *RouteExpr) IsAbsolute() bool
```

`validateParams` and `validateHeadersAndCookies` are unexported endpoint
validators.

## Pre-existing callers

Endpoint Prepare/Validate compute path and query params for every
route; transport codegen reads PathParams/QueryParams to build request
decoders.
