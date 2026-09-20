// This file defines the evaluated API expression and the immutable example
// randomizer configuration shared by independent code generation runs.
package expr

import (
	_ "sort"

	"example.internal/apikit/v3/eval"
)

type (
	// APIExpr contains the global properties for an API expression.
	APIExpr struct {
		// DSLFunc contains the DSL used to initialize the expression.
		eval.DSLFunc
		// Name of API
		Name string
		// Title of API
		Title string
		// Description of API
		Description string
		// Version is the version of the API described by this DSL.
		Version string
		// Servers lists the API hosts.
		Servers []*ServerExpr
		// TermsOfService describes or links to the service terms of API.
		TermsOfService string
		// Contact provides the API users with contact information.
		Contact *ContactExpr
		// License describes the API license.
		License *LicenseExpr
		// Docs points to the API external documentation.
		Docs *DocsExpr
		// Meta is a list of key/value pairs.
		Meta MetaExpr
		// Requirements contains the security requirements that apply to
		// all the API service methods. One requirement is composed of
		// potentially multiple schemes. Incoming requests must validate
		// at least one requirement to be authorized.
		Requirements []*SecurityExpr
		// ClientInterceptors is the list of API client interceptors.
		ClientInterceptors []*InterceptorExpr
		// ServerInterceptors is the list of API server interceptors.
		ServerInterceptors []*InterceptorExpr
		// HTTP contains the HTTP specific API level expressions.
		HTTP *HTTPExpr
		// GRPC contains the gRPC specific API level expressions.
		GRPC *GRPCExpr
		// JSONRPC contains the JSON-RPC specific API level expressions.
		JSONRPC *JSONRPCExpr

		// RandomizerFactory is the immutable configuration used to create a
		// fresh example value stream for each code generation run.
		RandomizerFactory RandomizerFactory
	}

	// ContactExpr contains the API contact information.
	ContactExpr struct {
		// Name of the contact person/organization
		Name string `json:"name,omitempty"`
		// Email address of the contact person/organization
		Email string `json:"email,omitempty"`
		// URL pointing to the contact information
		URL string `json:"url,omitempty"`
	}

	// LicenseExpr contains the license information for the API.
	LicenseExpr struct {
		// Name of license used for the API
		Name string `json:"name,omitempty"`
		// URL to the license used for the API
		URL string `json:"url,omitempty"`
	}

	// DocsExpr points to external documentation.
	DocsExpr struct {
		// Description of documentation.
		Description string `json:"description,omitempty"`
		// URL to documentation.
		URL string `json:"url,omitempty"`
	}

	// URLHolder is an interface that allows expression types to receive
	// a URL. Types implementing this interface can use the URL() DSL
	// function to set a URL.
	URLHolder interface {
		SetURL(string)
	}

	// DescriptionHolder is an interface that allows expression types to
	// receive a description. Types implementing this interface can use
	// the Description() DSL function to set a description.
	DescriptionHolder interface {
		SetDescription(string)
	}

	// TitleHolder is an interface that allows expression types to receive
	// a title. Types implementing this interface can use the Title() DSL
	// function to set a title.
	TitleHolder interface {
		SetTitle(string)
	}

	// VersionHolder is an interface that allows expression types to
	// receive a version. Types implementing this interface can use the
	// Version() DSL function to set a version.
	VersionHolder interface {
		SetVersion(string)
	}

	// TimeoutHolder is an interface that allows expression types to
	// receive a timeout duration. Types implementing this interface can
	// use the Timeout() DSL function to set a timeout.
	TimeoutHolder interface {
		SetTimeout(string) error
	}
)

// NewAPIExpr initializes an API expression.
func NewAPIExpr(name string, dsl func()) *APIExpr {
	panic("excised: NewAPIExpr")
}

// Schemes returns the list of transport schemes used by all the API servers.
// The possible values for the elements of the returned slice are "http",
// "https", "grpc" and "grpcs".
func (a *APIExpr) Schemes() []string {
	panic("excised: APIExpr.Schemes")
}

// DefaultServer returns a server expression that describes a server which
// exposes all the services in the design and listens on localhost port 80 for
// HTTP requests and port 8080 for gRPC requests.
func (a *APIExpr) DefaultServer() *ServerExpr {
	panic("excised: APIExpr.DefaultServer")
}

// EvalName is the qualified name of the expression.
func (a *APIExpr) EvalName() string {
	panic("excised: APIExpr.EvalName")
}

// Hash returns a unique hash value for a.
func (a *APIExpr) Hash() string {
	panic("excised: APIExpr.Hash")
}

// Finalize makes sure that the API name is initialized and there is at least
// one server definition (if none exists, it creates a default server). If API
// name is empty, it sets the name of the first service definition as API name.
func (a *APIExpr) Finalize() {
	panic("excised: APIExpr.Finalize")
}

// EvalName is the qualified name of the expression.
func (l *LicenseExpr) EvalName() string {
	panic("excised: LicenseExpr.EvalName")
}

// EvalName is the qualified name of the expression.
func (d *DocsExpr) EvalName() string {
	panic("excised: DocsExpr.EvalName")
}

// EvalName is the qualified name of the expression.
func (c *ContactExpr) EvalName() string {
	panic("excised: ContactExpr.EvalName")
}
