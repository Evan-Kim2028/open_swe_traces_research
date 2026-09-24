package expr_test

// Hidden black-box suite for unit "svcexpr".
// TestDetailNN numbers match DETAILS.md lines 1..12.
// Inferable:no lines assert shape only (presence/structure/relations),
// never the committed literal.

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/eval"
	"example.internal/apikit/v3/expr"
)

var bbSvcexprValidSchemes = []string{"http", "https", "grpc", "grpcs"}

// bbSvcexprErrs unwraps an error produced by Validate methods: a non-nil
// *eval.ValidationErrors may still hold zero entries.
func bbSvcexprErrs(t *testing.T, err error) []error {
	t.Helper()
	if err == nil {
		return nil
	}
	var ve *eval.ValidationErrors
	require.ErrorAs(t, err, &ve)
	return ve.Errors
}

// Detail 1 (Inferable: no): scheme is decided by prefix; the four real
// schemes round-trip including the prefix-ambiguous grpcs; anything else
// still reports one of the documented schemes.
func TestDetail01(t *testing.T) {
	for _, s := range bbSvcexprValidSchemes {
		require.Equal(t, s, expr.URIExpr(s+"://bb-host.example").Scheme(), "scheme of %q", s)
	}
	// the grpcs/grpc prefix overlap must resolve to the longer scheme
	require.Equal(t, "grpcs", expr.URIExpr("grpcs://bb:443").Scheme())
	// malformed or bare input must not escape the documented codomain
	for _, u := range []expr.URIExpr{"bb-nonsense", "", "://", "ftp://bb"} {
		got := u.Scheme()
		require.Contains(t, bbSvcexprValidSchemes, got, "URIExpr(%q).Scheme() = %q outside documented set", string(u), got)
	}
}

// Detail 2 (Inferable: no): every {…} group is extracted in order; a
// wildcard parameter reports a name identifying it.
func TestDetail02(t *testing.T) {
	params := expr.URIExpr("http://bb/{p1}/x/{p2}/{*wild}/y").Params()
	require.Len(t, params, 3)
	require.Equal(t, "p1", params[0])
	require.Equal(t, "p2", params[1])
	require.Equal(t, "wild", strings.TrimPrefix(params[2], "*"), "wildcard param must identify 'wild'")

	require.Empty(t, expr.URIExpr("http://bb/plain").Params())
	require.Equal(t, []string{"a"}, expr.URIExpr("{a}").Params())
}

// Detail 3 (Inferable: no): URIString resolves a listed URI substituting
// each parameter's default (doc-commented behavior) and errors on a URI
// not in the host.
func TestDetail03(t *testing.T) {
	host := &expr.HostExpr{
		URIs: []expr.URIExpr{"http://bb/{p}/{e}"},
		Variables: &expr.AttributeExpr{Type: &expr.Object{
			{Name: "p", Attribute: &expr.AttributeExpr{Type: expr.String, DefaultValue: "bbdef"}},
			{Name: "e", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"bbe1", "bbe2"}}}},
		}},
	}
	got, err := host.URIString("http://bb/{p}/{e}")
	require.NoError(t, err)
	require.Contains(t, got, "bbdef", "param with a default substitutes the default")
	require.Contains(t, got, "bbe1", "param with an enum substitutes the first enum value")
	require.NotContains(t, got, "{p}")

	_, err = host.URIString("http://bb/not-listed")
	require.Error(t, err)
	require.Contains(t, err.Error(), "http://bb/not-listed", "error must name the URI")
}

// Detail 4 (Inferable: partially): host validation masks {var} before
// parsing — assert each defect class produces an error naming the URI.
func TestDetail04(t *testing.T) {
	require.NotEmpty(t, bbSvcexprErrs(t, (&expr.HostExpr{}).Validate()), "empty URI list must error")

	malformed := bbSvcexprErrs(t, (&expr.HostExpr{URIs: []expr.URIExpr{"://bb-bad"}}).Validate())
	require.NotEmpty(t, malformed)
	require.Contains(t, fmt.Sprint(malformed), "://bb-bad", "error must name the malformed URI")

	badScheme := bbSvcexprErrs(t, (&expr.HostExpr{URIs: []expr.URIExpr{"ftp://bb"}}).Validate())
	require.NotEmpty(t, badScheme)
	require.Contains(t, fmt.Sprint(badScheme), "ftp", "error must identify the scheme")

	noScheme := bbSvcexprErrs(t, (&expr.HostExpr{URIs: []expr.URIExpr{"//bb-only-path"}}).Validate())
	require.NotEmpty(t, noScheme)

	ok := bbSvcexprErrs(t, (&expr.HostExpr{URIs: []expr.URIExpr{"http://bb", "grpcs://bb2/{v}"},
		Variables: &expr.AttributeExpr{Type: &expr.Object{
			{Name: "v", Attribute: &expr.AttributeExpr{Type: expr.String, DefaultValue: "x"}},
		}}}).Validate())
	require.Empty(t, ok, "valid URIs incl. a {var} placeholder must validate clean, got %v", ok)
}

// Detail 5 (Inferable: no): each URI variable must be primitive with a
// default or a non-empty enum; assert errors name the offending variable.
func TestDetail05(t *testing.T) {
	vars := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "objv", Attribute: &expr.AttributeExpr{Type: &expr.Object{}}},
		{Name: "barev", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "emptyenum", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{}}},
		{Name: "enumv", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"a"}}}},
		{Name: "defv", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 3}},
	}}
	h := &expr.HostExpr{
		URIs:      []expr.URIExpr{"http://bb/{objv}/{barev}/{emptyenum}/{enumv}/{defv}"},
		Variables: vars,
	}
	errs := bbSvcexprErrs(t, h.Validate())
	require.NotEmpty(t, errs)
	msg := fmt.Sprint(errs)
	for _, bad := range []string{"objv", "barev", "emptyenum"} {
		require.Contains(t, msg, bad, "error must name offending variable %q", bad)
	}
	require.NotContains(t, msg, "enumv")
	require.NotContains(t, msg, "defv")
}

// Detail 6 (Inferable: partially): scheme aggregations return sorted
// names covering exactly the present schemes.
func TestDetail06(t *testing.T) {
	h := &expr.HostExpr{URIs: []expr.URIExpr{"grpc://bb1", "http://bb2", "grpc://bb3", "grpcs://bb4"}}
	got := h.Schemes()
	require.True(t, sort.StringsAreSorted(got), "host schemes must be sorted, got %v", got)
	for _, s := range []string{"http", "grpc", "grpcs"} {
		require.Contains(t, got, s)
	}
	for _, s := range got {
		require.Contains(t, bbSvcexprValidSchemes, s)
	}
	require.LessOrEqual(t, len(got), 3, "schemes must not outnumber distinct schemes")

	s := &expr.ServerExpr{Hosts: []*expr.HostExpr{
		{URIs: []expr.URIExpr{"grpc://bb1"}},
		{URIs: []expr.URIExpr{"http://bb2", "grpc://bb3"}},
	}}
	got = s.Schemes()
	require.True(t, sort.StringsAreSorted(got))
	require.Contains(t, got, "http")
	require.Contains(t, got, "grpc")
}

// Detail 7 (Inferable: doc): HasHTTPScheme = any http/https URI;
// HasGRPCScheme = any grpc/grpcs URI.
func TestDetail07(t *testing.T) {
	require.True(t, (&expr.HostExpr{URIs: []expr.URIExpr{"http://bb"}}).HasHTTPScheme())
	require.True(t, (&expr.HostExpr{URIs: []expr.URIExpr{"https://bb"}}).HasHTTPScheme())
	require.False(t, (&expr.HostExpr{URIs: []expr.URIExpr{"grpc://bb"}}).HasHTTPScheme())
	require.True(t, (&expr.HostExpr{URIs: []expr.URIExpr{"grpcs://bb"}}).HasGRPCScheme())
	require.True(t, (&expr.HostExpr{URIs: []expr.URIExpr{"grpc://bb"}}).HasGRPCScheme())
	require.False(t, (&expr.HostExpr{URIs: []expr.URIExpr{"https://bb"}}).HasGRPCScheme())
	require.False(t, (&expr.HostExpr{}).HasHTTPScheme())
	require.False(t, (&expr.HostExpr{}).HasGRPCScheme())
}

// Detail 8 (Inferable: no): Finalize and Attribute lazily install an
// empty object attribute when Variables is nil.
func TestDetail08(t *testing.T) {
	h := &expr.HostExpr{URIs: []expr.URIExpr{"http://bb"}}
	h.Finalize()
	require.NotNil(t, h.Variables)
	require.True(t, expr.IsObject(h.Variables.Type), "installed Variables must be object-typed")

	h2 := &expr.HostExpr{}
	att := h2.Attribute()
	require.NotNil(t, att)
	require.True(t, expr.IsObject(att.Type))
}

// Detail 9 (Inferable: no): server finalize defaults empty Services to
// the design's services and an empty host list to a usable host — assert
// coverage and scheme validity, not literal names/URIs.
func TestDetail09(t *testing.T) {
	expr.RunDSL(t, func() {
		API("bbapi", func() {
			Server("bbsrv", func() {
				Host("bbh", func() { URI("http://bb-host:9") })
			})
		})
		Service("bbs1", func() {
			Method("bbm1", func() { HTTP(func() { GET("/") }) })
		})
		Service("bbs2", func() {
			Method("bbm2", func() { HTTP(func() { GET("/x") }) })
		})
	})
	var srv *expr.ServerExpr
	for _, s := range expr.Root.API.Servers {
		if s.Name == "bbsrv" {
			srv = s
		}
	}
	require.NotNil(t, srv, "declared server must exist")
	require.ElementsMatch(t, []string{"bbs1", "bbs2"}, srv.Services,
		"empty Services must default to every service in the design")
	require.NotEmpty(t, srv.Hosts)
	for _, h := range srv.Hosts {
		require.NotEmpty(t, h.URIs)
		for _, u := range h.URIs {
			require.Contains(t, bbSvcexprValidSchemes, u.Scheme(), "host URI %q must carry a valid scheme", string(u))
		}
	}
}

// Detail 10 (Inferable: no): finalize appends a URI for each scheme
// family a listed service needs but the host lacks — assert the scheme
// family appears, not the literal URI.
func TestDetail10(t *testing.T) {
	expr.RunDSL(t, func() {
		API("bbapi", func() {
			Server("bbsrv", func() {
				Host("bbh", func() { URI("grpc://bb-only-grpc:1") })
			})
		})
		Service("bbhttp", func() {
			Method("bbm", func() { HTTP(func() { GET("/") }) })
		})
	})
	var srv *expr.ServerExpr
	for _, s := range expr.Root.API.Servers {
		if s.Name == "bbsrv" {
			srv = s
		}
	}
	require.NotNil(t, srv)
	var host *expr.HostExpr
	for _, h := range srv.Hosts {
		if h.Name == "bbh" {
			host = h
		}
	}
	require.NotNil(t, host)
	var schemes []string
	for _, u := range host.URIs {
		schemes = append(schemes, u.Scheme())
	}
	require.Contains(t, schemes, "grpc", "original grpc URI must remain")
	require.True(t, host.HasHTTPScheme(), "host serving an HTTP service must gain an http-scheme URI, got %v", schemes)
}

// Detail 11 (Inferable: partially): server validation merges host errors
// and reports each listed-but-absent service by name.
func TestDetail11(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		API("bbapi", func() {
			Server("bbsrv", func() {
				Services("bbsvc-missing")
				Host("bbh", func() { URI("://bb-bad-uri") })
			})
		})
	})
	require.Error(t, err)
	msg := err.Error()
	require.Contains(t, msg, "bbsvc-missing", "must report the absent service by name")
	require.Contains(t, msg, "://bb-bad-uri", "must surface the host's URI error")
}

// Detail 12 (Inferable: partially): eval names identify server and host
// by name; host names its owning server.
func TestDetail12(t *testing.T) {
	s := &expr.ServerExpr{Name: "bbsrv"}
	require.NotEmpty(t, s.EvalName())
	require.Contains(t, s.EvalName(), "bbsrv")
	require.NotEqual(t, s.EvalName(), (&expr.ServerExpr{Name: "bbx"}).EvalName())
	require.NotEmpty(t, (&expr.ServerExpr{}).EvalName())

	h := &expr.HostExpr{Name: "bbh", ServerName: "bbsrv"}
	require.Contains(t, h.EvalName(), "bbh")
	require.Contains(t, h.EvalName(), "bbsrv", "host eval name must name its server")
}
