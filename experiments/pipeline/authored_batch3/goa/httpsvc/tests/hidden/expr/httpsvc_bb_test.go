// Hidden black-box tests for unit "httpsvc".
//
// TestDetailNN numbers match DETAILS.md lines 1..12.
// Inferable:no lines assert shape only (presence/structure/relations),
// never the committed literal.

package expr_test

import (
	"strings"
	"testing"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"

	"github.com/stretchr/testify/require"
)

// bbHTTPSvc returns the plain HTTP service expression for name.
func bbHTTPSvc(name string) *expr.HTTPServiceExpr {
	for _, s := range expr.Root.API.HTTP.Services {
		if s.ServiceExpr.Name == name {
			return s
		}
	}
	return nil
}

// Detail 1 (Inferable: doc): the canonical endpoint is the named one,
// defaulting to "show" when unset.
func TestDetail01(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbcanon", func() {
			HTTP(func() {})
			Method("show", func() { HTTP(func() { GET("/bbs") }) })
			Method("other", func() { HTTP(func() { GET("/bbo") }) })
		})
		Service("bbcanon2", func() {
			HTTP(func() { CanonicalMethod("other") })
			Method("show", func() { HTTP(func() { GET("/bbs2") }) })
			Method("other", func() { HTTP(func() { GET("/bbo2") }) })
		})
	})
	c := bbHTTPSvc("bbcanon").CanonicalEndpoint()
	require.NotNil(t, c)
	require.Equal(t, "show", c.MethodExpr.Name, "default canonical endpoint")

	c2 := bbHTTPSvc("bbcanon2").CanonicalEndpoint()
	require.NotNil(t, c2)
	require.Equal(t, "other", c2.MethodExpr.Name, "named canonical endpoint")
}

// Detail 2 (Inferable: no): FullPaths — no declared paths yields the root
// path; "//" paths are cleaned on their own; otherwise each path joins the
// parent canonical endpoint's first route full path (or the root path);
// trailing slash preserved. Assert structural relations.
func TestDetail02(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbparent", func() {
			HTTP(func() {})
			Method("show", func() { HTTP(func() { GET("/bbp") }) })
		})
		Service("bbchild", func() {
			HTTP(func() {
				Parent("bbparent")
				Path("/bbc")
			})
			Method("cm", func() { HTTP(func() { GET("/x") }) })
		})
		Service("bbchildslash", func() {
			HTTP(func() {
				Parent("bbparent")
				Path("/bbs/")
			})
			Method("cm", func() { HTTP(func() { GET("/x") }) })
		})
		Service("bbchildabs", func() {
			HTTP(func() {
				Parent("bbparent")
				Path("//bbabs")
			})
			Method("cm", func() { HTTP(func() { GET("/x") }) })
		})
		Service("bbrootless", func() {
			HTTP(func() {})
			Method("m", func() { HTTP(func() { GET("/x") }) })
		})
	})

	parentSvc := bbHTTPSvc("bbparent")
	require.NotNil(t, parentSvc)
	require.NotEmpty(t, parentSvc.FullPaths(), "no declared paths yields a path")
	// the join base is the parent canonical endpoint's first route
	routePath := parentSvc.CanonicalEndpoint().Routes[0].Path

	child := bbHTTPSvc("bbchild").FullPaths()
	require.NotEmpty(t, child)
	require.True(t, strings.HasPrefix(child[0], strings.TrimSuffix(routePath, "/")+"/"),
		"child full path %q joins parent route %q", child[0], routePath)
	require.Contains(t, child[0], "bbc", "child segment present")

	abs := bbHTTPSvc("bbchildabs").FullPaths()
	require.NotEmpty(t, abs)
	require.Contains(t, abs[0], "bbabs", "absolute path segment present")
	require.False(t, strings.Contains(abs[0], "bbp"),
		"// path is cleaned on its own, not joined to parent %q", abs[0])

	slash := bbHTTPSvc("bbchildslash").FullPaths()
	require.NotEmpty(t, slash)
	require.True(t, strings.HasSuffix(slash[0], "/"),
		"trailing slash preserved in %q", slash[0])
}

// Detail 3 (Inferable: yes): Parent resolves through the root's service
// list by ParentName.
func TestDetail03(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbparent", func() {
			HTTP(func() {})
			Method("show", func() { HTTP(func() { GET("/bbp") }) })
		})
		Service("bbchild", func() {
			HTTP(func() { Parent("bbparent") })
			Method("cm", func() { HTTP(func() { GET("/x") }) })
		})
		Service("bborphan", func() {
			HTTP(func() {})
			Method("om", func() { HTTP(func() { GET("/x") }) })
		})
	})
	p := bbHTTPSvc("bbchild").Parent()
	require.NotNil(t, p)
	require.Equal(t, "bbparent", p.ServiceExpr.Name)
	require.Nil(t, bbHTTPSvc("bborphan").Parent())
}

// Detail 4 (Inferable: no): JSON-RPC is decided by the service's
// "jsonrpc:service" meta — not by an explicit route. Assert: the JSONRPC()
// service reports true; a plain HTTP service (even with POST routes)
// reports false.
func TestDetail04(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbj", func() {
			JSONRPC(func() { POST("/bbrpc") })
			Method("jm", func() { JSONRPC(func() {}) })
		})
		Service("bbh", func() {
			Method("m", func() { HTTP(func() { POST("/bbh") }) })
		})
	})
	var jsvc *expr.HTTPServiceExpr
	for _, s := range expr.Root.API.JSONRPC.Services {
		if s.ServiceExpr.Name == "bbj" {
			jsvc = s
		}
	}
	require.NotNil(t, jsvc, "JSON-RPC service registered")
	require.True(t, jsvc.IsJSONRPC())
	require.False(t, bbHTTPSvc("bbh").IsJSONRPC(),
		"plain HTTP service with POST route is not JSON-RPC")
}

// Detail 5 (Inferable: no): Prepare pulls any API-level HTTP error matching
// a declared service error that the service does not already map —
// duplicated, not shared.
func TestDetail05(t *testing.T) {
	expr.RunDSL(t, func() {
		API("bbapi", func() {
			Error("bbapierr", func() { Attribute("m", String) })
			HTTP(func() { Response("bbapierr", StatusBadRequest) })
		})
		Service("bbpull", func() {
			Error("bbapierr")
			Method("m", func() { HTTP(func() { GET("/x") }) })
		})
		Service("bbown", func() {
			Error("bbapierr")
			HTTP(func() { Response("bbapierr", StatusTeapot) })
			Method("m", func() { HTTP(func() { GET("/x") }) })
		})
	})

	var apiErr *expr.HTTPErrorExpr
	for _, e := range expr.Root.API.HTTP.Errors {
		if e.Name == "bbapierr" {
			apiErr = e
		}
	}
	require.NotNil(t, apiErr, "API-level HTTP error exists")

	var pulled *expr.HTTPErrorExpr
	for _, e := range bbHTTPSvc("bbpull").HTTPErrors {
		if e.Name == "bbapierr" {
			pulled = e
		}
	}
	require.NotNil(t, pulled, "service pulls the matching API error")
	require.NotSame(t, apiErr, pulled, "pulled error is duplicated, not shared")

	count := 0
	for _, e := range bbHTTPSvc("bbown").HTTPErrors {
		if e.Name == "bbapierr" {
			count++
		}
	}
	require.Equal(t, 1, count, "service that maps the error itself is not re-pulled")
}

// Detail 6 (Inferable: no): JSON-RPC route defaults — no explicit route
// means every JSON-RPC endpoint shares POST on the service's first path
// ("/" when none).
func TestDetail06(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbjpath", func() {
			JSONRPC(func() { Path("/bbjbase") })
			Method("jm", func() { JSONRPC(func() {}) })
		})
		Service("bbjnopath", func() {
			JSONRPC(func() {})
			Method("jm", func() { JSONRPC(func() {}) })
		})
	})
	jpath := func(name string) *expr.HTTPServiceExpr {
		for _, s := range expr.Root.API.JSONRPC.Services {
			if s.ServiceExpr.Name == name {
				return s
			}
		}
		return nil
	}
	svc := jpath("bbjpath")
	require.NotNil(t, svc)
	require.NotEmpty(t, svc.HTTPEndpoints)
	for _, e := range svc.HTTPEndpoints {
		require.Len(t, e.Routes, 1)
		require.Equal(t, "POST", e.Routes[0].Method)
		require.Equal(t, "/bbjbase", e.Routes[0].Path)
	}

	svc2 := jpath("bbjnopath")
	require.NotNil(t, svc2)
	for _, e := range svc2.HTTPEndpoints {
		require.Len(t, e.Routes, 1)
		require.Equal(t, "POST", e.Routes[0].Method)
		require.Equal(t, "/", e.Routes[0].Path)
	}
}

// Detail 7 (Inferable: partially): missing parent errors; a parent without
// a canonical endpoint errors; a parent that is also a child errors.
func TestDetail07(t *testing.T) {
	// missing parent
	err := expr.RunInvalidDSL(t, func() {
		Service("bbchild", func() {
			HTTP(func() { Parent("bbghost") })
			Method("cm", func() { HTTP(func() { GET("/x") }) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbghost", "missing parent is named")

	// parent without canonical endpoint
	err = expr.RunInvalidDSL(t, func() {
		Service("bbnocanon", func() {
			HTTP(func() {})
			Method("notshow", func() { HTTP(func() { GET("/x") }) })
		})
		Service("bbchild", func() {
			HTTP(func() { Parent("bbnocanon") })
			Method("cm", func() { HTTP(func() { GET("/x") }) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbnocanon", "parent is named")

	// a parent that is also a child — the service named as parent must not
	// itself declare a parent pointing back
	err = expr.RunInvalidDSL(t, func() {
		Service("bbselfp", func() {
			HTTP(func() { Parent("bbselfp") })
			Method("show", func() { HTTP(func() { GET("/s") }) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbselfp", "self-parenting service is named")
}

// Detail 8 (Inferable: partially): validation chain order — attributes
// (params then headers), parent, canonical endpoint, errors, transports.
// Assert the observable relative order of a service violating params,
// headers and parent rules at once.
func TestDetail08(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("bborder", func() {
			HTTP(func() {
				Parent("bbghost")
				Params(func() {
					Param("bbbadparam", String, func() {
						Enum("a", "b")
						Default("zznotin")
					})
				})
				Headers(func() {
					Header("bbbadheader", String, func() {
						Enum("a", "b")
						Default("zznotin")
					})
				})
			})
			Method("m", func() {
				Payload(func() {
					Attribute("bbbadparam", String)
					Attribute("bbbadheader", String)
				})
				HTTP(func() { GET("/x") })
			})
		})
	})
	require.Error(t, err)
	msg := err.Error()
	ip := strings.Index(msg, "bbbadparam")
	ih := strings.Index(msg, "bbbadheader")
	ipa := strings.Index(msg, "bbghost")
	require.GreaterOrEqual(t, ip, 0, "param violation reported: %s", msg)
	require.GreaterOrEqual(t, ih, 0, "header violation reported: %s", msg)
	require.GreaterOrEqual(t, ipa, 0, "parent violation reported: %s", msg)
	require.Less(t, ip, ih, "params validated before headers")
	require.Less(t, ih, ipa, "attributes validated before parent")
}

// Detail 9 (Inferable: doc): error validation re-checks root HTTP errors —
// the same error may be validated repeatedly across services. Assert one
// bad root HTTP error produces a diagnostic per service.
func TestDetail09(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		API("bbapi", func() {
			HTTP(func() { Response("bbbadroot", func() {}) })
		})
		Service("bbs1", func() { Method("m", func() { HTTP(func() { GET("/a") }) }) })
		Service("bbs2", func() { Method("m", func() { HTTP(func() { GET("/b") }) }) })
	})
	require.Error(t, err)
	require.GreaterOrEqual(t, strings.Count(err.Error(), "bbbadroot"), 2,
		"root HTTP error re-validated per service: %s", err.Error())
}

// Detail 10 (Inferable: partially): JSON-RPC route validation requires POST
// on every route of every JSON-RPC endpoint. Assert a non-POST route errors.
func TestDetail10(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("bbjget", func() {
			JSONRPC(func() { GET("/bbget") })
			Method("jm", func() { JSONRPC(func() {}) })
		})
	})
	require.Error(t, err, "non-POST JSON-RPC route must error")
	require.Contains(t, err.Error(), "bbjget", "offending service named")
}

// Detail 11 (Inferable: yes): Finalize defaults an empty path list to ["/"].
func TestDetail11(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbnopath", func() {
			HTTP(func() {})
			Method("m", func() { HTTP(func() { GET("/x") }) })
		})
	})
	require.Equal(t, []string{"/"}, bbHTTPSvc("bbnopath").Paths)
}

// Detail 12 (Inferable: partially): EvalName is "unnamed service" or
// `service "<name>"`. Assert the name is carried for named services and an
// unnamed placeholder for anonymous ones.
func TestDetail12(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbnamedsvc", func() {
			HTTP(func() {})
			Method("m", func() { HTTP(func() { GET("/x") }) })
		})
	})
	require.Contains(t, bbHTTPSvc("bbnamedsvc").EvalName(), "bbnamedsvc")

	anon := (&expr.HTTPServiceExpr{ServiceExpr: &expr.ServiceExpr{}}).EvalName()
	require.Contains(t, strings.ToLower(anon), "unnamed")
}
