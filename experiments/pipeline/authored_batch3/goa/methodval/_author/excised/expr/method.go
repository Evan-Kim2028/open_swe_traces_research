// This file defines service methods and finalizes their payload, result,
// streaming, error, security, and interceptor contracts.
package expr

import (
	_ "errors"
	"fmt"
	_ "strings"

	"example.internal/apikit/v3/eval"
)

type (
	// StreamKind is a type denoting the kind of stream.
	StreamKind int

	// MethodExpr defines a single method.
	MethodExpr struct {
		// DSLFunc contains the DSL used to initialize the expression.
		eval.DSLFunc
		// Name of method.
		Name string
		// Description of method for consumption by humans.
		Description string
		// Docs points to the method external documentation if any.
		Docs *DocsExpr
		// Payload attribute
		Payload *AttributeExpr
		// Result attribute
		Result *AttributeExpr
		// Errors lists the error responses.
		Errors []*ErrorExpr
		// Requirements contains the security requirements for the
		// method. One requirement is composed of potentially multiple
		// schemes. Incoming requests must validate at least one
		// requirement to be authorized.
		Requirements []*SecurityExpr
		// ClientInterceptors is the list of client interceptors.
		ClientInterceptors []*InterceptorExpr
		// ServerInterceptors is the list of server interceptors.
		ServerInterceptors []*InterceptorExpr
		// Service that owns method.
		Service *ServiceExpr
		// Meta is an arbitrary set of key/value pairs, see dsl.Meta
		Meta MetaExpr
		// Idempotent reports whether replaying the exact invocation has the
		// same externally visible effect as invoking the method once.
		Idempotent bool
		// Stream is the kind of stream (none, payload, result, or both)
		// the method defines.
		Stream StreamKind
		// StreamingPayload is the payload sent across the stream.
		StreamingPayload *AttributeExpr
		// StreamingResult is the result sent across the stream when using SSE.
		// When Result and StreamingResult are both defined, the method supports
		// normal HTTP responses using Result and SSE streams using StreamingResult.
		StreamingResult *AttributeExpr
	}
)

const (
	// NoStreamKind represents no payload or result stream in method.
	NoStreamKind StreamKind = iota + 1
	// ClientStreamKind represents client sends a streaming payload to
	// method.
	ClientStreamKind
	// ServerStreamKind represents server sends a streaming result from
	// method.
	ServerStreamKind
	// BidirectionalStreamKind represents client and server sending payload
	// and result respectively via a stream.
	BidirectionalStreamKind
)

// Error returns the error with the given name declared by the method or its
// service, if any.
func (m *MethodExpr) Error(name string) *ErrorExpr {
	for _, err := range m.Errors {
		if err.Name == name {
			return err
		}
	}
	return m.Service.Error(name)
}

// EvalName returns the generic expression name used in error messages.
func (m *MethodExpr) EvalName() string {
	var prefix, suffix string
	if m.Name != "" {
		suffix = fmt.Sprintf("method %#v", m.Name)
	} else {
		suffix = "unnamed method"
	}
	if m.Service != nil {
		prefix = m.Service.EvalName() + " "
	}
	return prefix + suffix
}

// Prepare makes sure the payload and result types are initialized (to the Empty
// type if nil) and merges the method interceptors with the API and service level
// interceptors.
func (m *MethodExpr) Prepare() {
	panic("excised: MethodExpr.Prepare")
}

// Validate validates the method payloads, results, errors, security
// requirements, and interceptors.
func (m *MethodExpr) Validate() error {
	panic("excised: MethodExpr.Validate")
}

// validateRequirements validates the security requirements.
func (m *MethodExpr) validateRequirements() *eval.ValidationErrors {
	panic("excised: MethodExpr.validateRequirements")
}

// isSecurityAttribute reports whether metadata marks a method payload field as
// a username, password, token, or API key used by a security scheme.
func isSecurityAttribute(meta MetaExpr) bool {
	panic("excised: isSecurityAttribute")
}

// isStringType reports whether a design type is String or a named String type.
func isStringType(dt DataType) bool {
	panic("excised: isStringType")
}

// validateErrors validates the method errors.
func (m *MethodExpr) validateErrors() *eval.ValidationErrors {
	panic("excised: MethodExpr.validateErrors")
}

// validateInterceptors validates the method interceptors.
func (m *MethodExpr) validateInterceptors() *eval.ValidationErrors {
	panic("excised: MethodExpr.validateInterceptors")
}

// mergeInterceptors merges interceptors from different levels (method, service, API)
// while avoiding duplicates. The order of precedence is: method > service > API.
func mergeInterceptors(methodLevel, serviceLevel, apiLevel []*InterceptorExpr) []*InterceptorExpr {
	panic("excised: mergeInterceptors")
}

// hasTag is a helper function that traverses the given attribute and all its
// bases recursively looking for an attribute with the given tag meta. This
// recursion is only needed for attributes that have not been finalized yet.
func hasTag(p *AttributeExpr, tag string) bool {
	panic("excised: hasTag")
}

// hasTag is a helper function that traverses the given attribute and all its
// bases recursively looking for an attribute with the given tag meta prefix. This
// recursion is only needed for attributes that have not been finalized yet.
func hasTagPrefix(p *AttributeExpr, prefix string) bool {
	panic("excised: hasTagPrefix")
}

// Finalize makes sure the method payload and result types are set. It also
// projects the result if it is a result type and a view is explicitly set in
// the design or a result type having at most one view.
func (m *MethodExpr) Finalize() {
	panic("excised: MethodExpr.Finalize")
}

// IsStreaming determines whether the method streams payload or result.
func (m *MethodExpr) IsStreaming() bool {
	panic("excised: MethodExpr.IsStreaming")
}

// IsPayloadStreaming determines whether the method streams payload.
func (m *MethodExpr) IsPayloadStreaming() bool {
	panic("excised: MethodExpr.IsPayloadStreaming")
}

// IsResultStreaming determines whether the method streams payload.
func (m *MethodExpr) IsResultStreaming() bool {
	panic("excised: MethodExpr.IsResultStreaming")
}

// HasMixedResults returns true if the method defines Result and StreamingResult
// separately so HTTP clients can choose a normal response or an SSE stream.
func (m *MethodExpr) HasMixedResults() bool {
	panic("excised: MethodExpr.HasMixedResults")
}

// helper function that duplicates just enough of a security expression so that
// its scheme names can be overridden without affecting the original.
func copyReqs(reqs []*SecurityExpr) []*SecurityExpr {
	panic("excised: copyReqs")
}
