// Hidden black-box tests for unit "rootval".
//
// TestDetailNN numbers match DETAILS.md lines 1..14.
// Inferable:no lines assert shape only (presence/structure/relations),
// never the committed literal.

package expr_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/eval"
	"example.internal/apikit/v3/expr"

	"github.com/stretchr/testify/require"
)

// bbRootVerrs unwraps a *eval.ValidationErrors from err. Validate() returns a
// non-nil *ValidationErrors even when empty, so callers must inspect
// .Errors/.Expressions rather than relying on err != nil.
func bbRootVerrs(t *testing.T, err error) *eval.ValidationErrors {
	t.Helper()
	if err == nil {
		return nil
	}
	var ve *eval.ValidationErrors
	require.ErrorAs(t, err, &ve)
	return ve
}

// bbRootKind classifies a walked expression for the walk-order assertions.
func bbRootKind(e eval.Expression) string {
	switch v := e.(type) {
	case *expr.APIExpr:
		return "api"
	case *expr.ServerExpr:
		return "server"
	case *expr.UserTypeExpr, *expr.AttributeExpr:
		// user types walk as their underlying attribute expression
		return "usertype"
	case *expr.ResultTypeExpr:
		return "resulttype"
	case *expr.ServiceExpr:
		return "service"
	case *expr.MethodExpr:
		return "method"
	case *expr.HTTPExpr:
		return "httproot"
	case *expr.JSONRPCExpr:
		return "jsonrpcroot"
	case *expr.HTTPServiceExpr:
		if v.IsJSONRPC() {
			return "jsonrpcsvc"
		}
		return "httpsvc"
	case *expr.HTTPEndpointExpr:
		if v.IsJSONRPC() {
			return "jsonrpcend"
		}
		return "httpend"
	case *expr.HTTPFileServerExpr:
		return "fileserver"
	case *expr.GRPCExpr:
		return "grpcroot"
	case *expr.GRPCServiceExpr:
		return "grpcsvc"
	case *expr.GRPCEndpointExpr:
		return "grpcend"
	}
	return fmt.Sprintf("%T", e)
}

func bbKindIndex(sets []string, kind string) int {
	for i, k := range sets {
		if k == kind {
			return i
		}
	}
	return -1
}

// Detail 1 (Inferable: partially): walk order — API, servers, user types,
// result types, services, methods, HTTP services/endpoints/file servers,
// JSON-RPC, gRPC root/services/endpoints. The committed relative order is
// observable, so assert the relative order of the named groups.
func TestDetail01(t *testing.T) {
	expr.RunDSL(t, func() {
		API("bbwalk", func() {
			Server("bbhost", func() {
				Host("localhost", func() { URI("http://localhost") })
			})
		})
		Type("BbWalkT", func() { Attribute("f", String) })
		ResultType("application/vnd.bb.walk", func() {
			TypeName("BbWalkRT")
			Attributes(func() { Attribute("a", String) })
		})
		Service("bbhttpsvc", func() {
			Method("bm", func() {
				Payload(func() { Attribute("p", String) })
				HTTP(func() { GET("/bbm") })
			})
			Files("/bbstatic/file.txt", "/tmp/bbfile.txt")
		})
		Service("bbjrpcsvc", func() {
			JSONRPC(func() { POST("/bbjrpc") })
			Method("bjm", func() {})
		})
		Service("bbgrpcsvc", func() {
			GRPC(func() {})
			Method("bgm", func() { GRPC(func() {}) })
		})
	})

	var sets []string
	expr.Root.WalkSets(func(s eval.ExpressionSet) {
		for _, e := range s {
			sets = append(sets, bbRootKind(e))
		}
	})
	require.NotEmpty(t, sets, "WalkSets must yield sets for a populated design")

	before := func(a, b string) {
		ia, ib := bbKindIndex(sets, a), bbKindIndex(sets, b)
		require.GreaterOrEqual(t, ia, 0, "walk must visit %s (saw %v)", a, sets)
		require.GreaterOrEqual(t, ib, 0, "walk must visit %s (saw %v)", b, sets)
		require.Less(t, ia, ib, "walk must visit %s before %s (saw %v)", a, b, sets)
	}
	before("api", "server")
	before("server", "usertype")
	before("usertype", "resulttype")
	before("resulttype", "service")
	before("service", "method")
	before("method", "httpsvc")
	before("httpsvc", "httpend")
	before("httpend", "fileserver")
	before("fileserver", "jsonrpcsvc")
	before("jsonrpcsvc", "grpcroot")
	before("grpcroot", "grpcsvc")
	before("grpcsvc", "grpcend")
}

// Detail 2 (Inferable: partially): a nil API is created from the first
// service name, else "API". The creation happens as part of walking.
func TestDetail02(t *testing.T) {
	r := &expr.RootExpr{
		Services: []*expr.ServiceExpr{{Name: "bbonlysvc"}},
	}
	r.WalkSets(func(eval.ExpressionSet) {})
	require.NotNil(t, r.API, "a nil API must be created")
	require.Equal(t, "bbonlysvc", r.API.Name,
		"API created from the first service name")

	// no services at all: the committed default is "API"
	r2 := &expr.RootExpr{}
	r2.WalkSets(func(eval.ExpressionSet) {})
	require.NotNil(t, r2.API, "a nil API must be created")
	require.Equal(t, "API", r2.API.Name)
}

// Detail 3 (Inferable: no): within HTTP and gRPC service lists a child
// service sorts AFTER the parent named by its ParentName (stable).
// Assert relative order only — child lands after parent regardless of
// declaration order.
func TestDetail03(t *testing.T) {
	expr.RunDSL(t, func() {
		// child declared first on purpose
		Service("bbchild", func() {
			HTTP(func() { Parent("bbparent") })
			Method("cm", func() { HTTP(func() { GET("/bbc") }) })
		})
		Service("bbparent", func() {
			HTTP(func() {})
			Method("show", func() { HTTP(func() { GET("/bbp") }) })
		})
	})
	var names []string
	for _, s := range expr.Root.API.HTTP.Services {
		names = append(names, s.ServiceExpr.Name)
	}
	pi, ci := -1, -1
	for i, n := range names {
		if n == "bbparent" {
			pi = i
		}
		if n == "bbchild" {
			ci = i
		}
	}
	require.GreaterOrEqual(t, pi, 0, "parent service present in HTTP list")
	require.GreaterOrEqual(t, ci, 0, "child service present in HTTP list")
	require.Greater(t, ci, pi,
		"child must sort after parent (saw %v)", names)
}

// Detail 4 (Inferable: yes): UserType looks in declared types first, then
// result types.
func TestDetail04(t *testing.T) {
	expr.RunDSL(t, func() {
		Type("BbPlainT", func() { Attribute("k", String) })
		ResultType("application/vnd.bb.onlyrt", func() {
			TypeName("BbOnlyRT")
			Attributes(func() { Attribute("a", String) })
		})
	})
	ut := expr.Root.UserType("BbPlainT")
	require.NotNil(t, ut, "declared type found")
	require.Equal(t, "BbPlainT", ut.Name())
	_, isRT := ut.(*expr.ResultTypeExpr)
	require.False(t, isRT, "declared type wins over result types")

	rt := expr.Root.UserType("BbOnlyRT")
	require.NotNil(t, rt, "result type found via second lookup")
	_, isRT = rt.(*expr.ResultTypeExpr)
	require.True(t, isRT, "result type reachable when no declared type matches")

	require.Nil(t, expr.Root.UserType("BbMissing"), "missing name returns nil")
}

// Detail 5 (Inferable: no): duplicate checks — user types by resolved
// Name(); declared result types too, generated ones skipped. Assert shape:
// duplicate declared names produce an error naming the type.
func TestDetail05(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("BbDupT", func() { Attribute("a", String) })
		Type("BbDupT", func() { Attribute("b", String) })
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "BbDupT",
		"duplicate diagnostic names the offending type")

	// duplicate declared result type names error too
	err = expr.RunInvalidDSL(t, func() {
		ResultType("application/vnd.bb.one", func() {
			TypeName("BbDupRT")
			Attributes(func() { Attribute("a", String) })
		})
		ResultType("application/vnd.bb.two", func() {
			TypeName("BbDupRT")
			Attributes(func() { Attribute("b", String) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "BbDupRT",
		"duplicate diagnostic names the offending result type")
}

// Detail 6 (Inferable: partially): every user type and result type gets
// default-value validation scoped "type %q" / "result type %q". Assert the
// scope names the offending expression.
func TestDetail06(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("BbScopedT", func() {
			Attribute("n", Int, func() { Default("zzbad") })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "BbScopedT",
		"user-type default error names the type")

	err = expr.RunInvalidDSL(t, func() {
		ResultType("application/vnd.bb.scoped", func() {
			TypeName("BbScopedRT")
			Attributes(func() {
				Attribute("n", Int, func() { Default("zzbad") })
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "BbScopedRT",
		"result-type default error names the result type")
}

// Detail 7 (Inferable: no): each declared error is default-validated
// exactly once across root/service/method — deduped by pointer identity.
// Assert the same bad default inside one shared error reports once.
func TestDetail07(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		API("bbd", func() {
			Error("bbsharederr", func() {
				Attribute("bberrfield", Int, func() { Default("zzbad") })
			})
		})
		Service("bbs", func() {
			Error("bbsharederr")
			Method("m", func() {
				Error("bbsharederr")
			})
		})
	})
	require.Error(t, err)
	require.Equal(t, 1, strings.Count(err.Error(), "bberrfield"),
		"shared error default validated exactly once, got: %s", err.Error())
}

// Detail 8 (Inferable: no): one authored type under two or more distinct
// static error names is rejected unless a nested attribute carries the
// error-name marker meta. Assert: rejection fires for the unmarked type and
// names it; a type with a required ErrorName field is accepted.
func TestDetail08(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		ET := Type("BbErrT", func() { Attribute("n", String) })
		Service("bbs", func() {
			Method("a", func() { Error("eone", ET) })
			Method("b", func() { Error("etwo", ET) })
		})
	})
	require.Error(t, err, "shared error names must be rejected")
	require.Contains(t, err.Error(), "BbErrT",
		"diagnostic names the shared type")

	// a type whose ErrorName attribute is required may be shared
	expr.RunDSL(t, func() {
		ET2 := Type("BbErrT2", func() {
			ErrorName("bbename", String)
			Required("bbename")
		})
		Service("bbs2", func() {
			Method("a", func() { Error("emarked1", ET2) })
			Method("b", func() { Error("emarked2", ET2) })
		})
	})
}

// Detail 9 (Inferable: no): shared-route detection runs only when both HTTP
// and JSON-RPC transports exist; parameter names are normalized
// ("/{*x}" -> "/{*wildcard}", others -> "/{parameter}"); the error lands on
// the JSON-RPC route. Assert: collision errors, flagged expression is the
// JSON-RPC endpoint, and parameter-name differences still collide.
func TestDetail09(t *testing.T) {
	// colliding paths with different parameter names
	err := expr.RunInvalidDSL(t, func() {
		Service("bbhttpsvc", func() {
			Method("m", func() {
				Payload(func() { Attribute("x", String) })
				HTTP(func() { POST("/bb/{x}") })
			})
		})
		Service("bbjrpcsvc", func() {
			JSONRPC(func() { POST("/bb/{y}") })
			Method("jm", func() {
				Payload(func() { Attribute("y", String) })
				JSONRPC(func() {})
			})
		})
	})
	require.Error(t, err, "normalized shared route must error")
	// The diagnostic is reported in the context of the JSON-RPC endpoint —
	// its EvalName names the JSON-RPC service and method.
	require.Contains(t, err.Error(), `service "bbjrpcsvc" HTTP endpoint "jm"`,
		"shared-route error lands on the JSON-RPC route")

	// different methods never collide
	expr.RunDSL(t, func() {
		Service("bbhttpsvc", func() {
			Method("m", func() {
				Payload(func() { Attribute("x", String) })
				HTTP(func() { GET("/bb/{x}") })
			})
		})
		Service("bbjrpcsvc", func() {
			JSONRPC(func() { POST("/bb/{y}") })
			Method("jm", func() {
				Payload(func() { Attribute("y", String) })
				JSONRPC(func() {})
			})
		})
	})

	// different normalized paths never collide
	expr.RunDSL(t, func() {
		Service("bbhttpsvc", func() {
			Method("m", func() {
				Payload(func() { Attribute("x", String) })
				HTTP(func() { POST("/bba/{x}") })
			})
		})
		Service("bbjrpcsvc", func() {
			JSONRPC(func() { POST("/bbb/{y}") })
			Method("jm", func() {
				Payload(func() { Attribute("y", String) })
				JSONRPC(func() {})
			})
		})
	})
}

// Detail 10 (Inferable: doc): a server with an empty service list hosts
// every service — observable as Services filled with all services after
// finalize.
func TestDetail10(t *testing.T) {
	expr.RunDSL(t, func() {
		API("bbapi", func() {
			Server("bbone", func() {
				Host("dev", func() { URI("http://bbhost") })
			})
		})
		Service("bba", func() { Method("m", func() {}) })
		Service("bbb", func() { Method("m", func() {}) })
	})
	var srv *expr.ServerExpr
	for _, s := range expr.Root.API.Servers {
		if s.Name == "bbone" {
			srv = s
		}
	}
	require.NotNil(t, srv)
	require.ElementsMatch(t, []string{"bba", "bbb"}, srv.Services,
		"server with empty service list hosts every service")
}

// Detail 11 (Inferable: partially): type-map dedupe keys on (user type
// ORIGIN, reflected external type). Assert: same origin + same external
// errors, different external does not, both directions covered.
func TestDetail11(t *testing.T) {
	expr.RunDSL(t, func() {
		Type("BbMapT", func() { Attribute("f", String) })
		Service("bbs", func() { Method("m", func() {}) })
	})
	ut := expr.Root.UserType("BbMapT")
	dup := ut.Dup(&expr.AttributeExpr{Type: expr.String})
	require.Equal(t, ut.Origin(), dup.Origin(), "dup shares the origin")

	// same origin + same external type -> rejected
	expr.Root.Conversions = []*expr.TypeMap{
		{User: ut, External: time.Time{}},
		{User: dup, External: time.Time{}},
	}
	err := expr.Root.Validate()
	require.Error(t, err, "duplicate conversion map must error")
	require.Contains(t, err.Error(), "BbMapT",
		"diagnostic names the user type")

	// different external type -> allowed
	expr.Root.Conversions = []*expr.TypeMap{
		{User: ut, External: time.Time{}},
		{User: dup, External: time.Duration(0)},
	}
	ve := bbRootVerrs(t, expr.Root.Validate())
	require.Empty(t, ve.Errors, "distinct external types are not deduped")

	// creation direction is also deduped
	expr.Root.Conversions = nil
	expr.Root.Creations = []*expr.TypeMap{
		{User: ut, External: time.Time{}},
		{User: dup, External: time.Time{}},
	}
	err = expr.Root.Validate()
	require.Error(t, err, "duplicate creation map must error")
	require.Contains(t, err.Error(), "BbMapT")
}

// Detail 12 (Inferable: no): relocated types (struct:pkg:path) may depend
// only on declared types that also carry a package path. Assert shape:
// error names the offending dependency, located deps are exempt.
func TestDetail12(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("BbDep", func() { Attribute("a", String) })
		Type("BbLoc", func() {
			Attribute("bbdepfield", "BbDep")
			Meta("struct:pkg:path", "example.com/bbloc")
		})
	})
	require.Error(t, err, "relocated type depending on unlocated type errors")
	require.Contains(t, err.Error(), "bbdepfield",
		"diagnostic reports the dependency path")

	// dependency carrying its own package path is exempt
	expr.RunDSL(t, func() {
		Type("BbDep2", func() {
			Attribute("a", String)
			Meta("struct:pkg:path", "example.com/bbdep")
		})
		Type("BbLoc2", func() {
			Attribute("bbdepfield", "BbDep2")
			Meta("struct:pkg:path", "example.com/bbloc2")
		})
	})
}

// Detail 13 (Inferable: partially): Finalize defaults the API and creates a
// default server when none is declared, then finalizes root errors and
// servers.
func TestDetail13(t *testing.T) {
	r := &expr.RootExpr{API: &expr.APIExpr{}}
	r.Finalize()
	require.NotNil(t, r.API)
	require.NotEmpty(t, r.API.Servers,
		"Finalize creates a default server when none is declared")

	// a declared server is preserved, not replaced
	r2 := &expr.RootExpr{
		API: &expr.APIExpr{
			Servers: []*expr.ServerExpr{{Name: "bbdeclared"}},
		},
	}
	r2.Finalize()
	names := make([]string, 0, len(r2.API.Servers))
	for _, s := range r2.API.Servers {
		names = append(names, s.Name)
	}
	require.Contains(t, names, "bbdeclared", "declared server preserved")
}

// Detail 14 (Inferable: partially): MetaExpr.Merge appends only MISSING
// values onto existing keys and copies whole slices for absent keys; Last
// returns the last element plus an ok flag.
func TestDetail14(t *testing.T) {
	m := expr.MetaExpr{
		"k":    {"a"},
		"keep": {"1"},
	}
	m.Merge(expr.MetaExpr{
		"k":    {"a", "b"},
		"keep": {"1", "2"},
		"new":  {"x", "y"},
	})
	require.Equal(t, []string{"a", "b"}, m["k"],
		"missing values appended to existing keys")
	require.Equal(t, []string{"1", "2"}, m["keep"])
	require.Equal(t, []string{"x", "y"}, m["new"],
		"absent keys copy the whole slice")

	v, ok := expr.MetaExpr{"k": {"a", "b"}}.Last("k")
	require.True(t, ok)
	require.Equal(t, "b", v, "Last returns the last element")

	_, ok = expr.MetaExpr{}.Last("absent")
	require.False(t, ok, "missing key -> not ok")
	_, ok = expr.MetaExpr{"empty": {}}.Last("empty")
	require.False(t, ok, "empty slice -> not ok")
}
