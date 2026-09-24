// Hidden black-box tests for unit "svcerrors".
//
// TestDetailNN numbers match DETAILS.md lines 1..8.
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

func bbSvcMethod(svc *expr.ServiceExpr, name string) *expr.MethodExpr {
	for _, m := range svc.Methods {
		if m.Name == name {
			return m
		}
	}
	return nil
}

func bbMethodErr(m *expr.MethodExpr, name string) *expr.ErrorExpr {
	for _, e := range m.Errors {
		if e.Name == name {
			return e
		}
	}
	return nil
}

// Detail 1 (Inferable: partially): service EvalName is "unnamed service"
// when empty else `service "<name>"`; Hash is "_service_"+name.
func TestDetail01(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbnamed", func() { Method("m", func() {}) })
	})
	svc := expr.Root.Service("bbnamed")
	require.NotNil(t, svc)
	require.Contains(t, svc.EvalName(), "bbnamed")

	anon := (&expr.ServiceExpr{}).EvalName()
	require.Contains(t, strings.ToLower(anon), "unnamed")

	h := svc.Hash()
	require.Contains(t, h, "bbnamed")
	require.NotEqual(t, "bbnamed", h, "hash carries a prefix")
}

// Detail 2 (Inferable: partially): service Validate merges each error's
// own validation then checks inline method errors. Assert a bad service
// error and an inline mismatch both report, the service error first.
func TestDetail02(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Error("bbsvcerr", func() {
				Attribute("bbn", Int, func() { Default("zzbad") })
			})
			Method("a", func() {
				Error("bbinl", func() { Attribute("x", String) })
			})
			Method("b", func() {
				Error("bbinl", func() { Attribute("x", String); Attribute("y", Int) })
			})
		})
	})
	require.Error(t, err)
	msg := err.Error()
	isvc := strings.Index(msg, "bbn")
	iinl := strings.Index(msg, "bbinl")
	require.GreaterOrEqual(t, isvc, 0, "service error violation reported: %s", msg)
	require.GreaterOrEqual(t, iinl, 0, "inline mismatch reported: %s", msg)
	require.Less(t, isvc, iinl,
		"error's own validation precedes inline consistency check")
}

// Detail 3 (Inferable: no): a "generated-constructor" error is one whose
// type is not an authored user type OR is the built-in error result type.
// Observable through Detail 4's consistency check: an inline error is
// generated-constructor (participates); an authored user type error does
// not. Asserted via D4.
func TestDetail03(t *testing.T) {
	// Two methods declaring the same-named error with an authored user
	// type do not participate in the consistency check — the type is
	// shared so there is nothing to compare.
	expr.RunDSL(t, func() {
		ET := Type("BbAuthErr", func() { Attribute("n", String) })
		Service("bbs", func() {
			Method("a", func() { Error("bbe", ET) })
			Method("b", func() { Error("bbe", ET) })
		})
	})
	m := bbSvcMethod(expr.Root.Service("bbs"), "a")
	e := bbMethodErr(m, "bbe")
	require.NotNil(t, e)
	require.Same(t, expr.Root.UserType("BbAuthErr"), e.Type.(expr.UserType).Origin(),
		"authored user type stays authored")
}

// Detail 4 (Inferable: no): inline-error consistency — service errors seed
// the seen-set; each generated-constructor method error must match the
// previously seen contract. Assert: same-name different-shape errors
// error, matching shapes pass, service-seeded contracts apply to methods.
func TestDetail04(t *testing.T) {
	// two methods, same name, different shapes -> error
	err := expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Method("a", func() {
				Error("bbinl", func() { Attribute("x", String) })
			})
			Method("b", func() {
				Error("bbinl", func() { Attribute("x", Int) })
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbinl", "offending error named")

	// two methods, same name, same shape -> clean
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("a", func() {
				Error("bbinl", func() { Attribute("x", String) })
			})
			Method("b", func() {
				Error("bbinl", func() { Attribute("x", String) })
			})
		})
	})

	// service-seeded contract constrains methods
	err = expr.RunInvalidDSL(t, func() {
		Service("bbs", func() {
			Error("bbseed", func() { Attribute("x", String) })
			Method("a", func() {
				Error("bbseed", func() { Attribute("x", String); Attribute("y", Int) })
			})
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbseed")
}

// Detail 5 (Inferable: partially): at most one field per (possibly nested)
// type may carry the error-name marker; it must be String and required.
func TestDetail05(t *testing.T) {
	// two marked fields in one type
	err := expr.RunInvalidDSL(t, func() {
		ET := Type("BbTwoMark", func() {
			ErrorName("a", String)
			ErrorName("b", String)
			Required("a", "b")
		})
		Service("bbs", func() {
			Method("m", func() { Error("bbe", ET) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "BbTwoMark")

	// non-String marked field
	err = expr.RunInvalidDSL(t, func() {
		ET := Type("BbIntMark", func() {
			ErrorName("bbim", Int)
			Required("bbim")
		})
		Service("bbs", func() {
			Method("m", func() { Error("bbe", ET) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbim")

	// marked field not required
	err = expr.RunInvalidDSL(t, func() {
		ET := Type("BbOptMark", func() {
			ErrorName("bbom", String)
		})
		Service("bbs", func() {
			Method("m", func() { Error("bbe", ET) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbom")

	// a single required String marker is fine
	expr.RunDSL(t, func() {
		ET := Type("BbOkMark", func() {
			ErrorName("bbkm", String)
			Required("bbkm")
		})
		Service("bbs", func() {
			Method("a", func() { Error("e1", ET) })
			Method("b", func() { Error("e2", ET) })
		})
	})
}

// Detail 6 (Inferable: no): ErrorExpr.Finalize — an authored
// non-error-result user type gets the error-name marker unless a field
// already carries it; a non-user-type error wraps its attribute in a user
// type named after the error.
func TestDetail06(t *testing.T) {
	expr.RunDSL(t, func() {
		UT := Type("BbErrUT", func() { Attribute("n", String) })
		Service("bbs", func() {
			Error("bbwrap", func() { Attribute("x", String) })
			Error("bbut", UT)
			Method("m", func() {})
		})
	})
	svc := expr.Root.Service("bbs")

	var wrap *expr.ErrorExpr
	for _, e := range svc.Errors {
		if e.Name == "bbwrap" {
			wrap = e
		}
	}
	require.NotNil(t, wrap)
	ut, ok := wrap.Type.(expr.UserType)
	require.True(t, ok, "non-user-type error wrapped in a user type")
	require.Contains(t, ut.Name(), "bbwrap", "wrapper named after the error")

	// authored user type gained the error-name marker somewhere in its
	// attribute tree (type-level meta or a marked field)
	marked := false
	utExpr := expr.Root.UserType("BbErrUT")
	require.NotNil(t, utExpr)
	if _, ok := utExpr.Attribute().Meta["struct:error:name"]; ok {
		marked = true
	}
	if obj := expr.AsObject(utExpr.Attribute().Type); obj != nil {
		for _, att := range *obj {
			if _, ok := att.Attribute.Meta["struct:error:name"]; ok {
				marked = true
			}
		}
	}
	require.True(t, marked, "authored error type carries the error-name marker")
}

// Detail 7 (Inferable: no): method-typed error finalization builds a
// generated user type keyed by method+error example identity, reusing the
// ORIGIN of the first previously-generated same-named error in service
// method order.
func TestDetail07(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("first", func() {
				Error("bbme", func() { Attribute("x", String) })
			})
			Method("second", func() {
				Error("bbme", func() { Attribute("x", String) })
			})
		})
	})
	svc := expr.Root.Service("bbs")
	e1 := bbMethodErr(bbSvcMethod(svc, "first"), "bbme")
	e2 := bbMethodErr(bbSvcMethod(svc, "second"), "bbme")
	require.NotNil(t, e1)
	require.NotNil(t, e2)

	u1, ok1 := e1.Type.(expr.UserType)
	u2, ok2 := e2.Type.(expr.UserType)
	require.True(t, ok1 && ok2, "method errors get generated user types")
	require.Same(t, u1.Origin(), u2.Origin(),
		"later same-named error reuses the first one's origin")
}

// Detail 8 (Inferable: no): the previous-origin search walks methods
// strictly before the current one and only picks same-named errors whose
// type is a generated user type. Assert a differently-named error does not
// share the origin, and the first method's error is the origin donor.
func TestDetail08(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("first", func() {
				Error("bbme", func() { Attribute("x", String) })
				Error("bbother", func() { Attribute("z", String) })
			})
			Method("second", func() {
				Error("bbme", func() { Attribute("x", String) })
			})
		})
	})
	svc := expr.Root.Service("bbs")
	e1 := bbMethodErr(bbSvcMethod(svc, "first"), "bbme")
	eo := bbMethodErr(bbSvcMethod(svc, "first"), "bbother")
	e2 := bbMethodErr(bbSvcMethod(svc, "second"), "bbme")

	u1 := e1.Type.(expr.UserType)
	uo := eo.Type.(expr.UserType)
	u2 := e2.Type.(expr.UserType)

	require.NotSame(t, uo.Origin(), u1.Origin(),
		"different name does not share the origin")
	require.Same(t, u1.Origin(), u2.Origin(),
		"same name in a later method reuses the earlier origin")
}
