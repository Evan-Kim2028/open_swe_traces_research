// This file binds reusable HTTP error-response policy to the concrete error
// returned by each endpoint method.
package expr

import (
	"example.internal/apikit/v3/eval"
)

type (
	// HTTPErrorExpr defines a HTTP error response including its name,
	// status, headers and result type.
	HTTPErrorExpr struct {
		// ErrorExpr is the underlying apikit design error expression.
		*ErrorExpr
		// Name of error, we need a separate copy of the name to match it
		// up with the appropriate ErrorExpr.
		Name string
		// Response is the corresponding HTTP response.
		Response *HTTPResponseExpr
	}
)

// EvalName returns the generic definition name used in error messages.
func (e *HTTPErrorExpr) EvalName() string {
	panic("excised: HTTPErrorExpr.EvalName")
}

// IsJSONRPC reports whether the response maps an error for a JSON-RPC API,
// service, or method.
func (e *HTTPErrorExpr) IsJSONRPC() bool {
	panic("excised: HTTPErrorExpr.IsJSONRPC")
}

// Validate makes sure there is a error expression that matches the HTTP error
// expression.
func (e *HTTPErrorExpr) Validate() *eval.ValidationErrors {
	panic("excised: HTTPErrorExpr.Validate")
}

// jsonRPCErrorCodeReserved reports whether code belongs to the part of the
// JSON-RPC reserved range that an application cannot use.
func jsonRPCErrorCodeReserved(code int) bool {
	panic("excised: jsonRPCErrorCodeReserved")
}

// Finalize looks up the corresponding method error expression.
func (e *HTTPErrorExpr) Finalize(a *HTTPEndpointExpr) {
	panic("excised: HTTPErrorExpr.Finalize")
}

// mappedError returns the error declaration that owns this reusable HTTP
// response policy before the policy is applied to an endpoint method.
func (e *HTTPErrorExpr) mappedError() (*ErrorExpr, string) {
	panic("excised: HTTPErrorExpr.mappedError")
}

// Dup creates a copy of the error expression.
func (e *HTTPErrorExpr) Dup() *HTTPErrorExpr {
	panic("excised: HTTPErrorExpr.Dup")
}
