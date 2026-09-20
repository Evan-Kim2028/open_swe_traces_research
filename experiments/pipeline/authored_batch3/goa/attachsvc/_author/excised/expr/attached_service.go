// This file checks and finishes services that generators add after the design
// DSL has run.
package expr

import (
	_ "fmt"
	_ "slices"

	"example.internal/apikit/v3/eval"
)

// EvaluateAttachedServices prepares, checks, and finishes services that a
// generator added to r. The services and types must already belong to r. No
// service is finished unless every added expression is valid.
func (r *RootExpr) EvaluateAttachedServices(services []*ServiceExpr, types ...UserType) error {
	panic("excised: RootExpr.EvaluateAttachedServices")
}

// attachedServiceExpressions verifies that each service and type belongs to r
// and that every endpoint points to a method on its service. It returns the
// expressions in the order that Prepare, Validate, and Finalize must run.
func (r *RootExpr) attachedServiceExpressions(
	services []*ServiceExpr,
	types []UserType,
) ([]eval.ExpressionSet, error) {
	panic("excised: RootExpr.attachedServiceExpressions")
}

// collectHTTPExpressions returns the selected HTTP services, endpoints, and
// file servers. It rejects a child that points to another service.
func collectHTTPExpressions(
	transports []*HTTPServiceExpr,
	httpRoot *HTTPExpr,
	selected map[*ServiceExpr]struct{},
) (eval.ExpressionSet, eval.ExpressionSet, eval.ExpressionSet, error) {
	panic("excised: collectHTTPExpressions")
}

// collectGRPCExpressions returns the selected gRPC services and endpoints and
// rejects any endpoint that points outside its service.
func collectGRPCExpressions(
	transports []*GRPCServiceExpr,
	selected map[*ServiceExpr]struct{},
) (eval.ExpressionSet, eval.ExpressionSet, error) {
	panic("excised: collectGRPCExpressions")
}

// prepareExpressions calls Prepare on each added expression in Apikit's required
// order.
func prepareExpressions(sets []eval.ExpressionSet) {
	panic("excised: prepareExpressions")
}

// validateExpressions checks the complete design and each added expression. It
// returns all errors together.
func validateExpressions(root *RootExpr, sets []eval.ExpressionSet) error {
	panic("excised: validateExpressions")
}

// finalizeExpressions calls Finalize on each added expression after every
// check succeeds.
func finalizeExpressions(sets []eval.ExpressionSet) {
	panic("excised: finalizeExpressions")
}
