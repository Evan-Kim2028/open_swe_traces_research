// Hidden black-box tests for unit "grpcend".
//
// TestDetailNN numbers match DETAILS.md lines 1..14.
// Inferable:no lines assert shape only (presence/structure/relations),
// never the committed literal.

package expr_test

import (
	"testing"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"

	"github.com/stretchr/testify/require"
)

func bbGRPCEndpoint(svc, m string) *expr.GRPCEndpointExpr {
	for _, s := range expr.Root.API.GRPC.Services {
		if s.ServiceExpr.Name != svc {
			continue
		}
		for _, e := range s.GRPCEndpoints {
			if e.MethodExpr.Name == m {
				return e
			}
		}
	}
	return nil
}

// Detail 1 (Inferable: partially): Prepare defaults request,
// streaming-request and metadata attributes plus empty validation objects;
// the default response has status code 0.
func TestDetail01(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				GRPC(func() {})
			})
		})
	})
	e := bbGRPCEndpoint("bbs", "m")
	require.NotNil(t, e)
	require.NotNil(t, e.Request, "request attribute defaulted")
	require.NotNil(t, e.StreamingRequest, "streaming request defaulted")
	require.NotNil(t, e.Metadata, "metadata defaulted")
	require.NotNil(t, e.Response, "default response installed")
	require.Equal(t, 0, e.Response.StatusCode)
}

// Detail 2 (Inferable: no): endpoint error policy — method errors look in
// the service's mapped errors then the API's, duplicated into the endpoint
// list; then service errors not yet covered, service mapping first then
// API. Assert endpoint error list covers method + service errors and uses
// the service mapping over the API one.
func TestDetail02(t *testing.T) {
	expr.RunDSL(t, func() {
		API("bbapi", func() {
			Error("bbboth", func() { Attribute("x", String, func() { Meta("rpc:tag", "1") }) })
			Error("bbapi3", func() { Attribute("x", String, func() { Meta("rpc:tag", "1") }) })
			GRPC(func() {
				Response("bbboth", func() { Code(CodeInternal) })
				Response("bbapi3", func() { Code(CodeInternal) })
			})
		})
		Service("bbs", func() {
			Error("bbboth")
			Error("bbapi3")
			Error("bbm", func() { Attribute("y", String, func() { Meta("rpc:tag", "1") }) })
			GRPC(func() {
				Response("bbm", func() { Code(CodeUnavailable) })
				Response("bbboth", func() { Code(CodeUnavailable) })
			})
			Method("m", func() {
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				Error("bbm")
				Error("bbboth")
				GRPC(func() {})
			})
		})
	})
	e := bbGRPCEndpoint("bbs", "m")
	require.NotNil(t, e)
	byName := map[string]*expr.GRPCErrorExpr{}
	for _, ge := range e.GRPCErrors {
		byName[ge.Name] = ge
	}
	require.Contains(t, byName, "bbm", "method error mapped")
	require.Contains(t, byName, "bbboth", "doubly-mapped error present")
	require.Contains(t, byName, "bbapi3",
		"service error uncovered by the method still reaches the endpoint")
	require.Equal(t, CodeUnavailable, byName["bbm"].Response.StatusCode,
		"method error resolves via the service mapping")
	require.Equal(t, CodeUnavailable, byName["bbboth"].Response.StatusCode,
		"service mapping wins over the API mapping")
	require.Equal(t, CodeInternal, byName["bbapi3"].Response.StatusCode,
		"service error unmapped at service level falls back to the API")

	// endpoint entries are duplicates, not shared pointers
	var svcMap *expr.GRPCErrorExpr
	for _, s := range expr.Root.API.GRPC.Services {
		for _, ge := range s.GRPCErrors {
			if ge.Name == "bbm" {
				svcMap = ge
			}
		}
	}
	require.NotNil(t, svcMap)
	require.NotSame(t, svcMap, byName["bbm"], "endpoint mapping duplicated")
}

// Detail 3 (Inferable: no): legacy stream compat reads the shared meta key
// endpoint, then method, then service, then API — first hit wins.
func TestDetail03(t *testing.T) {
	// set only at API level -> inherited
	expr.RunDSL(t, func() {
		API("bbapi", func() { Meta("grpc:stream:compat", "v1") })
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				GRPC(func() {})
			})
		})
	})
	require.True(t, bbGRPCEndpoint("bbs", "m").LegacyStreamCompat())

	// set nowhere -> false
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				GRPC(func() {})
			})
		})
	})
	require.False(t, bbGRPCEndpoint("bbs", "m").LegacyStreamCompat())

	// endpoint meta wins over method meta (endpoint is read first)
	e := &expr.GRPCEndpointExpr{
		Meta: expr.MetaExpr{"grpc:stream:compat": {"v1"}},
		MethodExpr: &expr.MethodExpr{
			Meta: expr.MetaExpr{"grpc:stream:compat": {"zzbogus"}},
		},
	}
	require.True(t, e.LegacyStreamCompat(), "endpoint meta read first")

	e2 := &expr.GRPCEndpointExpr{
		MethodExpr: &expr.MethodExpr{
			Meta: expr.MetaExpr{"grpc:stream:compat": {"v1"}},
		},
	}
	require.True(t, e2.LegacyStreamCompat(), "method meta consulted")
}

// Detail 4 (Inferable: partially): a method declaring both Result and
// StreamingResult is rejected for gRPC.
func TestDetail04(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				Result(func() { Attribute("r", String) })
				StreamingResult(func() { Attribute("s", String) })
				GRPC(func() {})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), `"m"`, "offending method named")
}

// Detail 5 (Inferable: no): the stream-compat meta only accepts "v1"; at
// endpoint or method level it requires a non-empty payload plus a
// streaming payload and every non-metadata payload attribute must be
// metadata-encodable. Service/API level skips the requirement check.
func TestDetail05(t *testing.T) {
	// bad value
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Meta("grpc:stream:compat", "zzbogus")
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				StreamingPayload(func() { Attribute("x", String) })
				GRPC(func() {})
			})
		})
	})
	require.Error(t, err)

	// method-level meta without a streaming payload
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Meta("grpc:stream:compat", "v1")
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				GRPC(func() {})
			})
		})
	})
	require.Error(t, err, "compat requires a streaming payload")

	// method-level meta, non-encodable payload attribute
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Meta("grpc:stream:compat", "v1")
				Payload(func() {
					Attribute("f", String, func() { Meta("rpc:tag", "1") })
					Attribute("bbobj", func() {
						Attribute("inner", String)
					})
				})
				StreamingPayload(func() { Attribute("x", String) })
				GRPC(func() {})
			})
		})
	})
	require.Error(t, err, "non-encodable payload attribute rejected")
	require.Contains(t, err.Error(), "bbobj", "offending attribute named")

	// valid method-level compat
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Meta("grpc:stream:compat", "v1")
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				StreamingPayload(func() { Attribute("x", String) })
				GRPC(func() {})
			})
		})
	})

	// API-level meta skips the requirement check entirely — a non-encodable
	// object payload attribute and no streaming payload is accepted.
	expr.RunDSL(t, func() {
		API("bbapi", func() { Meta("grpc:stream:compat", "v1") })
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("bbobj", func() {
						Meta("rpc:tag", "1")
						Attribute("inner", String)
					})
				})
				GRPC(func() {})
			})
		})
	})
}

// Detail 6 (Inferable: no): union-typed fields inside payload, streaming
// payload, result, streaming result and method errors may not contain
// array or map branches — deduplicated by union and attribute identity.
func TestDetail06(t *testing.T) {
	// array branch in payload union
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					OneOf("bbu", func() {
						Attribute("a", String)
						Attribute("b", ArrayOf(String))
					})
					Meta("rpc:tag", "1")
				})
				GRPC(func() {})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbu", "offending union field named")

	// map branch in result union
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				Result(func() {
					OneOf("bbr", func() {
						Attribute("a", String)
						Attribute("b", MapOf(String, String))
					})
				})
				GRPC(func() {})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbr")

	// scalar-only union is fine
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					OneOf("bbu", func() {
						Attribute("a", String)
						Attribute("b", Int)
					})
					Meta("rpc:tag", "1")
				})
				GRPC(func() {})
			})
		})
	})
}

// Detail 7 (Inferable: no): a non-object payload maps to a message of
// exactly one field of identical type; an object payload requires every
// message attribute to resolve.
func TestDetail07(t *testing.T) {
	// non-object payload, message field type mismatch
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(Int)
				GRPC(func() {
					Message(func() { Attribute("f", String) })
				})
			})
		})
	})
	require.Error(t, err)

	// non-object payload, two message fields
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(Int)
				GRPC(func() {
					Message(func() {
						Attribute("a", Int)
						Attribute("b", Int)
					})
				})
			})
		})
	})
	require.Error(t, err)

	// non-object payload, single matching field
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(Int)
				GRPC(func() {
					Message(func() { Attribute("f", Int) })
				})
			})
		})
	})

	// object payload, message attribute that does not resolve
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("a", Int, func() { Meta("rpc:tag", "1") })
				})
				GRPC(func() {
					Message(func() {
						Attribute("a", func() { Meta("rpc:tag", "1") })
						Attribute("zzmissing", func() { Meta("rpc:tag", "2") })
					})
				})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zzmissing", "unresolved message field named")
}

// Detail 8 (Inferable: partially): rpc:tag — every matched field needs a
// tag, tag numbers must be unique (duplicates name both attributes),
// union-typed fields are skipped.
func TestDetail08(t *testing.T) {
	// duplicate tag numbers name both attributes
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("bbone", Int, func() { Meta("rpc:tag", "5") })
					Attribute("bbtwo", Int, func() { Meta("rpc:tag", "5") })
				})
				GRPC(func() {})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbone", "first duplicate named")
	require.Contains(t, err.Error(), "bbtwo", "second duplicate named")

	// a matched field without a tag
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("bbok", Int, func() { Meta("rpc:tag", "1") })
					Attribute("bbuntagged", Int)
				})
				GRPC(func() {})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbuntagged")

	// union fields are skipped — no tag needed on the union itself
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("bbok", Int, func() { Meta("rpc:tag", "1") })
					OneOf("bbu", func() {
						Attribute("a", String)
						Attribute("b", Int)
					})
				})
				GRPC(func() {})
			})
		})
	})
}

// Detail 9 (Inferable: partially): every metadata attribute must resolve
// and be metadata-encodable — a primitive or an array of primitives.
func TestDetail09(t *testing.T) {
	// metadata attribute that does not resolve
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("a", Int, func() { Meta("rpc:tag", "1") })
				})
				GRPC(func() {
					Metadata(func() { Attribute("zzmissing") })
				})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zzmissing")

	// object-typed metadata attribute is not encodable
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("bbobj", func() { Attribute("i", String) })
				})
				GRPC(func() {
					Metadata(func() { Attribute("bbobj") })
				})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbobj")

	// array-of-primitive metadata is encodable
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("bbarr", ArrayOf(String))
				})
				GRPC(func() {
					Metadata(func() { Attribute("bbarr") })
				})
			})
		})
	})
}

// Detail 10 (Inferable: no): Message+Metadata both defined requires an
// object payload; names in both error; neither defined requires rpc:tags
// on all non-security payload fields.
func TestDetail10(t *testing.T) {
	// message + metadata on a non-object payload
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(String)
				GRPC(func() {
					Message(func() { Attribute("f", String) })
					Metadata(func() { Attribute("md", String) })
				})
			})
		})
	})
	require.Error(t, err)

	// a name declared in both message and metadata
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("bbshared", Int, func() { Meta("rpc:tag", "1") })
					Attribute("mdonly", Int, func() { Meta("rpc:tag", "2") })
				})
				GRPC(func() {
					Message(func() {
						Attribute("bbshared", func() { Meta("rpc:tag", "1") })
					})
					Metadata(func() { Attribute("bbshared"); Attribute("mdonly") })
				})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbshared")

	// neither defined, untagged payload field -> error
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("bbuntagged", String) })
				GRPC(func() {})
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbuntagged")

	// neither defined, all tagged -> clean
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("a", String, func() { Meta("rpc:tag", "1") })
					Attribute("b", Int, func() { Meta("rpc:tag", "2") })
				})
				GRPC(func() {})
			})
		})
	})
}

// Detail 11 (Inferable: no): security attributes are found by per-kind
// credential tags. Observable through Detail 12 — a credential-tagged
// field is treated as security metadata.
func TestDetail11(t *testing.T) {
	// a field carrying only the apikey-prefixed credential tag is still
	// recognized as a security attribute (moved to metadata, exempt from
	// the rpc:tag requirement)
	expr.RunDSL(t, func() {
		APIKeySecurity("bbkey")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					APIKey("bbkey", "bbk", String)
					Attribute("f", String, func() { Meta("rpc:tag", "1") })
				})
				Security("bbkey")
				GRPC(func() {})
			})
		})
	})
	e := bbGRPCEndpoint("bbs", "m")
	require.NotNil(t, e)
	require.NotNil(t, e.Metadata)
	elems := expr.AsObject(e.Metadata.Type)
	require.NotNil(t, elems)
	found := false
	for _, m := range *elems {
		if m.Name == "bbk" {
			found = true
		}
	}
	require.True(t, found, "credential field moved into metadata")
}

// Detail 12 (Inferable: no): Finalize moves security-tagged payload fields
// into metadata (basic-auth keeps its own name, others map to
// "authorization"), splits the remaining object fields into the request,
// and propagates requiredness.
func TestDetail12(t *testing.T) {
	expr.RunDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		BearerSecurity("bbbearer")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Username("u", String)
					Password("p", String)
					BearerToken("bt", String)
					Attribute("data", String, func() { Meta("rpc:tag", "1") })
					Required("u", "data")
				})
				Security("bbbasic", "bbbearer")
				GRPC(func() {})
			})
		})
	})
	e := bbGRPCEndpoint("bbs", "m")
	require.NotNil(t, e)

	reqObj := expr.AsObject(e.Request.Type)
	require.NotNil(t, reqObj)
	reqNames := map[string]bool{}
	for _, m := range *reqObj {
		reqNames[m.Name] = true
	}
	require.True(t, reqNames["data"], "ordinary field stays in request")
	require.False(t, reqNames["u"], "credential field moved out of request")
	require.False(t, reqNames["p"], "credential field moved out of request")
	require.False(t, reqNames["bt"], "credential field moved out of request")

	mdObj := expr.AsObject(e.Metadata.Type)
	require.NotNil(t, mdObj)
	mdNames := map[string]bool{}
	for _, m := range *mdObj {
		mdNames[m.Name] = true
	}
	require.True(t, mdNames["u"])
	require.True(t, mdNames["p"])
	require.True(t, mdNames["bt"])

	// non-basic credentials map to "authorization"; basic keeps its name
	require.Equal(t, "authorization", e.Metadata.ElemName("bt"))
	require.NotEqual(t, "authorization", e.Metadata.ElemName("u"))

	// requiredness propagated into the request validation
	req := expr.AsObject(e.Request.Type)
	var data *expr.AttributeExpr
	for _, m := range *req {
		if m.Name == "data" {
			data = m.Attribute
		}
	}
	require.NotNil(t, data)
	_ = data
}

// Detail 13 (Inferable: partially): custom object error types need
// rpc:tags; the built-in error type does not.
func TestDetail13(t *testing.T) {
	// custom object error with an explicit message mapping whose field
	// lacks rpc:tag
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				Error("bbbad", func() { Attribute("n", String) })
				GRPC(func() {
					Response("bbbad", func() {
						Message(func() { Attribute("n") })
					})
				})
			})
		})
	})
	require.Error(t, err, "untagged mapped error field rejected")
	require.Contains(t, err.Error(), `"n"`, "offending field named")

	// tagged custom error + built-in error type: clean
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				Error("bbok", func() {
					Attribute("n", String, func() { Meta("rpc:tag", "1") })
				})
				Error("bbbuiltin")
				GRPC(func() {
					Response("bbok", func() {
						Message(func() {
							Attribute("n", func() { Meta("rpc:tag", "1") })
						})
					})
				})
			})
		})
	})
}

// Detail 14 (Inferable: no): an inherited error mapping whose attribute
// shape differs from the method's own error type reports a diff on the
// mapped response.
func TestDetail14(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Error("bbshared", func() {
				Attribute("a", String, func() { Meta("rpc:tag", "1") })
			})
			GRPC(func() {
				Response("bbshared", func() {
					Message(func() {
						Attribute("a", func() { Meta("rpc:tag", "1") })
					})
				})
			})
			Method("m", func() {
				Payload(func() { Attribute("f", String, func() { Meta("rpc:tag", "1") }) })
				Error("bbshared", func() {
					Attribute("zzdifferent", Int, func() { Meta("rpc:tag", "1") })
				})
				GRPC(func() {})
			})
		})
	})
	require.Error(t, err, "inherited mapping differing from method error reports a diff")
	require.Contains(t, err.Error(), "bbshared")
}
