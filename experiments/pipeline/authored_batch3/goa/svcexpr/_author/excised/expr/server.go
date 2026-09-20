package expr

import (
	_ "errors"
	_ "fmt"
	_ "net/url"
	"regexp"
	_ "slices"
	_ "sort"
	_ "strings"

	_ "example.internal/apikit/v3/eval"
)

// uriVariableRegex is the regular expression used to capture variables in URI
// expression.
var uriVariableRegex = regexp.MustCompile(`{\*?([a-zA-Z0-9_]+)}`)

type (
	// ServerExpr contains a single API host information.
	ServerExpr struct {
		// Name of server
		Name string
		// Description of server
		Description string
		// Services list the services hosted by the server.
		Services []string
		// Hosts list the server hosts.
		Hosts []*HostExpr
		// Meta is a set of key/value pairs.
		Meta MetaExpr
	}

	// HostExpr describes a server host.
	HostExpr struct {
		// Name of host
		Name string
		// Name of server that uses host.
		ServerName string
		// Description of host
		Description string
		// URIs to host if any, may contain parameter elements using
		// the "{param}" syntax.
		URIs []URIExpr
		// Variables defines the URI variables if any.
		Variables *AttributeExpr
		// Meta is a set of key/value pairs.
		Meta MetaExpr
	}

	// URIExpr represents a parameterized URI.
	URIExpr string
)

// EvalName is the qualified name of the expression.
func (s *ServerExpr) EvalName() string {
	panic("excised: ServerExpr.EvalName")
}

// Validate validates the server and server hosts.
func (s *ServerExpr) Validate() error {
	panic("excised: ServerExpr.Validate")
}

// Finalize initializes the server services and/or host with default values if
// not set explicitly in the design.
func (s *ServerExpr) Finalize() {
	panic("excised: ServerExpr.Finalize")
}

// Schemes returns the list of transport schemes used by all the server
// endpoints. The possible values for the elements of the returned slice are
// "http", "https", "grpc" and "grpcs".
func (s *ServerExpr) Schemes() []string {
	panic("excised: ServerExpr.Schemes")
}

var validSchemes = map[string]struct{}{"http": {}, "https": {}, "grpc": {}, "grpcs": {}}

// Validate validates the host.
func (h *HostExpr) Validate() error {
	panic("excised: HostExpr.Validate")
}

// Finalize makes sure Variables is set.
func (h *HostExpr) Finalize() {
	panic("excised: HostExpr.Finalize")
}

// EvalName returns the name returned in error messages.
func (h *HostExpr) EvalName() string {
	panic("excised: HostExpr.EvalName")
}

// Attribute returns the variables attribute.
func (h *HostExpr) Attribute() *AttributeExpr {
	panic("excised: HostExpr.Attribute")
}

// Schemes returns the list of transport schemes defined for the host. The
// possible values for the elements of the returned slice are "http", "https",
// "grpc" and "grpcs".
func (h *HostExpr) Schemes() []string {
	panic("excised: HostExpr.Schemes")
}

// HasHTTPScheme returns true if at least one of the URIs in the host
// expression define "http" or "https" scheme.
func (h *HostExpr) HasHTTPScheme() bool {
	panic("excised: HostExpr.HasHTTPScheme")
}

// HasGRPCScheme returns true if at least one of the URIs in the host
// expression define "grpc" or "grpcs" scheme.
func (h *HostExpr) HasGRPCScheme() bool {
	panic("excised: HostExpr.HasGRPCScheme")
}

// URIString returns a valid URI string by substituting the parameters with
// their default value if present or the first item in their enum. It returns
// an error if the given URI expression is not found in the host URIs.
func (h *HostExpr) URIString(u URIExpr) (string, error) {
	panic("excised: HostExpr.URIString")
}

// Params return the names of the parameters used in URI if any.
func (u URIExpr) Params() []string {
	panic("excised: URIExpr.Params")
}

// Scheme returns the URI scheme. Possible values are http, https, grpc, and
// grpcs.
func (u URIExpr) Scheme() string {
	panic("excised: URIExpr.Scheme")
}
