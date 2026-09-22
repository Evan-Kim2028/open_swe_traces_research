// This file builds repeatable keys for example values from evaluated service,
// method, type, field, and transport names.
package expr

import (
	_ "encoding/base64"
	_ "encoding/binary"
)

type (
	// ExampleIdentity selects a repeatable sequence of generated example values.
	// Equal values select the same sequence. Its fields are private so callers
	// must use the constructors below.
	ExampleIdentity struct {
		seed string
	}

	exampleIdentityKind byte
)

const (
	userTypeExampleKind exampleIdentityKind = iota + 1
	methodPayloadExampleKind
	methodResultExampleKind
	methodStreamingPayloadExampleKind
	methodStreamingResultExampleKind
	methodErrorExampleKind
	httpRequestBodyExampleKind
	httpResponseBodyExampleKind
	httpErrorResponseBodyExampleKind
	jsonRPCRequestBodyExampleKind
	jsonRPCResponseBodyExampleKind
	jsonRPCErrorResponseBodyExampleKind
	grpcRequestMessageExampleKind
	grpcResponseMessageExampleKind
	grpcStreamingRequestMessageExampleKind
	grpcStreamingResponseMessageExampleKind
	grpcErrorMessageExampleKind
	grpcArrayWrapperExampleKind
	grpcMapWrapperExampleKind
	memberExampleKind
	arrayElementExampleKind
	mapKeyExampleKind
	mapValueExampleKind
	unionMemberExampleKind
)

// UserTypeExampleIdentity returns the example key for typ.
func UserTypeExampleIdentity(typ UserType) ExampleIdentity {
	panic("excised: UserTypeExampleIdentity")
}

// GeneratedUserTypeExampleIdentity returns the example key stored on a user
// type created by Apikit. The second result is false for types written in the
// design.
func GeneratedUserTypeExampleIdentity(typ UserType) (ExampleIdentity, bool) {
	panic("excised: GeneratedUserTypeExampleIdentity")
}

// MethodPayloadExampleIdentity returns the example key for method's payload.
func MethodPayloadExampleIdentity(method *MethodExpr) ExampleIdentity {
	panic("excised: MethodPayloadExampleIdentity")
}

// MethodResultExampleIdentity returns the example key for method's result.
func MethodResultExampleIdentity(method *MethodExpr) ExampleIdentity {
	panic("excised: MethodResultExampleIdentity")
}

// MethodStreamingPayloadExampleIdentity returns the example key for method's
// streaming payload.
func MethodStreamingPayloadExampleIdentity(method *MethodExpr) ExampleIdentity {
	panic("excised: MethodStreamingPayloadExampleIdentity")
}

// MethodStreamingResultExampleIdentity returns the example key for method's
// streaming result.
func MethodStreamingResultExampleIdentity(method *MethodExpr) ExampleIdentity {
	panic("excised: MethodStreamingResultExampleIdentity")
}

// MethodErrorExampleIdentity returns the example key for err in method.
func MethodErrorExampleIdentity(method *MethodExpr, err *ErrorExpr) ExampleIdentity {
	panic("excised: MethodErrorExampleIdentity")
}

// RequestBodyExampleIdentity returns the example key for endpoint's request
// body. HTTP and JSON-RPC endpoints receive different keys.
func RequestBodyExampleIdentity(endpoint *HTTPEndpointExpr) ExampleIdentity {
	panic("excised: RequestBodyExampleIdentity")
}

// ResponseBodyExampleIdentity returns the example key for a successful response
// body. HTTP and JSON-RPC endpoints receive different keys, and each successful
// status code receives its own key.
func ResponseBodyExampleIdentity(endpoint *HTTPEndpointExpr, response *HTTPResponseExpr) ExampleIdentity {
	panic("excised: ResponseBodyExampleIdentity")
}

// ErrorResponseBodyExampleIdentity returns the example key for an error response
// body. HTTP and JSON-RPC endpoints receive different keys. The error name keeps
// two errors with the same HTTP status separate.
func ErrorResponseBodyExampleIdentity(endpoint *HTTPEndpointExpr, response *HTTPErrorExpr) ExampleIdentity {
	panic("excised: ErrorResponseBodyExampleIdentity")
}

// GRPCRequestMessageExampleIdentity returns the example key for method's gRPC
// request message.
func GRPCRequestMessageExampleIdentity(method *MethodExpr) ExampleIdentity {
	panic("excised: GRPCRequestMessageExampleIdentity")
}

// GRPCResponseMessageExampleIdentity returns the example key for method's gRPC
// response message.
func GRPCResponseMessageExampleIdentity(method *MethodExpr) ExampleIdentity {
	panic("excised: GRPCResponseMessageExampleIdentity")
}

// GRPCStreamingRequestMessageExampleIdentity returns the example key for
// method's streaming gRPC request message.
func GRPCStreamingRequestMessageExampleIdentity(method *MethodExpr) ExampleIdentity {
	panic("excised: GRPCStreamingRequestMessageExampleIdentity")
}

// GRPCStreamingResponseMessageExampleIdentity returns the example key for
// method's streaming gRPC response message.
func GRPCStreamingResponseMessageExampleIdentity(method *MethodExpr) ExampleIdentity {
	panic("excised: GRPCStreamingResponseMessageExampleIdentity")
}

// GRPCErrorMessageExampleIdentity returns the example key for err's gRPC
// message in method.
func GRPCErrorMessageExampleIdentity(method *MethodExpr, err *ErrorExpr) ExampleIdentity {
	panic("excised: GRPCErrorMessageExampleIdentity")
}

// GRPCArrayWrapperExampleIdentity returns the example key for the gRPC message
// that wraps an array type written in the design.
func GRPCArrayWrapperExampleIdentity(typ UserType) ExampleIdentity {
	panic("excised: GRPCArrayWrapperExampleIdentity")
}

// GRPCMapWrapperExampleIdentity returns the example key for the gRPC message
// that wraps a map type written in the design.
func GRPCMapWrapperExampleIdentity(typ UserType) ExampleIdentity {
	panic("excised: GRPCMapWrapperExampleIdentity")
}

// Seed returns the complete encoded key passed to custom randomizer factories.
func (i ExampleIdentity) Seed() string {
	panic("excised: ExampleIdentity.Seed")
}

// Member returns the example key for name within i.
func (i ExampleIdentity) Member(name string) ExampleIdentity {
	panic("excised: ExampleIdentity.Member")
}

// ArrayElement returns the example key for index within the array at i.
func (i ExampleIdentity) ArrayElement(index int) ExampleIdentity {
	panic("excised: ExampleIdentity.ArrayElement")
}

// MapKey returns the example key for key index within the map at i.
func (i ExampleIdentity) MapKey(index int) ExampleIdentity {
	panic("excised: ExampleIdentity.MapKey")
}

// MapValue returns the example key for value index within the map at i.
func (i ExampleIdentity) MapValue(index int) ExampleIdentity {
	panic("excised: ExampleIdentity.MapValue")
}

// UnionMember returns the example key for name within the union at i.
func (i ExampleIdentity) UnionMember(name string) ExampleIdentity {
	panic("excised: ExampleIdentity.UnionMember")
}

// A new example key encodes a value kind and each component's byte length so
// different component lists cannot produce the same key.
func newExampleIdentity(kind exampleIdentityKind, components ...[]byte) ExampleIdentity {
	panic("excised: newExampleIdentity")
}

// Method example keys are built from the evaluated service and method names.
func methodExampleIdentity(kind exampleIdentityKind, method *MethodExpr) ExampleIdentity {
	panic("excised: methodExampleIdentity")
}

// Integers in example keys are written as eight bytes in big-endian order.
func exampleIdentityInt(value int) []byte {
	panic("excised: exampleIdentityInt")
}

// Each added part writes its kind, number of components, and every component's
// byte length before the component bytes.
func appendExampleIdentitySegment(seed []byte, kind exampleIdentityKind, components ...[]byte) []byte {
	panic("excised: appendExampleIdentitySegment")
}

// append adds one member, array, map, or union part to the current key.
func (i ExampleIdentity) append(kind exampleIdentityKind, components ...[]byte) ExampleIdentity {
	panic("excised: ExampleIdentity.append")
}
