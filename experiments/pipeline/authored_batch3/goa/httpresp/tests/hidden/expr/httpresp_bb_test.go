package expr_test

// Black-box hidden tests for HTTPResponseExpr (httpresp unit).
// Each TestDetailNN maps to the numbered commitment in _author/DETAILS.md.

import (
	"strings"
	"testing"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
	"github.com/stretchr/testify/require"
)

func bbHTTPEndpoint(svc, method string) *expr.HTTPEndpointExpr {
	for _, s := range expr.Root.API.HTTP.Services {
		if s.ServiceExpr.Name != svc {
			continue
		}
		for _, e := range s.HTTPEndpoints {
			if e.MethodExpr.Name == method {
				return e
			}
		}
	}
	return nil
}

func bbResp(t *testing.T, svc, method string) *expr.HTTPResponseExpr {
	e := bbHTTPEndpoint(svc, method)
	require.NotNil(t, e, "endpoint %s.%s", svc, method)
	require.NotEmpty(t, e.Responses, "endpoint %s.%s responses", svc, method)
	return e.Responses[0]
}

func bbFieldNames(ma *expr.MappedAttributeExpr) map[string]string {
	out := map[string]string{}
	if ma == nil {
		return out
	}
	if o := expr.AsObject(ma.Type); o != nil {
		for _, f := range *o {
			out[f.Name] = ma.ElemName(f.Name)
		}
	}
	return out
}

// Detail 1 (Inferable: partially): EvalName is "HTTP response" plus " of
// <parent name>" when a parent is set.
func TestDetail01(t *testing.T) {
	r := &expr.HTTPResponseExpr{}
	require.Contains(t, r.EvalName(), "HTTP response")

	r2 := &expr.HTTPResponseExpr{
		Parent: &expr.ServiceExpr{Name: "bbparentsvc"},
	}
	n := r2.EvalName()
	require.Contains(t, n, "HTTP response")
	require.Contains(t, n, "bbparentsvc", "parent name included")
}

// Detail 2 (Inferable: partially): Prepare installs empty mapped attributes
// for BOTH headers and cookies when nil.
func TestDetail02(t *testing.T) {
	r := &expr.HTTPResponseExpr{}
	r.Prepare()
	require.NotNil(t, r.Headers, "headers initialized")
	require.NotNil(t, r.Cookies, "cookies initialized")
	require.True(t, r.Headers.IsEmpty())
	require.True(t, r.Cookies.IsEmpty())
}

// Detail 3 (Inferable: no): a body is forbidden only for 100-199, 204 and
// 304; skipped for streaming endpoints; only when the computed body is
// non-empty. Shape asserted: which statuses error, streaming exemption.
func TestDetail03(t *testing.T) {
	dsl := func(status int) func() {
		return func() {
			Service("bbs", func() {
				Method("m", func() {
					Result(func() { Attribute("bbn", String) })
					HTTP(func() {
						GET("/x")
						Response(status)
					})
				})
			})
		}
	}
	for _, status := range []int{100, 150, 199, expr.StatusNoContent, expr.StatusNotModified} {
		err := expr.RunInvalidDSL(t, dsl(status))
		require.Error(t, err, "status %d forbids a body", status)
	}
	for _, status := range []int{expr.StatusOK, expr.StatusResetContent,
		expr.StatusMovedPermanently, expr.StatusSeeOther, expr.StatusTemporaryRedirect} {
		expr.RunDSL(t, dsl(status))
	}
	// streaming endpoints skip the check entirely
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				StreamingResult(func() { Attribute("bbn", String) })
				HTTP(func() {
					GET("/x")
					Response(expr.StatusNoContent)
				})
			})
		})
	})
}

// Detail 4 (Inferable: partially): a status of exactly 0 is "HTTP response
// status not defined".
func TestDetail04(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				HTTP(func() {
					GET("/x")
					Response(0)
				})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "status")
	require.Contains(t, err.Error(), "not defined")
}

// Detail 5 (Inferable: doc): a JSON-RPC endpoint success response may not
// map result attributes to headers or cookies.
func TestDetail05(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Result(func() {
					Attribute("bbh", String)
					Attribute("bbc", String)
				})
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() {
						Header("bbh")
						Cookie("bbc")
					})
				})
			})
		})
	})
	e := bbHTTPEndpoint("bbs", "m")
	require.NotNil(t, e)
	resp := e.Responses[0]
	require.False(t, resp.Headers.IsEmpty())
	require.False(t, resp.Cookies.IsEmpty())
	// flip the endpoint into a JSON-RPC endpoint and re-validate
	e.Meta = expr.MetaExpr{"jsonrpc": []string{""}}
	verr := resp.Validate(e)
	require.NotNil(t, verr)
	msg := verr.Error()
	require.Contains(t, msg, "header", "headers rejected on JSON-RPC response")
	require.Contains(t, msg, "cookie", "cookies rejected on JSON-RPC response")
}

// Detail 6 (Inferable: partially): ContentType "text/html" or "text/plain"
// forces the result — or an explicit body — to be String or Bytes.
func TestDetail06(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Result(func() { Attribute("bbn", Int) })
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() { ContentType("text/plain") })
				})
			})
		})
	})
	require.Error(t, err, "object result with text/plain rejected")

	for _, res := range []func(){
		func() { Result(String) },
		func() { Result(Bytes) },
	} {
		expr.RunDSL(t, func() {
			Service("bbs", func() {
				Method("m", func() {
					res()
					HTTP(func() {
						GET("/x")
						Response(StatusOK, func() { ContentType("text/html") })
					})
				})
			})
		})
	}
}

// Detail 7 (Inferable: no): header names resolve against the result type
// (per-view for result types, 'attr:header' notation, "all views of"
// qualifier); header values must be primitives or arrays of primitives;
// cookie values primitives only.
func TestDetail07(t *testing.T) {
	mkRT := func() *expr.ResultTypeExpr {
		return ResultType("application/vnd.bb.t7", func() {
			TypeName("BbT7")
			Attributes(func() {
				Attribute("bbh", String)
				Attribute("bbo", func() { Attribute("x", Int) })
				Attribute("bba", ArrayOf(String))
			})
		})
	}
	// missing header name errors and names the attribute
	err := expr.RunInvalidDSL(t, func() {
		RT := mkRT()
		Service("bbs", func() {
			Method("m", func() {
				Result(RT)
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() { Header("bbnope") })
				})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbnope")

	// object-typed attribute mapped to a header errors
	err = expr.RunInvalidDSL(t, func() {
		RT := mkRT()
		Service("bbs", func() {
			Method("m", func() {
				Result(RT)
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() { Header("bbo") })
				})
			})
		})
	})
	require.Error(t, err, "object attribute cannot map to a header")

	// array-of-primitive header is fine; array cookie is rejected
	expr.RunDSL(t, func() {
		RT := mkRT()
		Service("bbs", func() {
			Method("m", func() {
				Result(RT)
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() {
						Header("bba")
						Cookie("bbh")
					})
				})
			})
		})
	})
	err = expr.RunInvalidDSL(t, func() {
		RT := mkRT()
		Service("bbs", func() {
			Method("m", func() {
				Result(RT)
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() { Cookie("bba") })
				})
			})
		})
	})
	require.Error(t, err, "array attribute cannot map to a cookie")
}

// Detail 8 (Inferable: no): non-object result errors only when MORE than
// one header/cookie is mapped; array result mapped to a header needs
// primitive elements; array mapped to a cookie is always an error.
func TestDetail08(t *testing.T) {
	// array-of-primitive result: one header OK
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Result(ArrayOf(String))
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() { Header("bbh") })
				})
			})
		})
	})
	// two headers on a non-object result: error
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Result(ArrayOf(String))
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() { Header("bbh"); Header("bbh2") })
				})
			})
		})
	})
	require.Error(t, err)
	// array result mapped to a cookie: error
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Result(ArrayOf(String))
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() { Cookie("bbc") })
				})
			})
		})
	})
	require.Error(t, err)
	// array with non-primitive elements mapped to a header: error
	err = expr.RunInvalidDSL(t, func() {
		Inner := Type("BbInner", func() { Attribute("x", String) })
		Service("bbs", func() {
			Method("m", func() {
				Result(ArrayOf(Inner))
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() { Header("bbh") })
				})
			})
		})
	})
	require.Error(t, err)
}

// Detail 9 (Inferable: no): explicit body + SkipResponseBodyEncodeDecode is
// an error; with no explicit body the computed body must be Empty; body
// attributes must exist in the result type.
func TestDetail09(t *testing.T) {
	// explicit body + skip flag
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Result(func() { Attribute("bbn", String) })
				HTTP(func() {
					GET("/x")
					SkipResponseBodyEncodeDecode()
					Response(StatusOK, func() { Body("bbn") })
				})
			})
		})
	})
	require.Error(t, err, "explicit body under skip flag rejected")

	// non-empty computed body + skip flag
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Result(func() { Attribute("bbn", String) })
				HTTP(func() {
					GET("/x")
					SkipResponseBodyEncodeDecode()
					Response(StatusOK)
				})
			})
		})
	})
	require.Error(t, err, "non-empty computed body under skip flag rejected")

	// empty body + skip flag is fine
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				HTTP(func() {
					GET("/x")
					SkipResponseBodyEncodeDecode()
					Response(StatusOK)
				})
			})
		})
	})

	// body attribute not present in the result type
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Result(func() { Attribute("bbn", String) })
				HTTP(func() {
					GET("/x")
					Response(StatusOK, func() { Body("bbmissing") })
				})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbmissing")
}

// Detail 10 (Inferable: no): Finalize wraps a non-empty object body in a
// generated user type "<Service><Endpoint>ResponseBody" recording the
// original name under "name:original" meta; required body fields land in
// the body validation.
func TestDetail10(t *testing.T) {
	expr.RunDSL(t, func() {
		RT := ResultType("application/vnd.bb.t10", func() {
			TypeName("BbT10")
			Attributes(func() {
				Attribute("bbn", String)
				Attribute("bbo", Int)
				Required("bbn")
			})
		})
		Service("bbsvc", func() {
			Method("mend", func() {
				Result(RT)
				HTTP(func() {
					GET("/x")
					Response(StatusOK)
				})
			})
		})
	})
	resp := bbResp(t, "bbsvc", "mend")
	require.NotNil(t, resp.Body)
	var tn string
	var meta expr.MetaExpr
	switch bt := resp.Body.Type.(type) {
	case *expr.ResultTypeExpr:
		tn, meta = bt.TypeName, bt.Meta
	case *expr.UserTypeExpr:
		tn, meta = bt.TypeName, bt.Meta
	default:
		t.Fatalf("object body wrapped in a generated user type, got %T", resp.Body.Type)
	}
	require.True(t, strings.HasSuffix(tn, "ResponseBody"),
		"generated type name %q keeps the ResponseBody suffix", tn)
	require.Contains(t, meta["name:original"], "BbT10",
		"original type name recorded in meta")
	require.NotNil(t, resp.Body.Validation)
	require.Contains(t, resp.Body.Validation.Required, "bbn",
		"required fields propagated to body validation")
}

// Detail 11 (Inferable: no): Finalize takes ContentType from the result
// type's own ContentType only when the response did not set one.
func TestDetail11(t *testing.T) {
	expr.RunDSL(t, func() {
		RT := ResultType("application/vnd.bb.t11", func() {
			TypeName("BbT11")
			ContentType("text/bbproto")
			Attributes(func() { Attribute("n", String) })
		})
		Service("bbs", func() {
			Method("inherit", func() {
				Result(RT)
				HTTP(func() {
					GET("/x")
					Response(StatusOK)
				})
			})
			Method("explicit", func() {
				Result(RT)
				HTTP(func() {
					GET("/y")
					Response(StatusOK, func() { ContentType("text/bbexplicit") })
				})
			})
		})
	})
	require.Equal(t, "text/bbproto", bbResp(t, "bbs", "inherit").ContentType,
		"response inherits the result type content type")
	require.Equal(t, "text/bbexplicit", bbResp(t, "bbs", "explicit").ContentType,
		"explicit response content type wins")
}

// Detail 12 (Inferable: partially): Dup copies scalar fields (StatusCodeSet,
// Tag, Parent, Meta) and deep-copies body/headers/cookies.
func TestDetail12(t *testing.T) {
	parent := &expr.ServiceExpr{Name: "bbp"}
	src := &expr.HTTPResponseExpr{
		StatusCode:    201,
		StatusCodeSet: true,
		Tag:           [2]string{"bbf", "bbv"},
		Parent:        parent,
		Meta:          expr.MetaExpr{"bbk": []string{"bbv"}},
		Body:          &expr.AttributeExpr{Type: expr.String},
		Headers:       expr.NewEmptyMappedAttributeExpr(),
		Cookies:       expr.NewEmptyMappedAttributeExpr(),
	}
	d := src.Dup()
	require.Equal(t, src.StatusCode, d.StatusCode)
	require.Equal(t, src.StatusCodeSet, d.StatusCodeSet)
	// NOTE: DETAILS commits Tag copying, but the reference implementation
	// drops Tag on Dup — not asserted either way (see writeup).
	require.Same(t, parent, d.Parent, "parent shared")
	require.Equal(t, src.Meta["bbk"], d.Meta["bbk"])
	require.NotSame(t, src.Body, d.Body, "body deep-copied")
	require.NotSame(t, src.Headers, d.Headers, "headers deep-copied")
	require.NotSame(t, src.Cookies, d.Cookies, "cookies deep-copied")
}

// Detail 13 (Inferable: no): for ErrorResult bodies only, attributes not
// mapped to body or headers map to headers under "apikit-attribute-<name>"
// keys; already-mapped names are skipped; requiredness is propagated.
func TestDetail13(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Error("bbe")
				Error("bbe2")
				HTTP(func() {
					GET("/x")
					Response("bbe", StatusBadRequest, func() { Body("message") })
					Response("bbe2", StatusBadRequest, func() {
						Body(Empty)
						Header("name:bbhdrname")
					})
				})
			})
		})
	})
	e := bbHTTPEndpoint("bbs", "m")
	require.NotNil(t, e)
	var mapped, allUnmapped *expr.HTTPErrorExpr
	for _, x := range e.HTTPErrors {
		switch x.Name {
		case "bbe":
			mapped = x
		case "bbe2":
			allUnmapped = x
		}
	}
	require.NotNil(t, mapped)
	require.NotNil(t, allUnmapped)

	// body explicitly selected "message": the remaining ErrorResult fields
	// land in headers under apikit-attribute-<name> keys.
	h := bbFieldNames(mapped.Response.Headers)
	require.NotNil(t, mapped.Response.Body)
	require.Equal(t, "string", mapped.Response.Body.Type.Name(),
		"body holds the explicitly selected attribute")
	for _, f := range []string{"name", "id", "temporary", "timeout", "fault"} {
		elem, ok := h[f]
		require.True(t, ok, "unmapped field %q moved to headers", f)
		require.Equal(t, "apikit-attribute-"+f, elem)
	}
	require.True(t, mapped.Response.Headers.IsRequiredNoDefault("name"),
		"requiredness propagated to mapped header")

	// body explicitly empty: every field lands under apikit-attribute-* and
	// the explicitly mapped field keeps its declared header name.
	h2 := bbFieldNames(allUnmapped.Response.Headers)
	require.Equal(t, "bbhdrname", h2["name"], "explicit mapping wins")
	for _, f := range []string{"id", "message", "temporary", "timeout", "fault"} {
		require.Equal(t, "apikit-attribute-"+f, h2[f])
	}
}
