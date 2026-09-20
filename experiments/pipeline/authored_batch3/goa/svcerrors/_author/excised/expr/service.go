// This file defines services and their errors. It also distinguishes errors
// named in the design from errors created for one method.
package expr

import (
	_ "errors"
	_ "fmt"
	_ "strings"

	"example.internal/apikit/v3/eval"
)

type (
	// ServiceExpr describes a set of related methods.
	ServiceExpr struct {
		// DSLFunc contains the DSL used to initialize the expression.
		eval.DSLFunc
		// Name of service.
		Name string
		// Description of service used in documentation.
		Description string
		// Docs points to external documentation
		Docs *DocsExpr
		// Methods is the list of service methods.
		Methods []*MethodExpr
		// Errors list the errors common to all the service methods.
		Errors []*ErrorExpr
		// Requirements contains the security requirements that apply to
		// all the service methods. One requirement is composed of
		// potentially multiple schemes. Incoming requests must validate
		// at least one requirement to be authorized.
		Requirements []*SecurityExpr
		// ClientInterceptors is the list of client interceptors.
		ClientInterceptors []*InterceptorExpr
		// ServerInterceptors is the list of server interceptors.
		ServerInterceptors []*InterceptorExpr
		// Meta is a set of key/value pairs with semantic that is
		// specific to each generator.
		Meta MetaExpr
		// design points to the root containing this service.
		design *RootExpr
	}

	// ErrorExpr defines an error response. It consists of a named
	// attribute.
	ErrorExpr struct {
		// AttributeExpr is the underlying attribute.
		*AttributeExpr
		// Name is the unique name of the error.
		Name string
	}
)

// Method returns the method expression with the given name, nil if there isn't
// one.
func (s *ServiceExpr) Method(n string) *MethodExpr {
	panic("excised: ServiceExpr.Method")
}

// EvalName returns the generic expression name used in error messages.
func (s *ServiceExpr) EvalName() string {
	panic("excised: ServiceExpr.EvalName")
}

// Error returns the error with the given name declared by the service, if any.
func (s *ServiceExpr) Error(name string) *ErrorExpr {
	panic("excised: ServiceExpr.Error")
}

// Hash returns a unique hash value for s.
func (s *ServiceExpr) Hash() string {
	panic("excised: ServiceExpr.Hash")
}

// Validate validates the service methods and errors.
func (s *ServiceExpr) Validate() error {
	panic("excised: ServiceExpr.Validate")
}

// validateInlineMethodErrors rejects two inline errors that request one public
// Go error name but define different values.
func (s *ServiceExpr) validateInlineMethodErrors() *eval.ValidationErrors {
	panic("excised: ServiceExpr.validateInlineMethodErrors")
}

// standardErrorUsesGeneratedConstructor reports whether Apikit generates the
// shared Make<Name> function whose behavior repeated declarations could change.
func standardErrorUsesGeneratedConstructor(errorExpression *ErrorExpr) bool {
	panic("excised: standardErrorUsesGeneratedConstructor")
}

// Finalize finalizes all the service methods and errors.
func (s *ServiceExpr) Finalize() {
	panic("excised: ServiceExpr.Finalize")
}

// Validate checks that the error name is found in the result meta for
// custom error types.
func (e *ErrorExpr) Validate() error {
	panic("excised: ErrorExpr.Validate")
}

// Finalize makes sure the error type is a user type since it has to generate a
// Go error.
// Note: this may produce a user type with an attribute that is not an object!
func (e *ErrorExpr) Finalize() {
	panic("excised: ErrorExpr.Finalize")
}

// finalizeMethodType wraps an inline method error and assigns the repeatable
// example key used by service and transport generators for that method error.
func (e *ErrorExpr) finalizeMethodType(method *MethodExpr) {
	panic("excised: ErrorExpr.finalizeMethodType")
}

// previousInlineMethodErrorOrigin returns the declaration already created for
// the same inline error by an earlier method in this service.
func previousInlineMethodErrorOrigin(method *MethodExpr, name string) UserType {
	panic("excised: previousInlineMethodErrorOrigin")
}
