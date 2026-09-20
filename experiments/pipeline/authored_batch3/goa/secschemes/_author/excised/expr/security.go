package expr

import (
	_ "fmt"
	_ "net/url"

	"example.internal/apikit/v3/eval"
)

// SchemeKind is a type of security scheme.
type SchemeKind int

const (
	// OAuth2Kind identifies a "OAuth2" security scheme.
	OAuth2Kind SchemeKind = iota + 1
	// BasicAuthKind means "basic" security scheme.
	BasicAuthKind
	// APIKeyKind means "apiKey" security scheme.
	APIKeyKind
	// JWTKind means a "JWT" security scheme with support for scopes.
	JWTKind
	// NoKind means to have no security for this endpoint.
	NoKind
	// BearerKind means an HTTP "bearer" security scheme with support for
	// scopes.
	BearerKind
)

// FlowKind is a type of OAuth2 flow.
type FlowKind int

const (
	// AuthorizationCodeFlowKind identifies a OAuth2 authorization code
	// flow.
	AuthorizationCodeFlowKind FlowKind = iota + 1
	// ImplicitFlowKind identifiers a OAuth2 implicit flow.
	ImplicitFlowKind
	// PasswordFlowKind identifies a Resource Owner Password flow.
	PasswordFlowKind
	// ClientCredentialsFlowKind identifies a OAuth Client Credentials flow.
	ClientCredentialsFlowKind
)

type (
	// SecurityHolder is an interface that allows expression types to receive
	// security requirements. Types implementing this interface can use the
	// Security() DSL function to add security schemes.
	SecurityHolder interface {
		AddSecurityRequirement(*SecurityExpr)
	}

	// SecurityExpr defines a security requirement.
	SecurityExpr struct {
		// Schemes is the list of security schemes used for this
		// requirement.
		Schemes []*SchemeExpr
		// Scopes list the required scopes if any.
		Scopes []string
	}

	// SchemeExpr defines a security scheme used to authenticate against the
	// method being designed.
	SchemeExpr struct {
		// Kind is the sort of security scheme this object represents.
		Kind SchemeKind
		// SchemeName is the name of the security scheme, e.g. "googAuth",
		// "my_big_token", "jwt".
		SchemeName string
		// Description describes the security scheme e.g. "Google OAuth2"
		Description string
		// In determines the location of the API key, one of "header" or
		// "query".
		In string
		// Name refers to a header or parameter name, based on In's
		// value.
		Name string
		// BearerFormat is a hint to identify how the bearer token is
		// formatted. It is emitted in OpenAPI v3 bearerFormat when set.
		BearerFormat string
		// Scopes lists the Basic, APIKey, Bearer, JWT or OAuth2 scopes.
		Scopes []*ScopeExpr
		// Flows determine the oauth2 flows supported by this scheme.
		Flows []*FlowExpr
		// Meta is a list of key/value pairs
		Meta MetaExpr
		// authored points to the security scheme copied for a transport. It is
		// nil while this value is the scheme declared by the design.
		authored *SchemeExpr
	}

	// FlowExpr describes a specific OAuth2 flow.
	FlowExpr struct {
		// Kind is the kind of flow.
		Kind FlowKind
		// AuthorizationURL to be used for implicit or authorizationCode
		// flows.
		AuthorizationURL string
		// TokenURL to be used for password, clientCredentials or
		// authorizationCode flows.
		TokenURL string
		// RefreshURL to be used for obtaining refresh token.
		RefreshURL string
	}

	// ScopeExpr defines a security scope.
	ScopeExpr struct {
		// Name of the scope.
		Name string
		// Description is the description of the scope.
		Description string
	}
)

// EvalName returns the generic definition name used in error messages.
func (s *SecurityExpr) EvalName() string {
	panic("excised: SecurityExpr.EvalName")
}

// DupRequirement creates a copy of the given security requirement.
func DupRequirement(req *SecurityExpr) *SecurityExpr {
	panic("excised: DupRequirement")
}

// DupScheme creates a copy of the given scheme expression.
func DupScheme(sch *SchemeExpr) *SchemeExpr {
	panic("excised: DupScheme")
}

// AuthoredScheme returns the security scheme declared by the design. It
// returns s when s has not been copied for a transport.
func (s *SchemeExpr) AuthoredScheme() *SchemeExpr {
	panic("excised: SchemeExpr.AuthoredScheme")
}

// HasNoSecurity returns true if the security requirements explicitly disable
// security.
func HasNoSecurity(reqs []*SecurityExpr) bool {
	panic("excised: HasNoSecurity")
}

// EffectiveSecurityRequirements returns the security requirements that should
// be enforced. It returns nil when reqs explicitly disable security.
func EffectiveSecurityRequirements(reqs []*SecurityExpr) []*SecurityExpr {
	panic("excised: EffectiveSecurityRequirements")
}

// Type returns the type of the scheme.
func (s *SchemeExpr) Type() string {
	panic("excised: SchemeExpr.Type")
}

// EvalName returns the generic definition name used in error messages.
func (s *SchemeExpr) EvalName() string {
	panic("excised: SchemeExpr.EvalName")
}

// Hash returns a unique hash value for s.
func (s *SchemeExpr) Hash() string {
	panic("excised: SchemeExpr.Hash")
}

// Validate ensures that the method payload contains attributes required
// by the scheme.
func (s *SchemeExpr) Validate() *eval.ValidationErrors {
	panic("excised: SchemeExpr.Validate")
}

// EvalName returns the name of the expression used in error messages.
func (f *FlowExpr) EvalName() string {
	panic("excised: FlowExpr.EvalName")
}

// Validate ensures that TokenURL and AuthorizationURL are valid URLs.
func (f *FlowExpr) Validate() *eval.ValidationErrors {
	panic("excised: FlowExpr.Validate")
}

// Type returns the grant type of the OAuth2 grant.
func (f *FlowExpr) Type() string {
	panic("excised: FlowExpr.Type")
}

func (k SchemeKind) String() string {
	panic("excised: SchemeKind.String")
}
