package expr

import (
	_ "fmt"

	"example.internal/apikit/v3/eval"
)

type (
	// GRPCResponseExpr defines a gRPC response including its status code, result
	// type, and metadata.
	GRPCResponseExpr struct {
		// gRPC status code
		StatusCode int
		// Response description
		Description string
		// Response Message if any
		Message *AttributeExpr
		// Parent expression, one of EndpointExpr, ServiceExpr or
		// RootExpr.
		Parent eval.Expression
		// Headers is the header metadata to be sent in the gRPC response.
		Headers *MappedAttributeExpr
		// Trailers is the trailer metadata to be sent in the gRPC response.
		Trailers *MappedAttributeExpr
		// Meta is a list of key/value pairs.
		Meta MetaExpr
	}
)

// EvalName returns the generic definition name used in error messages.
func (r *GRPCResponseExpr) EvalName() string {
	panic("excised: GRPCResponseExpr.EvalName")
}

// Prepare makes sure the response message and metadata are initialized.
func (r *GRPCResponseExpr) Prepare() {
	panic("excised: GRPCResponseExpr.Prepare")
}

// Validate checks that the response definition is consistent: its status is set
// and the result type definition if any is valid.
func (r *GRPCResponseExpr) Validate(e *GRPCEndpointExpr) *eval.ValidationErrors {
	panic("excised: GRPCResponseExpr.Validate")
}

// Finalize ensures that the response message type is set. If Message DSL is
// used to set the response message then the message type is set by mapping
// the attributes to the method Result expression. If no response message set
// explicitly, the message is set from the method Result expression.
func (r *GRPCResponseExpr) Finalize(a *GRPCEndpointExpr, svcAtt *AttributeExpr) {
	panic("excised: GRPCResponseExpr.Finalize")
}

// Dup creates a copy of the response expression.
func (r *GRPCResponseExpr) Dup() *GRPCResponseExpr {
	panic("excised: GRPCResponseExpr.Dup")
}
