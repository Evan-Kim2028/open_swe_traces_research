// Hidden black-box tests for unit "methodval".
//
// TestDetailNN numbers match DETAILS.md lines 1..13.
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

func bbMethod(svc, m string) *expr.MethodExpr {
	s := expr.Root.Service(svc)
	if s == nil {
		return nil
	}
	return s.Method(m)
}

// Detail 1 (Inferable: partially): validation merges payload,
// streaming_payload, result, then streaming_result — then requirements,
// errors, interceptors. Assert relative order of diagnostics from a method
// violating a payload rule, a result rule, and a requirement rule at once.
func TestDetail01(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("bbp", Int, func() { Default("zzbad") })
				})
				Result(func() {
					Attribute("bbr", Int, func() { Default("zzbad") })
				})
				Security("bbbasic")
			})
		})
	})
	require.Error(t, err)
	msg := err.Error()
	ip := strings.Index(msg, "bbp")
	ir := strings.Index(msg, "bbr")
	is := strings.Index(msg, "username")
	require.GreaterOrEqual(t, ip, 0, "payload violation reported: %s", msg)
	require.GreaterOrEqual(t, ir, 0, "result violation reported: %s", msg)
	require.GreaterOrEqual(t, is, 0, "requirement violation reported: %s", msg)
	require.Less(t, ip, ir, "payload validated before result")
	require.Less(t, ir, is, "results validated before requirements")
}

// Detail 2 (Inferable: partially): a security-attribute payload field must
// be String (or named String), carry no field-type override, and no
// default. Assert each violation produces an error naming the field.
func TestDetail02(t *testing.T) {
	// non-String credential field
	err := expr.RunInvalidDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Username("bbu", Int)
					Password("bbp", String)
				})
				Security("bbbasic")
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbu", "offending field named")

	// credential field with a default
	err = expr.RunInvalidDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Username("bbu", String, func() { Default("zzdef") })
					Password("bbp", String)
				})
				Security("bbbasic")
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbu", "defaulted credential field named")

	// credential field with a field-type override
	err = expr.RunInvalidDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Username("bbu", String, func() { Meta("struct:field:type", "string") })
					Password("bbp", String)
				})
				Security("bbbasic")
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbu", "overridden credential field named")

	// clean baseline: string fields, no defaults
	expr.RunDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Username("bbu", String)
					Password("bbp", String)
				})
				Security("bbbasic")
			})
		})
	})
}

// Detail 3 (Inferable: yes): effective requirements are the method's own,
// else the service's, else the API's.
func TestDetail03(t *testing.T) {
	expr.RunDSL(t, func() {
		APIKeySecurity("bbapikey")
		BasicAuthSecurity("bbbasic")
		API("bbapi", func() {
			Security("bbapikey")
		})
		Service("bbsvcsec", func() {
			Security("bbbasic")
			Method("inherit", func() {
				Payload(func() {
					Username("u", String)
					Password("p", String)
				})
			})
		})
		Service("bbown", func() {
			Security("bbbasic")
			Method("own", func() {
				Payload(func() {
					APIKey("bbapikey", "k", String)
				})
				Security("bbapikey")
			})
		})
		Service("bbapiinherit", func() {
			Method("m", func() {
				Payload(func() {
					APIKey("bbapikey", "k", String)
				})
			})
		})
	})

	// service-level requirement inherited
	s := bbMethod("bbsvcsec", "inherit")
	require.NotNil(t, s)
	require.NotEmpty(t, s.Requirements, "service requirement inherited")
	require.Equal(t, expr.BasicAuthKind, s.Requirements[0].Schemes[0].Kind)

	// method-level requirement wins over the service's
	o := bbMethod("bbown", "own")
	require.NotNil(t, o)
	require.Len(t, o.Requirements, 1)
	require.Equal(t, expr.APIKeyKind, o.Requirements[0].Schemes[0].Kind,
		"method requirement wins over service requirement")

	// API-level requirement inherited through the service
	a := bbMethod("bbapiinherit", "m")
	require.NotNil(t, a)
	require.NotEmpty(t, a.Requirements, "API requirement inherited")
	require.Equal(t, expr.APIKeyKind, a.Requirements[0].Schemes[0].Kind)
}

// Detail 4 (Inferable: partially): each scheme kind demands its own
// payload tag — basic needs username AND password; API key needs the tag
// namespaced by scheme name; bearer, JWT and OAuth2 each need their tag.
// Assert each missing tag errors and a fully-tagged payload passes.
func TestDetail04(t *testing.T) {
	// basic auth missing the password field
	err := expr.RunInvalidDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Username("u", String) })
				Security("bbbasic")
			})
		})
	})
	require.Error(t, err)

	// API key missing its namespaced field
	err = expr.RunInvalidDSL(t, func() {
		APIKeySecurity("bbkey")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("u", String) })
				Security("bbkey")
			})
		})
	})
	require.Error(t, err)

	// JWT missing its token field
	err = expr.RunInvalidDSL(t, func() {
		JWTSecurity("bbjwt")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("u", String) })
				Security("bbjwt")
			})
		})
	})
	require.Error(t, err)

	// bearer missing its bearer-token field
	err = expr.RunInvalidDSL(t, func() {
		BearerSecurity("bbbearer")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("u", String) })
				Security("bbbearer")
			})
		})
	})
	require.Error(t, err)

	// OAuth2 missing its access-token field
	err = expr.RunInvalidDSL(t, func() {
		OAuth2Security("bboauth", func() {
			ClientCredentialsFlow("https://bb.invalid/token", "https://bb.invalid/refresh")
		})
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Attribute("u", String) })
				Security("bboauth")
			})
		})
	})
	require.Error(t, err)

	// fully tagged payloads pass
	expr.RunDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		APIKeySecurity("bbkey")
		JWTSecurity("bbjwt")
		BearerSecurity("bbbearer")
		OAuth2Security("bboauth", func() {
			ClientCredentialsFlow("https://bb.invalid/token", "https://bb.invalid/refresh")
		})
		Service("bbs", func() {
			Security("bbbasic", "bbkey", "bbjwt", "bbbearer", "bboauth")
			Method("m", func() {
				Payload(func() {
					Username("u", String)
					Password("p", String)
					APIKey("bbkey", "k", String)
					Token("t", String)
					BearerToken("bt", String)
					AccessToken("at", String)
				})
			})
		})
	})
}

// Detail 5 (Inferable: no): every scope in a requirement must exist in at
// least one of that requirement's credential-kind schemes. Assert a
// missing scope errors and a declared scope passes.
func TestDetail05(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		JWTSecurity("bbjwt", func() { Scope("bbknown") })
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Token("t", String) })
				Security("bbjwt", func() { Scope("zzmissing") })
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zzmissing", "missing scope named")

	expr.RunDSL(t, func() {
		JWTSecurity("bbjwt", func() { Scope("bbknown") })
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() { Token("t", String) })
				Security("bbjwt", func() { Scope("bbknown") })
			})
		})
	})
}

// Detail 6 (Inferable: no): a payload carrying a credential tag with no
// matching scheme kind errors (the API-key check is a prefix match).
func TestDetail06(t *testing.T) {
	// bearer-token field but only a JWT requirement
	err := expr.RunInvalidDSL(t, func() {
		JWTSecurity("bbjwt")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Token("t", String)
					BearerToken("bbbt", String)
				})
				Security("bbjwt")
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Bearer", "credential kind named")

	// api-key field with no api-key scheme
	err = expr.RunInvalidDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Username("u", String)
					Password("p", String)
					APIKey("bbmissing", "bbk", String)
				})
				Security("bbbasic")
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "API key", "credential kind named")
}

// Detail 7 (Inferable: no): "security attribute" = the five fixed
// credential tags plus any tag with the API-key prefix. Assert a field
// carrying only the apikey-prefixed meta (no DSL helper) is treated as a
// credential attribute — it may not declare a default.
func TestDetail07(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		APIKeySecurity("bbkey")
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("bbk", String, func() {
						Meta("security:apikey:bbkey")
						Default("zzdef")
					})
				})
				Security("bbkey")
			})
		})
	})
	require.Error(t, err, "apikey-prefixed meta marks a security attribute")
	require.Contains(t, err.Error(), "bbk")
}

// Detail 8 (Inferable: no): a named type counts as String only after
// unwrapping user types to their underlying attribute.
func TestDetail08(t *testing.T) {
	// named String alias satisfies the credential type rule
	expr.RunDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		Type("BbStrAlias", String)
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Username("u", "BbStrAlias")
					Password("p", String)
				})
				Security("bbbasic")
			})
		})
	})

	// named Int alias does not
	err := expr.RunInvalidDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		Type("BbIntAlias", Int)
		Service("bbs", func() {
			Method("m", func() {
				Payload(func() {
					Username("u", "BbIntAlias")
					Password("p", String)
				})
				Security("bbbasic")
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "u", "non-string-typed credential named")
}

// Detail 9 (Inferable: partially): method+service+API interceptors merge
// into the method lists in precedence order, deduplicated by name — method
// wins.
func TestDetail09(t *testing.T) {
	expr.RunDSL(t, func() {
		Interceptor("bbapi-int")
		Interceptor("bbsvc-int")
		Interceptor("bbm-int")
		Interceptor("bbshared")
		API("bbapi", func() {
			ServerInterceptor("bbapi-int", "bbshared")
		})
		Service("bbs", func() {
			ServerInterceptor("bbsvc-int")
			Method("m", func() {
				ServerInterceptor("bbm-int", "bbshared")
			})
		})
	})
	m := bbMethod("bbs", "m")
	require.NotNil(t, m)
	var names []string
	shared := 0
	for _, i := range m.ServerInterceptors {
		names = append(names, i.Name)
		if i.Name == "bbshared" {
			shared++
		}
	}
	for _, want := range []string{"bbm-int", "bbsvc-int", "bbapi-int", "bbshared"} {
		require.Contains(t, names, want)
	}
	require.Equal(t, 1, shared, "interceptor deduplicated by name")
	require.Less(t,
		indexOf(names, "bbm-int"), indexOf(names, "bbsvc-int"),
		"method interceptor precedes service interceptor: %v", names)
	require.Less(t,
		indexOf(names, "bbsvc-int"), indexOf(names, "bbapi-int"),
		"service interceptor precedes API interceptor: %v", names)
}

func indexOf(xs []string, v string) int {
	for i, x := range xs {
		if x == v {
			return i
		}
	}
	return -1
}

// Detail 10 (Inferable: no): credential tags are found by walking the
// payload's own meta, then recursively through its base user types, then
// its type. Assert a named-type payload carrying credential fields
// satisfies a basic-auth requirement.
func TestDetail10(t *testing.T) {
	expr.RunDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		CT := Type("BbCreds", func() {
			Username("u", String)
			Password("p", String)
		})
		Service("bbs", func() {
			Method("m", func() {
				Payload(CT)
				Security("bbbasic")
			})
		})
	})
}

// Detail 11 (Inferable: no): Finalize defaults nil payload/streaming-
// payload/result to empty attributes, inherits service errors by name,
// inherits requirements, and an explicit no-security declaration produces
// a single no-kind scheme.
func TestDetail11(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Error("bbsvcerr", func() { Attribute("m", String) })
			Method("bare", func() {})
			Method("ownerr", func() {
				Error("bbsvcerr", func() { Attribute("m", String) })
			})
			Method("nosec", func() {
				NoSecurity()
			})
		})
	})

	bare := bbMethod("bbs", "bare")
	require.NotNil(t, bare)
	require.NotNil(t, bare.Payload, "nil payload defaulted to empty attribute")
	require.NotNil(t, bare.Result, "nil result defaulted to empty attribute")
	found := false
	for _, e := range bare.Errors {
		if e.Name == "bbsvcerr" {
			found = true
		}
	}
	require.True(t, found, "service error inherited by name")

	own := bbMethod("bbs", "ownerr")
	require.NotNil(t, own)
	count := 0
	for _, e := range own.Errors {
		if e.Name == "bbsvcerr" {
			count++
		}
	}
	require.Equal(t, 1, count, "method's own error not shadowed/duplicated")

	nosec := bbMethod("bbs", "nosec")
	require.NotNil(t, nosec)
	require.Len(t, nosec.Requirements, 1, "no-security gives one requirement")
	require.Len(t, nosec.Requirements[0].Schemes, 1)
	require.Equal(t, expr.NoKind, nosec.Requirements[0].Schemes[0].Kind)
}

// Detail 12 (Inferable: no): inherited requirements are duplicated, not
// shared.
func TestDetail12(t *testing.T) {
	expr.RunDSL(t, func() {
		BasicAuthSecurity("bbbasic")
		Service("bbs", func() {
			Security("bbbasic")
			Method("m", func() {
				Payload(func() {
					Username("u", String)
					Password("p", String)
				})
			})
		})
	})
	svc := expr.Root.Service("bbs")
	m := svc.Method("m")
	require.NotEmpty(t, svc.Requirements)
	require.NotEmpty(t, m.Requirements)
	require.NotSame(t, svc.Requirements[0], m.Requirements[0],
		"inherited requirement is a copy, not the same pointer")
}

// Detail 13 (Inferable: partially): payload streams on client and
// bidirectional kinds, result streams on server and bidirectional kinds;
// mixed results requires both result objects present and different.
func TestDetail13(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("cs", func() {
				StreamingPayload(func() { Attribute("x", String) })
			})
			Method("ss", func() {
				StreamingResult(func() { Attribute("x", String) })
			})
			Method("bidi", func() {
				StreamingPayload(func() { Attribute("x", String) })
				StreamingResult(func() { Attribute("y", String) })
			})
			Method("mixed", func() {
				Result(func() { Attribute("r", String) })
				StreamingResult(func() { Attribute("s", String) })
			})
			Method("plain", func() {
				Result(func() { Attribute("r", String) })
			})
		})
	})

	cs := bbMethod("bbs", "cs")
	require.True(t, cs.IsStreaming())
	require.True(t, cs.IsPayloadStreaming())
	require.False(t, cs.IsResultStreaming())

	ss := bbMethod("bbs", "ss")
	require.True(t, ss.IsStreaming())
	require.False(t, ss.IsPayloadStreaming())
	require.True(t, ss.IsResultStreaming())

	bidi := bbMethod("bbs", "bidi")
	require.True(t, bidi.IsPayloadStreaming())
	require.True(t, bidi.IsResultStreaming())

	mixed := bbMethod("bbs", "mixed")
	require.True(t, mixed.HasMixedResults(),
		"Result + StreamingResult (distinct objects) is mixed")

	plain := bbMethod("bbs", "plain")
	require.False(t, plain.HasMixedResults())
	require.False(t, plain.IsStreaming())
}
