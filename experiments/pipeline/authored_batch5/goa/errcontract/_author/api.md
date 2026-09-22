# Exported API — errcontract

The closure is package-private; callers reach it through the evaluated
service/method error contract:

```go
// equivalentErrorAttributes reports whether two error attributes generate
// the same service value contract.
func equivalentErrorAttributes(first, second *AttributeExpr) bool

// differingErrorQualifierSettings lists error settings that would change
// the generated service error returned to callers.
func differingErrorQualifierSettings(first, second *AttributeExpr) []string

// effectiveErrorAttribute returns a detached copy with References and Bases
// applied by AttributeExpr.Finalize.
func effectiveErrorAttribute(source *AttributeExpr) *AttributeExpr
```

## Pre-existing callers

Service and transport error mapping (`expr/service.go`,
`expr/http_error.go`, `expr/grpc_error.go`, `expr/jsonrpc.go`) validate that
an inherited or redeclared error is contract-compatible: method errors may
replace service or API errors only when they generate the same service
value, and qualifier differences are reported by name.
