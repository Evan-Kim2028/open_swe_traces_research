# Exported API — exid

```go
type ExampleIdentity struct{ /* private seed */ }

func UserTypeExampleIdentity(typ UserType) ExampleIdentity
func GeneratedUserTypeExampleIdentity(typ UserType) (ExampleIdentity, bool)
func MethodPayloadExampleIdentity(m *MethodExpr) ExampleIdentity
func MethodResultExampleIdentity(m *MethodExpr) ExampleIdentity
func MethodStreamingPayloadExampleIdentity(m *MethodExpr) ExampleIdentity
func MethodStreamingResultExampleIdentity(m *MethodExpr) ExampleIdentity
func MethodErrorExampleIdentity(m *MethodExpr, err *ErrorExpr) ExampleIdentity
func RequestBodyExampleIdentity(e *HTTPEndpointExpr) ExampleIdentity
func ResponseBodyExampleIdentity(e *HTTPEndpointExpr, r *HTTPResponseExpr) ExampleIdentity
func ErrorResponseBodyExampleIdentity(e *HTTPEndpointExpr, r *HTTPErrorExpr) ExampleIdentity
func GRPCRequestMessageExampleIdentity(m *MethodExpr) ExampleIdentity
// ... plus streaming, error, wrapper variants and Member/ArrayElement/
// MapKey/MapValue/UnionMember descent and Seed().
```

## Pre-existing callers

The example generator anchors every random draw on an ExampleIdentity;
generated service code passes `Seed()` to custom randomizer factories.
