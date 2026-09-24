package expr

import (
	_ "fmt"
	_ "path"
	_ "strings"

	_ "github.com/dimfeld/httppath"
	"example.internal/apikit/v3/eval"
)

type (
	// HTTPServiceExpr describes a HTTP service. It defines both a result
	// type and a set of endpoints that can be executed through HTTP
	// requests. HTTPServiceExpr embeds a ServiceExpr and adds HTTP specific
	// properties.
	HTTPServiceExpr struct {
		eval.DSLFunc
		// Root is the root HTTP expression.
		Root *HTTPExpr
		// ServiceExpr is the service expression that backs this
		// service.
		ServiceExpr *ServiceExpr
		// Common URL prefixes to all service endpoint HTTP requests
		Paths []string
		// Params defines the HTTP request path and query parameters
		// common to all the service endpoints.
		Params *MappedAttributeExpr
		// Headers defines the HTTP request headers common to all the
		// service endpoints.
		Headers *MappedAttributeExpr
		// Cookies defines the HTTP request cookies common to all the
		// service endpoints.
		Cookies *MappedAttributeExpr
		// Name of parent service if any
		ParentName string
		// Endpoint with canonical service path
		CanonicalEndpointName string
		// HTTPEndpoints is the list of service endpoints.
		HTTPEndpoints []*HTTPEndpointExpr
		// HTTPErrors lists HTTP errors that apply to all endpoints.
		HTTPErrors []*HTTPErrorExpr
		// FileServers is the list of static asset serving endpoints
		FileServers []*HTTPFileServerExpr
		// SSE defines the Server-Sent Events configuration for all streaming endpoints
		// in this service. If nil, streaming endpoints use WebSockets by default.
		SSE *HTTPSSEExpr
		// JSONRPCRoute is the route used for all JSON-RPC endpoints in this service.
		// Only applicable to JSON-RPC services.
		JSONRPCRoute *RouteExpr
		// Meta is a set of key/value pairs with semantic that is
		// specific to each generator.
		Meta MetaExpr
	}
)

// Name of service (service)
func (svc *HTTPServiceExpr) Name() string {
	panic("excised: HTTPServiceExpr.Name")
}

// Description of service (service)
func (svc *HTTPServiceExpr) Description() string {
	panic("excised: HTTPServiceExpr.Description")
}

// Error returns the error contract available to service-level HTTP mappings.
// Service declarations take precedence over reusable API declarations.
func (svc *HTTPServiceExpr) Error(name string) *ErrorExpr {
	panic("excised: HTTPServiceExpr.Error")
}

// Endpoint returns the service endpoint with the given name or nil if there
// isn't one.
func (svc *HTTPServiceExpr) Endpoint(name string) *HTTPEndpointExpr {
	panic("excised: HTTPServiceExpr.Endpoint")
}

// EndpointFor builds the endpoint for the given method.
func (svc *HTTPServiceExpr) EndpointFor(m *MethodExpr) *HTTPEndpointExpr {
	panic("excised: HTTPServiceExpr.EndpointFor")
}

// CanonicalEndpoint returns the canonical endpoint of the service if any.
// The canonical endpoint is used to compute hrefs to services.
func (svc *HTTPServiceExpr) CanonicalEndpoint() *HTTPEndpointExpr {
	panic("excised: HTTPServiceExpr.CanonicalEndpoint")
}

// FullPaths computes the base paths to the service endpoints concatenating the
// API and parent service base paths as needed.
func (svc *HTTPServiceExpr) FullPaths() []string {
	panic("excised: HTTPServiceExpr.FullPaths")
}

// Parent returns the parent service if any, nil otherwise.
func (svc *HTTPServiceExpr) Parent() *HTTPServiceExpr {
	panic("excised: HTTPServiceExpr.Parent")
}

// HTTPError returns the service HTTP error with given name if any.
func (svc *HTTPServiceExpr) HTTPError(name string) *HTTPErrorExpr {
	panic("excised: HTTPServiceExpr.HTTPError")
}

// EvalName returns the generic definition name used in error messages.
func (svc *HTTPServiceExpr) EvalName() string {
	panic("excised: HTTPServiceExpr.EvalName")
}

// IsJSONRPC reports whether svc describes the JSON-RPC transport for its service.
func (svc *HTTPServiceExpr) IsJSONRPC() bool {
	panic("excised: HTTPServiceExpr.IsJSONRPC")
}

// Prepare initializes the error responses.
func (svc *HTTPServiceExpr) Prepare() {
	panic("excised: HTTPServiceExpr.Prepare")
}

// Validate makes sure the service is valid.
func (svc *HTTPServiceExpr) Validate() error {
	panic("excised: HTTPServiceExpr.Validate")
}

// validateAttributes validates service parameters and headers
func (svc *HTTPServiceExpr) validateAttributes(verr *eval.ValidationErrors) {
	panic("excised: HTTPServiceExpr.validateAttributes")
}

// validateParent validates parent service configuration
func (svc *HTTPServiceExpr) validateParent(verr *eval.ValidationErrors) {
	panic("excised: HTTPServiceExpr.validateParent")
}

// validateCanonicalEndpoint validates canonical endpoint configuration
func (svc *HTTPServiceExpr) validateCanonicalEndpoint(verr *eval.ValidationErrors) {
	panic("excised: HTTPServiceExpr.validateCanonicalEndpoint")
}

// validateErrors validates HTTP errors
func (svc *HTTPServiceExpr) validateErrors(verr *eval.ValidationErrors) {
	panic("excised: HTTPServiceExpr.validateErrors")
}

// validateTransports validates JSON-RPC route constraints.
func (svc *HTTPServiceExpr) validateTransports(verr *eval.ValidationErrors) {
	panic("excised: HTTPServiceExpr.validateTransports")
}

// Finalize initializes the path if no path is set in design.
func (svc *HTTPServiceExpr) Finalize() {
	panic("excised: HTTPServiceExpr.Finalize")
}

// prepareJSONRPCRoutes creates routes for all JSON-RPC endpoints.
// All JSON-RPC methods share the same route.
func (svc *HTTPServiceExpr) prepareJSONRPCRoutes() {
	panic("excised: HTTPServiceExpr.prepareJSONRPCRoutes")
}

// validateJSONRPCRoutes checks that every JSON-RPC route uses POST.
func (svc *HTTPServiceExpr) validateJSONRPCRoutes(verr *eval.ValidationErrors) {
	panic("excised: HTTPServiceExpr.validateJSONRPCRoutes")
}
