package expr_test

// Hidden black-box suite for unit "secschemes".
// TestDetailNN numbers match DETAILS.md lines 1..12.
// Inferable:no lines assert shape only (presence/structure/relations),
// never the committed literal.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/expr"
)

var bbSecSchemesKinds = []expr.SchemeKind{
	expr.OAuth2Kind, expr.BasicAuthKind, expr.APIKeyKind, expr.JWTKind, expr.BearerKind,
}

// Detail 1 (Inferable: no): two name spellings exist and disagree on
// exactly one kind. Assert injectivity + the relational shape only.
func TestDetail01(t *testing.T) {
	types := map[string]expr.SchemeKind{}
	for _, k := range bbSecSchemesKinds {
		v := (&expr.SchemeExpr{Kind: k}).Type()
		require.NotEmpty(t, v, "Type() empty for kind %d", int(k))
		if prev, dup := types[v]; dup {
			t.Fatalf("Type() collision: kinds %d and %d both %q", int(prev), int(k), v)
		}
		types[v] = k
	}
	strs := map[string]expr.SchemeKind{}
	for _, k := range append(append([]expr.SchemeKind{}, bbSecSchemesKinds...), expr.NoKind) {
		v := k.String()
		require.NotEmpty(t, v, "String() empty for kind %d", int(k))
		if prev, dup := strs[v]; dup {
			t.Fatalf("String() collision: kinds %d and %d both %q", int(prev), int(k), v)
		}
		strs[v] = k
	}
	disagree := 0
	for _, k := range bbSecSchemesKinds {
		if (&expr.SchemeExpr{Kind: k}).Type() != k.String() {
			disagree++
		}
	}
	require.Equal(t, 1, disagree, "Type() and String() must disagree on exactly one kind")
}

// Detail 2 (Inferable: no): unknown kinds rejected loudly by both
// lookups; Type()'s domain excludes the no-security kind, String()'s
// does not. Shape = panic as rejection mechanism, not messages.
func TestDetail02(t *testing.T) {
	for _, k := range []expr.SchemeKind{expr.SchemeKind(0), expr.SchemeKind(97), expr.SchemeKind(-1)} {
		assert.Panics(t, func() { _ = k.String() }, "String(%d) must panic", int(k))
		assert.Panics(t, func() { _ = (&expr.SchemeExpr{Kind: k}).Type() }, "Type(%d) must panic", int(k))
	}
	assert.Panics(t, func() { _ = (&expr.SchemeExpr{Kind: expr.NoKind}).Type() })
	require.NotEmpty(t, expr.NoKind.String())
}

// Detail 3 (Inferable: no): requirement eval name identifies the first
// scheme when named; degenerate scheme lists share one fixed name.
func TestDetail03(t *testing.T) {
	empty := (&expr.SecurityExpr{}).EvalName()
	require.NotEmpty(t, empty)
	require.Equal(t, empty, (&expr.SecurityExpr{Schemes: nil}).EvalName())
	require.Equal(t, empty, (&expr.SecurityExpr{Schemes: []*expr.SchemeExpr{}}).EvalName())
	require.Equal(t, empty, (&expr.SecurityExpr{Schemes: []*expr.SchemeExpr{{}}}).EvalName())

	named := (&expr.SecurityExpr{Schemes: []*expr.SchemeExpr{
		{SchemeName: "bbalpha"}, {SchemeName: "bbbeta"},
	}}).EvalName()
	require.NotEmpty(t, named)
	require.NotEqual(t, empty, named)
	require.Contains(t, named, "bbalpha")
}

// Detail 4 (Inferable: partially): composed eval names — assert the
// composition, which is derivable, not the literal type strings.
func TestDetail04(t *testing.T) {
	for _, k := range bbSecSchemesKinds {
		sch := &expr.SchemeExpr{Kind: k}
		require.Equal(t, sch.Type()+"Security", sch.EvalName(), "scheme eval name must append Security to Type()")
	}
	for _, fk := range []expr.FlowKind{
		expr.AuthorizationCodeFlowKind, expr.ImplicitFlowKind, expr.PasswordFlowKind, expr.ClientCredentialsFlowKind,
	} {
		f := &expr.FlowExpr{Kind: fk}
		require.Equal(t, "flow "+f.Type(), f.EvalName(), "flow eval name must prepend 'flow ' to Type()")
	}
}

// Detail 5 (Inferable: doc): OAuth2 grant-type strings are the standard
// names; unknown flow kinds panic.
func TestDetail05(t *testing.T) {
	require.Equal(t, "authorization_code", (&expr.FlowExpr{Kind: expr.AuthorizationCodeFlowKind}).Type())
	require.Equal(t, "implicit", (&expr.FlowExpr{Kind: expr.ImplicitFlowKind}).Type())
	require.Equal(t, "password", (&expr.FlowExpr{Kind: expr.PasswordFlowKind}).Type())
	require.Equal(t, "client_credentials", (&expr.FlowExpr{Kind: expr.ClientCredentialsFlowKind}).Type())
	assert.Panics(t, func() { _ = (&expr.FlowExpr{Kind: expr.FlowKind(0)}).Type() })
	assert.Panics(t, func() { _ = (&expr.FlowExpr{Kind: expr.FlowKind(42)}).Type() })
}

// Detail 6 (Inferable: no): each malformed URL is its own error naming
// the URL; empty URLs accepted. Shape = per-URL errors + count.
func TestDetail06(t *testing.T) {
	verr := (&expr.FlowExpr{Kind: expr.AuthorizationCodeFlowKind}).Validate()
	require.True(t, verr == nil || len(verr.Errors) == 0, "empty-URL flow must validate clean, got %v", verr)

	f := &expr.FlowExpr{
		Kind:             expr.AuthorizationCodeFlowKind,
		AuthorizationURL: "://bb-bad-auth",
		TokenURL:         "://bb-bad-token",
		RefreshURL:       "://bb-bad-refresh",
	}
	verr = f.Validate()
	require.NotNil(t, verr)
	require.Len(t, verr.Errors, 3, "three malformed URLs must yield three errors")
	msg := verr.Error()
	for _, u := range []string{"://bb-bad-auth", "://bb-bad-token", "://bb-bad-refresh"} {
		require.Contains(t, msg, u, "error must name the offending URL %q", u)
	}

	verr = (&expr.FlowExpr{Kind: expr.ImplicitFlowKind, AuthorizationURL: "://bb-one"}).Validate()
	require.NotNil(t, verr)
	require.Len(t, verr.Errors, 1)
	require.Contains(t, verr.Error(), "://bb-one")
}

// Detail 7 (Inferable: partially): scheme validation merges every flow's
// errors, in flow order.
func TestDetail07(t *testing.T) {
	sch := &expr.SchemeExpr{
		Kind: expr.OAuth2Kind,
		Flows: []*expr.FlowExpr{
			{Kind: expr.ImplicitFlowKind, AuthorizationURL: "://bb-first"},
			{Kind: expr.PasswordFlowKind, TokenURL: "://bb-second"},
		},
	}
	verr := sch.Validate()
	require.NotNil(t, verr)
	require.Len(t, verr.Errors, 2)
	msg := verr.Error()
	require.Contains(t, msg, "://bb-first")
	require.Contains(t, msg, "://bb-second")
	require.Less(t, strings.Index(msg, "://bb-first"), strings.Index(msg, "://bb-second"),
		"flow errors must appear in flow order")

	clean := &expr.SchemeExpr{Kind: expr.OAuth2Kind, Flows: []*expr.FlowExpr{{Kind: expr.ImplicitFlowKind}}}
	verr = clean.Validate()
	require.True(t, verr == nil || len(verr.Errors) == 0, "valid scheme must validate clean, got %v", verr)
}

// Detail 8 (Inferable: no): DupRequirement returns a distinct requirement
// whose schemes are deep copies; scopes content preserved.
func TestDetail08(t *testing.T) {
	req := &expr.SecurityExpr{
		Schemes: []*expr.SchemeExpr{
			{Kind: expr.BasicAuthKind, SchemeName: "bbone"},
			{Kind: expr.JWTKind, SchemeName: "bbtwo"},
		},
		Scopes: []string{"bbs1", "bbs2"},
	}
	dup := expr.DupRequirement(req)
	require.NotSame(t, req, dup)
	require.Len(t, dup.Schemes, 2)
	for i := range req.Schemes {
		require.NotSame(t, req.Schemes[i], dup.Schemes[i], "scheme %d must be a copy", i)
		require.Equal(t, req.Schemes[i].SchemeName, dup.Schemes[i].SchemeName)
		require.Equal(t, req.Schemes[i].Kind, dup.Schemes[i].Kind)
	}
	require.Equal(t, req.Scopes, dup.Scopes)
}

// Detail 9 (Inferable: no): DupScheme yields a distinct copy with equal
// scalar fields; copies resolve AuthoredScheme to the declared original,
// including copy-of-a-copy.
func TestDetail09(t *testing.T) {
	src := &expr.SchemeExpr{
		Kind: expr.APIKeyKind, SchemeName: "bbkey", In: "header", Name: "X-Bb-Key",
		Description: "bb desc", BearerFormat: "bbf",
		Scopes: []*expr.ScopeExpr{{Name: "bbs"}},
		Flows:  []*expr.FlowExpr{{Kind: expr.ImplicitFlowKind}},
		Meta:   expr.MetaExpr{"bbm": []string{"v"}},
	}
	dup := expr.DupScheme(src)
	require.NotSame(t, src, dup)
	require.Equal(t, src.SchemeName, dup.SchemeName)
	require.Equal(t, src.Kind, dup.Kind)
	require.Equal(t, src.In, dup.In)
	require.Equal(t, src.Name, dup.Name)
	require.Equal(t, src.Description, dup.Description)
	require.Equal(t, src.BearerFormat, dup.BearerFormat)
	require.Equal(t, len(src.Scopes), len(dup.Scopes))
	require.Equal(t, len(src.Flows), len(dup.Flows))

	// a copy resolves to the declared original, and so does a copy of it
	require.Same(t, src, dup.AuthoredScheme())
	dup2 := expr.DupScheme(dup)
	require.Same(t, src, dup2.AuthoredScheme(), "copy-of-copy must still resolve to the original")
}

// Detail 10 (Inferable: doc): AuthoredScheme returns the receiver when
// the scheme was never copied for a transport.
func TestDetail10(t *testing.T) {
	s := &expr.SchemeExpr{Kind: expr.JWTKind, SchemeName: "bbjwt"}
	require.Same(t, s, s.AuthoredScheme())
}

// Detail 11 (Inferable: partially): HasNoSecurity is true when ANY scheme
// in ANY requirement is the no-security kind; EffectiveSecurityRequirements
// returns nil exactly then and the input otherwise.
func TestDetail11(t *testing.T) {
	basic := &expr.SchemeExpr{Kind: expr.BasicAuthKind}
	none := &expr.SchemeExpr{Kind: expr.NoKind}

	plain := []*expr.SecurityExpr{{Schemes: []*expr.SchemeExpr{basic}}}
	require.False(t, expr.HasNoSecurity(plain))
	require.Equal(t, plain, expr.EffectiveSecurityRequirements(plain))

	secondReq := []*expr.SecurityExpr{
		{Schemes: []*expr.SchemeExpr{basic}},
		{Schemes: []*expr.SchemeExpr{none}},
	}
	require.True(t, expr.HasNoSecurity(secondReq))
	require.Nil(t, expr.EffectiveSecurityRequirements(secondReq))

	secondScheme := []*expr.SecurityExpr{{Schemes: []*expr.SchemeExpr{basic, none}}}
	require.True(t, expr.HasNoSecurity(secondScheme), "NoKind in a non-first position still disables")

	require.False(t, expr.HasNoSecurity(nil))
	require.Nil(t, expr.EffectiveSecurityRequirements(nil))
}

// Detail 12 (Inferable: no): Hash is a deterministic value sensitive to
// each of scheme name, location and element name.
func TestDetail12(t *testing.T) {
	base := &expr.SchemeExpr{Kind: expr.APIKeyKind, SchemeName: "bbsch", In: "header", Name: "X-Bb"}
	h := base.Hash()
	require.NotEmpty(t, h)
	require.Equal(t, h, base.Hash(), "hash must be deterministic")

	variants := []*expr.SchemeExpr{
		{Kind: expr.APIKeyKind, SchemeName: "bbother", In: "header", Name: "X-Bb"},
		{Kind: expr.APIKeyKind, SchemeName: "bbsch", In: "query", Name: "X-Bb"},
		{Kind: expr.APIKeyKind, SchemeName: "bbsch", In: "header", Name: "X-Bb-Other"},
	}
	for i, v := range variants {
		require.NotEqual(t, h, v.Hash(), "variant %d must change the hash", i)
	}
}
