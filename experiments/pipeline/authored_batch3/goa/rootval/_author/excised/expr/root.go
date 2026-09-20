// This file defines the evaluated design root and validates relationships
// between its API, services, generated types, and explicitly relocated user
// types before code generation begins.
package expr

import (
	_ "cmp"
	"fmt"
	_ "maps"
	_ "reflect"
	_ "slices"
	_ "sort"
	_ "strings"

	"example.internal/apikit/v3/eval"
	goa "example.internal/apikit/v3/pkg"
)

// Root is the root object built by the DSL.
var Root = new(RootExpr)

// DefaultProtoc is the default command to be invoked for generating code from protobuf schemas.
const DefaultProtoc = "protoc"

type (
	// RootExpr is the struct built by the DSL on process start.
	RootExpr struct {
		// API contains the API expression built by the DSL.
		API *APIExpr
		// Services contains the list of services exposed by the API.
		Services []*ServiceExpr
		// Interceptors contains the list of interceptors.
		Interceptors []*InterceptorExpr
		// Errors contains reusable error definitions. Services and methods
		// select these errors by naming them in their own Error DSL.
		Errors []*ErrorExpr
		// Types contains the user types described in the DSL.
		Types []UserType
		// ResultTypes contains the result types generated during DSL
		// execution.
		ResultTypes []*ResultTypeExpr
		// Conversions list the user type to external type mappings.
		Conversions []*TypeMap
		// Creations list the external type to user type mappings.
		Creations []*TypeMap
		// Schemes list the registered security schemes.
		Schemes []*SchemeExpr
	}

	// MetaExpr is a set of key/value pairs
	MetaExpr map[string][]string

	// TypeMap defines a user to external type mapping.
	TypeMap struct {
		// User is the user type being converted or created.
		User UserType

		// External is an instance of the type being converted from or to.
		External any
	}

	// mountedHTTPRoute identifies one final route and the service that owns it.
	mountedHTTPRoute struct {
		service string
		route   *RouteExpr
		path    string
	}
)

// WalkSets returns the expressions in order of evaluation.
func (r *RootExpr) WalkSets(walk eval.SetWalker) {
	panic("excised: RootExpr.WalkSets")
}

// DependsOn returns nil, the core DSL has no dependency.
func (*RootExpr) DependsOn() []eval.Root { return nil }

// Packages returns the Go import path to this and the dsl packages.
func (*RootExpr) Packages() []string {
	return []string{
		"example.internal/apikit/v3/expr",
		"example.internal/apikit/v3/dsl",
		fmt.Sprintf("example.internal/apikit/v3@%s/expr", goa.Version()),
		fmt.Sprintf("example.internal/apikit/v3@%s/dsl", goa.Version()),
	}
}

// UserType returns the user type expression with the given name if found, nil otherwise.
func (r *RootExpr) UserType(name string) UserType {
	panic("excised: RootExpr.UserType")
}

// Service returns the service with the given name.
func (r *RootExpr) Service(name string) *ServiceExpr {
	panic("excised: RootExpr.Service")
}

// Error returns the error with the given name.
func (r *RootExpr) Error(name string) *ErrorExpr {
	panic("excised: RootExpr.Error")
}

// EvalName is the name of the DSL.
func (*RootExpr) EvalName() string {
	return "design"
}

// Validate makes sure the root expression is valid for code generation.
func (r *RootExpr) Validate() error {
	panic("excised: RootExpr.Validate")
}

// ValidateSharedErrorNames rejects an authored type with several static error
// names unless each value carries its selected name in an ErrorName field. The
// roots must include every design linked by one generation command.
func ValidateSharedErrorNames(roots ...*RootExpr) *eval.ValidationErrors {
	panic("excised: ValidateSharedErrorNames")
}

// hasErrorNameAttribute reports whether the type stores its error name in one
// generated field.
func hasErrorNameAttribute(attribute *AttributeExpr) bool {
	panic("excised: hasErrorNameAttribute")
}

// validateErrorDefaults checks each declared error once. Services and methods
// may reuse the same error expression, so pointer identity prevents repeated
// diagnostics for one declaration.
func (r *RootExpr) validateErrorDefaults() *eval.ValidationErrors {
	panic("excised: RootExpr.validateErrorDefaults")
}

// validateSharedHTTPRoutes rejects ordinary HTTP and JSON-RPC routes that
// would replace one another when mounted on the same server.
func (r *RootExpr) validateSharedHTTPRoutes() *eval.ValidationErrors {
	panic("excised: RootExpr.validateSharedHTTPRoutes")
}

// routesMountedOnServer returns every final route owned by services mounted on
// server.
func routesMountedOnServer(server *ServerExpr, services []*HTTPServiceExpr) []mountedHTTPRoute {
	panic("excised: routesMountedOnServer")
}

// serverHostsService reports whether server mounts service. An empty service
// list means that the server mounts every service in the design.
func serverHostsService(server *ServerExpr, service string) bool {
	panic("excised: serverHostsService")
}

// routePatternWithoutParameterNames returns the path form used to detect
// router replacement. Parameter names do not change which requests match.
func routePatternWithoutParameterNames(routePath string) string {
	panic("excised: routePatternWithoutParameterNames")
}

// validateTypeMappings rejects repeated declarations that would generate the
// same method on one user type. A reflected type includes its package path, so
// equally named external types from different packages remain distinct.
func validateTypeMappings(direction string, mappings []*TypeMap) *eval.ValidationErrors {
	panic("excised: validateTypeMappings")
}

// validateRelocatedUserTypes enforces that relocated user types (those with
// `struct:pkg:path`) only depend on other declared user types with an explicit
// generation location.
//
// Without this constraint, generated code would need to reference service-local
// user types across packages, which is not safe (it either fails to compile or
// forces import cycles). Generated/derived user types (e.g. union branch
// wrappers) are exempt because they are materialized alongside their owning
// types.
func (r *RootExpr) validateRelocatedUserTypes() *eval.ValidationErrors {
	panic("excised: RootExpr.validateRelocatedUserTypes")
}

// walkUserTypeDependencies traverses the attribute graph reachable from root and
// invokes visit for each encountered user type.
func (r *RootExpr) walkUserTypeDependencies(root UserType, seen map[UserType]struct{}, path string, visit func(UserType, string)) {
	panic("excised: RootExpr.walkUserTypeDependencies")
}

// walkAttributeUserTypes traverses att and invokes visit for each encountered
// user type.
//
// The path argument records the traversal path through objects, arrays, maps,
// and unions and is intended for diagnostics.
func (r *RootExpr) walkAttributeUserTypes(att *AttributeExpr, seen map[UserType]struct{}, path string, visit func(UserType, string)) {
	panic("excised: RootExpr.walkAttributeUserTypes")
}

// Finalize finalizes the server expressions.
func (r *RootExpr) Finalize() {
	panic("excised: RootExpr.Finalize")
}

// walkHTTPServices walks the HTTP services and endpoints.
func (r *RootExpr) walkHTTPServices(root eval.Expression, svcs []*HTTPServiceExpr, walk eval.SetWalker) {
	panic("excised: RootExpr.walkHTTPServices")
}

// Dup creates a new map from the given expression.
func (m MetaExpr) Dup() MetaExpr {
	panic("excised: MetaExpr.Dup")
}

// Merge merges src meta expression with m. If meta has intersecting set of
// keys on both m and src, then the values for those keys in src is appended
// to the values of the keys in m if not already existing.
func (m MetaExpr) Merge(src MetaExpr) {
	panic("excised: MetaExpr.Merge")
}

// Last returns the last value for a specific key, if the key exists and has
// values; otherwise returns an empty string, with the "ok" flag set to false.
func (m MetaExpr) Last(key string) (string, bool) {
	panic("excised: MetaExpr.Last")
}
