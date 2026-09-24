// Hidden black-box tests for unit "httperrexpr".
//
// TestDetailNN numbers match DETAILS.md lines 1..10.
// Inferable:no lines assert shape only (presence/structure/relations),
// never the committed literal.

package expr_test

import (
	"testing"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"

	"github.com/stretchr/testify/require"
)

// bbEndpointErr returns the HTTP error mapping named name on the endpoint.
func bbEndpointErr(svc, m, name string) *expr.HTTPErrorExpr {
	for _, s := range expr.Root.API.HTTP.Services {
		if s.ServiceExpr.Name != svc {
			continue
		}
		for _, e := range s.HTTPEndpoints {
			if e.MethodExpr.Name != m {
				continue
			}
			for _, he := range e.HTTPErrors {
				if he.Name == name {
					return he
				}
			}
		}
	}
	return nil
}

// Detail 1 (Inferable: partially): EvalName is "HTTP error " plus the
// declared error name.
func TestDetail01(t *testing.T) {
	n := (&expr.HTTPErrorExpr{Name: "bbnamed"}).EvalName()
	require.Contains(t, n, "bbnamed")
	require.Contains(t, n, "HTTP")
}

// Detail 2 (Inferable: partially): JSON-RPC is decided by the response
// parent — a JSON-RPC API parent is true; service/endpoint parents
// delegate to that parent's own predicate; anything else is false.
func TestDetail02(t *testing.T) {
	jrpc := &expr.HTTPErrorExpr{Response: &expr.HTTPResponseExpr{Parent: &expr.JSONRPCExpr{}}}
	require.True(t, jrpc.IsJSONRPC(), "JSON-RPC API parent -> true")

	svc := &expr.HTTPErrorExpr{Response: &expr.HTTPResponseExpr{
		Parent: &expr.HTTPServiceExpr{ServiceExpr: &expr.ServiceExpr{
			Meta: expr.MetaExpr{"jsonrpc:service": {}},
		}},
	}}
	require.True(t, svc.IsJSONRPC(), "JSON-RPC service parent delegates -> true")

	ep := &expr.HTTPErrorExpr{Response: &expr.HTTPResponseExpr{
		Parent: &expr.HTTPEndpointExpr{Meta: expr.MetaExpr{"jsonrpc": {}}},
	}}
	require.True(t, ep.IsJSONRPC(), "JSON-RPC endpoint parent delegates -> true")

	plain := &expr.HTTPErrorExpr{Response: &expr.HTTPResponseExpr{
		Parent: &expr.HTTPServiceExpr{ServiceExpr: &expr.ServiceExpr{}},
	}}
	require.False(t, plain.IsJSONRPC(), "plain HTTP service parent -> false")

	none := &expr.HTTPErrorExpr{Response: &expr.HTTPResponseExpr{}}
	require.False(t, none.IsJSONRPC(), "no parent -> false")
}

// Detail 3 (Inferable: doc): on a JSON-RPC parent, reserved status codes
// and any mapped headers or cookies are each rejected with their own
// errors.
func TestDetail03(t *testing.T) {
	// reserved code on a JSON-RPC error mapping
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() { Attribute("n", String) })
				JSONRPC(func() { Response("bbe", -32500) })
			})
		})
	})
	require.Error(t, err, "reserved code rejected")

	// headers on a JSON-RPC error mapping
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() { Attribute("n", String) })
				JSONRPC(func() {
					Response("bbe", func() {
						Code(7001)
						Header("bbh")
					})
				})
			})
		})
	})
	require.Error(t, err, "mapped headers on JSON-RPC error rejected")

	// cookies on a JSON-RPC error mapping
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() { Attribute("n", String) })
				JSONRPC(func() {
					Response("bbe", func() {
						Code(7001)
						Cookie("bbc")
					})
				})
			})
		})
	})
	require.Error(t, err, "mapped cookies on JSON-RPC error rejected")

	// ordinary HTTP parent: headers are fine
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() { Attribute("n", String) })
				HTTP(func() {
					POST("/x")
					Response("bbe", StatusBadRequest, func() { Header("n") })
				})
			})
		})
	})
}

// Detail 4 (Inferable: doc): a code is unusable only inside
// -32768..-32100 exclusive of the five standard protocol codes;
// -32099..-32000 and everything else is allowed.
func TestDetail04(t *testing.T) {
	reserved := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() { Attribute("n", String) })
				JSONRPC(func() { Response("bbe", -32500) })
			})
		})
	})
	require.Error(t, reserved)

	// the five standard protocol codes are allowed
	for _, code := range []int{-32700, -32600, -32601, -32602, -32603} {
		expr.RunDSL(t, func() {
			Service("bbs", func() {
				Method("m", func() {
					Error("bbe", func() { Attribute("n", String) })
					JSONRPC(func() { Response("bbe", code) })
				})
			})
		})
	}

	// -32099..-32000 implementation-defined block is allowed
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() { Attribute("n", String) })
				JSONRPC(func() { Response("bbe", -32099) })
			})
		})
	})

	// positive application codes are allowed
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() { Attribute("n", String) })
				JSONRPC(func() { Response("bbe", 7001) })
			})
		})
	})
}

// Detail 5 (Inferable: partially): the "does not match an error defined
// in the ..." check names the scope — method, service, or API — matching
// the response parent kind.
func TestDetail05(t *testing.T) {
	// method scope
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				HTTP(func() {
					GET("/x")
					Response("zznone", StatusNotFound)
				})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zznone")
	require.Contains(t, err.Error(), "method", "method scope named: %s", err.Error())

	// service scope
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			HTTP(func() { Response("zznone", StatusNotFound) })
			Method("m", func() { HTTP(func() { GET("/x") }) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zznone")
	require.Contains(t, err.Error(), "service", "service scope named: %s", err.Error())

	// API scope
	err = expr.RunInvalidDSL(t, func() {
		API("bbapi", func() {
			HTTP(func() { Response("zznone", StatusNotFound) })
		})
		Service("bbs", func() {
			Method("m", func() { HTTP(func() { GET("/x") }) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zznone")
	require.Contains(t, err.Error(), "API", "API scope named: %s", err.Error())
}

// Detail 6 (Inferable: partially): an empty error type with any headers
// errors; an object error type requires each header name to resolve and
// each mapped attribute to be a primitive or array of primitives.
func TestDetail06(t *testing.T) {
	// empty error type + headers
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", Empty)
				HTTP(func() {
					GET("/x")
					Response("bbe", StatusBadRequest, func() { Header("bbh") })
				})
			})
		})
	})
	require.Error(t, err, "headers on an empty error type rejected")

	// object error + unresolvable header name
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() { Attribute("a", String) })
				HTTP(func() {
					GET("/x")
					Response("bbe", StatusBadRequest, func() { Header("zzmissing") })
				})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zzmissing", "unresolved header named")

	// object error + non-primitive mapped attribute
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() {
					Attribute("bbobj", func() { Attribute("i", String) })
				})
				HTTP(func() {
					GET("/x")
					Response("bbe", StatusBadRequest, func() { Header("bbobj") })
				})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbobj", "non-primitive header field named")

	// array-of-primitives mapped attribute is fine
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() { Attribute("a", ArrayOf(String)) })
				HTTP(func() {
					GET("/x")
					Response("bbe", StatusBadRequest, func() { Header("a") })
				})
			})
		})
	})
}

// Detail 7 (Inferable: no): for a non-object error type, headers are
// allowed only when at most one is mapped; an array error type needs
// primitive elements; a map error type is always rejected.
func TestDetail07(t *testing.T) {
	// scalar error, two headers -> error
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", String)
				HTTP(func() {
					GET("/x")
					Response("bbe", StatusBadRequest, func() {
						Header("h1")
						Header("h2")
					})
				})
			})
		})
	})
	require.Error(t, err, "more than one header on a non-object error rejected")

	// scalar error, one header -> clean
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", String)
				HTTP(func() {
					GET("/x")
					Response("bbe", StatusBadRequest, func() { Header("h1") })
				})
			})
		})
	})

	// map error type + header -> always rejected
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", MapOf(String, String))
				HTTP(func() {
					GET("/x")
					Response("bbe", StatusBadRequest, func() { Header("h1") })
				})
			})
		})
	})
	require.Error(t, err, "map error type with headers rejected")
}

// Detail 8 (Inferable: partially): Finalize resolves the error expression
// from the method, finalizes the response, defaults a missing body via
// the shared error body computation, and maps unmapped error attributes
// to headers.
func TestDetail08(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", func() {
					Attribute("bbar", String)
					Attribute("bbbaz", Int)
				})
				HTTP(func() {
					GET("/x")
					Response("bbe", StatusBadRequest, func() { Header("bbar") })
				})
			})
		})
	})
	he := bbEndpointErr("bbs", "m", "bbe")
	require.NotNil(t, he)
	require.NotNil(t, he.Response)
	// explicitly mapped attr lands in headers under its own name
	require.NotNil(t, he.Response.Headers)
	hObj := expr.AsObject(he.Response.Headers.Type)
	require.NotNil(t, hObj)
	hNames := map[string]bool{}
	for _, f := range *hObj {
		hNames[f.Name] = true
	}
	require.True(t, hNames["bbar"], "mapped attr present in headers")
	// body defaulted via the shared error body computation: holds the
	// attributes not mapped to headers
	require.NotNil(t, he.Response.Body)
	bObj := expr.AsObject(he.Response.Body.Type)
	require.NotNil(t, bObj, "error body computed")
	bNames := map[string]bool{}
	for _, f := range *bObj {
		bNames[f.Name] = true
	}
	require.True(t, bNames["bbbaz"], "unmapped attr retained in default body")
	require.False(t, bNames["bbar"], "header-mapped attr removed from body")
}

// Detail 9 (Inferable: no): Finalize sets the response content type to
// the result type's IDENTIFIER — not its content type — only when the
// body is non-empty, the content type is unset, and the body type is a
// result type.
func TestDetail09(t *testing.T) {
	expr.RunDSL(t, func() {
		RT := ResultType("application/vnd.bb.err", func() {
			TypeName("BbErrRT")
			Attributes(func() {
				Attribute("n", String)
			})
		})
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe", RT)
				HTTP(func() {
					GET("/x")
					Response("bbe", StatusBadRequest)
				})
			})
		})
	})
	he := bbEndpointErr("bbs", "m", "bbe")
	require.NotNil(t, he)
	require.NotNil(t, he.Response)
	require.NotNil(t, he.Response.Body)
	require.NotEmpty(t, expr.AsObject(he.Response.Body.Type),
		"error body computed")
	require.Equal(t, "application/vnd.bb.err", he.Response.ContentType,
		"content type falls back to the result type identifier")
}

// Detail 10 (Inferable: partially): Dup shares the resolved error
// expression pointer and name but deep-copies the response.
func TestDetail10(t *testing.T) {
	errExpr := &expr.ErrorExpr{
		Name:          "bbe",
		AttributeExpr: &expr.AttributeExpr{Type: expr.String},
	}
	src := &expr.HTTPErrorExpr{
		ErrorExpr: errExpr,
		Name:      "bbe",
		Response: &expr.HTTPResponseExpr{
			StatusCode: 500,
			Headers:    expr.NewEmptyMappedAttributeExpr(),
		},
	}
	d := src.Dup()
	require.Same(t, src.ErrorExpr, d.ErrorExpr, "error expression shared")
	require.Equal(t, src.Name, d.Name, "name preserved")
	require.NotSame(t, src.Response, d.Response, "response deep-copied")
	require.Equal(t, src.Response.StatusCode, d.Response.StatusCode)
	if src.Response.Headers != nil && d.Response.Headers != nil {
		require.NotSame(t, src.Response.Headers, d.Response.Headers,
			"headers deep-copied")
	}
}
