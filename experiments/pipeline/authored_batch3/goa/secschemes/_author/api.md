# Exported API — secschemes

```go
type SchemeKind int   // OAuth2Kind, BasicAuthKind, APIKeyKind, JWTKind, NoKind, BearerKind
type FlowKind int     // AuthorizationCodeFlowKind, ImplicitFlowKind, PasswordFlowKind, ClientCredentialsFlowKind

type SecurityExpr struct {
    Schemes []*SchemeExpr
    Scopes  []string
}

type SchemeExpr struct {
    Kind         SchemeKind
    SchemeName   string
    Description  string
    In           string        // "header" or "query"
    Name         string        // header or parameter name
    BearerFormat string
    Scopes       []*ScopeExpr
    Flows        []*FlowExpr
    Meta         MetaExpr
}

type FlowExpr struct {
    Kind             FlowKind
    AuthorizationURL string
    TokenURL         string
    RefreshURL       string
}

type ScopeExpr struct{ Name, Description string }

func (s *SecurityExpr) EvalName() string
func DupRequirement(req *SecurityExpr) *SecurityExpr
func DupScheme(sch *SchemeExpr) *SchemeExpr
func (s *SchemeExpr) AuthoredScheme() *SchemeExpr
func HasNoSecurity(reqs []*SecurityExpr) bool
func EffectiveSecurityRequirements(reqs []*SecurityExpr) []*SecurityExpr
func (s *SchemeExpr) Type() string
func (s *SchemeExpr) EvalName() string
func (s *SchemeExpr) Hash() string
func (s *SchemeExpr) Validate() *eval.ValidationErrors
func (f *FlowExpr) EvalName() string
func (f *FlowExpr) Validate() *eval.ValidationErrors
func (f *FlowExpr) Type() string
func (k SchemeKind) String() string
```

## Pre-existing callers

The DSL's `Security()`/`BasicAuthSecurity()`/`OAuth2Security()`/etc. build
`SchemeExpr`/`FlowExpr` values that the eval engine validates via
`SchemeExpr.Validate`. `MethodExpr.Finalize` copies requirements through
`DupRequirement` and resolves `EffectiveSecurityRequirements`. codegen reads
`Type()`, `SchemeKind.String()` and `Hash()` when naming generated types and
security schemes.
