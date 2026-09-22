package expr

import (
	"example.internal/apikit/v3/eval"
)

type (
	// InterceptorExpr describes an interceptor definition in the design.
	// Interceptors are used to inject user code into the request/response processing pipeline.
	// There are four kinds of interceptors, in order of execution:
	//   * client-side payload: executes after the payload is encoded and before the request is sent to the server
	//   * server-side request: executes after the request is decoded and before the payload is sent to the service
	//   * server-side result: executes after the service returns a result and before the response is encoded
	//   * client-side response: executes after the response is decoded and before the result is sent to the client
	InterceptorExpr struct {
		// Name is the name of the interceptor
		Name string
		// Description is the optional description of the interceptor
		Description string
		// ReadPayload lists the payload attribute names read by the interceptor
		ReadPayload *AttributeExpr
		// WritePayload lists the payload attribute names written by the interceptor
		WritePayload *AttributeExpr
		// ReadResult lists the result attribute names read by the interceptor
		ReadResult *AttributeExpr
		// WriteResult lists the result attribute names written by the interceptor
		WriteResult *AttributeExpr
		// ReadStreamingPayload lists the streaming payload attribute names read by the interceptor
		ReadStreamingPayload *AttributeExpr
		// WriteStreamingPayload lists the streaming payload attribute names written by the interceptor
		WriteStreamingPayload *AttributeExpr
		// ReadStreamingResult lists the streaming result attribute names read by the interceptor
		ReadStreamingResult *AttributeExpr
		// WriteStreamingResult lists the streaming result attribute names written by the interceptor
		WriteStreamingResult *AttributeExpr
	}
)

// EvalName returns the generic expression name used in error messages.
func (i *InterceptorExpr) EvalName() string {
	panic("excised: InterceptorExpr.EvalName")
}

// validate validates the interceptor.
func (i *InterceptorExpr) validate(m *MethodExpr) *eval.ValidationErrors {
	panic("excised: InterceptorExpr.validate")
}

// validateAttributeAccess validates that all attributes in attr exist in obj
func (i *InterceptorExpr) validateAttributeAccess(m *MethodExpr, source string, verr *eval.ValidationErrors, target *AttributeExpr, attr *AttributeExpr) {
	panic("excised: InterceptorExpr.validateAttributeAccess")
}
