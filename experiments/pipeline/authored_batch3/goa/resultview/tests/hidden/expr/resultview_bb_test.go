package expr_test

// Black-box hidden tests for result types and views (resultview unit).
// Each TestDetailNN maps to the numbered commitment in _author/DETAILS.md.

import (
	"strings"
	"testing"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
	"github.com/stretchr/testify/require"
)

func bbMkRT(name, id string, fields ...string) *expr.ResultTypeExpr {
	rt := expr.NewResultTypeExpr(name, id, nil)
	obj := &expr.Object{}
	for _, f := range fields {
		*obj = append(*obj, &expr.NamedAttributeExpr{
			Name:      f,
			Attribute: &expr.AttributeExpr{Type: expr.String},
		})
	}
	rt.AttributeExpr.Type = obj
	return rt
}

func bbViewNames(rt *expr.ResultTypeExpr) []string {
	var out []string
	for _, v := range rt.Views {
		out = append(out, v.Name)
	}
	return out
}

func bbFieldAttr(rt *expr.ResultTypeExpr, name string) *expr.AttributeExpr {
	if o := expr.AsObject(rt.Type); o != nil {
		for _, f := range *o {
			if f.Name == name {
				return f.Attribute
			}
		}
	}
	return nil
}

// Detail 1 (Inferable: no): the canonical identifier drops the "+suffix"
// media-type extension but preserves other parameters; unparseable input
// returns unchanged.
func TestDetail01(t *testing.T) {
	c := expr.CanonicalIdentifier("application/vnd.bb.alpha+json")
	require.Equal(t, "application/vnd.bb.alpha", c, "+suffix dropped")

	withParam := expr.CanonicalIdentifier("application/vnd.bb.beta+json;view=tiny")
	require.True(t, strings.HasPrefix(withParam, "application/vnd.bb.beta"),
		"+suffix dropped with params kept")
	require.Contains(t, withParam, "view=tiny", "parameter preserved")

	plain := expr.CanonicalIdentifier("application/vnd.bb.gamma;view=tiny")
	require.Contains(t, plain, "application/vnd.bb.gamma")
	require.Contains(t, plain, "view=tiny")

	require.Equal(t, "%%%not-a-media-type%%%",
		expr.CanonicalIdentifier("%%%not-a-media-type%%%"),
		"unparseable identifier returned unchanged")
}

// Detail 2 (Inferable: partially): IsErrorResult matches any user type
// whose ORIGIN is the built-in error type — including generator copies.
func TestDetail02(t *testing.T) {
	require.True(t, expr.IsErrorResult(expr.ErrorResult))
	d := expr.ErrorResult.Dup(&expr.AttributeExpr{Type: expr.ErrorResult.Type})
	require.True(t, expr.IsErrorResult(d), "copy of ErrorResult still matches")
	require.False(t, expr.IsErrorResult(&expr.UserTypeExpr{
		TypeName:      "BbPlain",
		AttributeExpr: &expr.AttributeExpr{Type: expr.String},
	}))
	require.False(t, expr.IsErrorResult(expr.String))
}

// Detail 3 (Inferable: no): Origin returns the earliest declaration via the
// stored origin pointer (self when unset); Rename clears the pointer so the
// renamed type starts a new origin.
func TestDetail03(t *testing.T) {
	rt := bbMkRT("BbOrig", "application/vnd.bb.origin", "a")
	require.Same(t, rt, rt.Origin(), "origin is self when unset")

	d := rt.Dup(&expr.AttributeExpr{Type: rt.Type})
	require.Same(t, rt, d.Origin(), "copy carries the original as origin")

	d.Rename("BbRenamed")
	require.Same(t, d, d.Origin(), "renamed copy starts a new origin")
}

// Detail 4 (Inferable: partially): Dup deep-copies the user-type part,
// keeps Identifier and Views, and carries the ORIGIN (not self).
func TestDetail04(t *testing.T) {
	rt := bbMkRT("BbDup", "application/vnd.bb.dup", "a", "b")
	rt.Views = []*expr.ViewExpr{{Name: "tiny", Parent: rt,
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}}

	d, ok := rt.Dup(&expr.AttributeExpr{Type: rt.Type}).(*expr.ResultTypeExpr)
	require.True(t, ok)
	require.Equal(t, rt.Identifier, d.Identifier, "identifier kept")
	require.Equal(t, bbViewNames(rt), bbViewNames(d), "views kept")
	require.Same(t, rt, d.Origin(), "origin carried, not self")
	require.NotSame(t, rt.AttributeExpr, d.AttributeExpr, "attribute deep-copied")
}

// Detail 5 (Inferable: no): Finalize applies the explicit view first,
// ensures a default view, finalizes the user type, then gives EACH distinct
// nested result type the same treatment (deduped by identifier).
func TestDetail05(t *testing.T) {
	var Inner, Outer *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		Inner = ResultType("application/vnd.bb.in5", func() {
			TypeName("BbIn5")
			Attributes(func() { Attribute("a", String) })
		})
		Outer = ResultType("application/vnd.bb.out5", func() {
			TypeName("BbOut5")
			Attributes(func() {
				Attribute("in1", Inner)
				Attribute("in2", Inner)
				Attribute("x", String)
			})
		})
	})
	require.NotNil(t, Inner.View("default"), "nested RT got a default view")
	require.NotNil(t, Outer.View("default"), "outer RT has a default view")
	require.NotNil(t, expr.AsObject(Inner.Type), "nested RT finalized")
	require.NotNil(t, expr.AsObject(Outer.Type), "outer RT finalized")
}

// Detail 6 (Inferable: no): the default view is built from a DEEP copy of
// the type attribute; array-typed result types use the element's object
// shape; the view's parent is the result type.
func TestDetail06(t *testing.T) {
	rt := bbMkRT("BbDV", "application/vnd.bb.dv", "a", "b")
	rt.Finalize()
	v := rt.View("default")
	require.NotNil(t, v)
	require.Same(t, rt, v.Parent, "view parent is the result type")
	require.NotSame(t, rt.AttributeExpr, v.AttributeExpr,
		"default view built from a copy")

	var Elem, C *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		Elem = ResultType("application/vnd.bb.el", func() {
			TypeName("BbEl")
			Attributes(func() {
				Attribute("a", String)
				Attribute("b", Int)
			})
		})
		C = CollectionOf(Elem)
	})
	cv := C.View("default")
	require.NotNil(t, cv)
	require.Same(t, C, cv.Parent)
	require.NotNil(t, bbFieldAttrOf(cv.AttributeExpr, "a"),
		"collection default view uses the element's object shape")
	require.NotNil(t, bbFieldAttrOf(cv.AttributeExpr, "b"))
}

func bbFieldAttrOf(att *expr.AttributeExpr, name string) *expr.AttributeExpr {
	if att == nil {
		return nil
	}
	if o := expr.AsObject(att.Type); o != nil {
		for _, f := range *o {
			if f.Name == name {
				return f.Attribute
			}
		}
	}
	return nil
}

// Detail 7 (Inferable: partially): an explicit view meta triggers an
// in-place projection; failure is a panic because the view was validated
// earlier.
func TestDetail07(t *testing.T) {
	var RT *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		RT = ResultType("application/vnd.bb.ev", func() {
			TypeName("BbEV")
			Attributes(func() {
				Attribute("bbkeep", String)
				Attribute("bbdrop", Int)
			})
			View("tiny", func() { Attribute("bbkeep") })
			View("tiny")
		})
	})
	require.NotNil(t, bbFieldAttr(RT, "bbkeep"), "view field kept after projection")
	require.Nil(t, bbFieldAttr(RT, "bbdrop"),
		"field outside the explicit view dropped by in-place projection")

	bad := bbMkRT("BbP", "application/vnd.bb.p", "a")
	bad.Meta = expr.MetaExpr{expr.ViewMetaKey: []string{"bbnoview"}}
	require.Panics(t, func() { bad.Finalize() },
		"invalid explicit view panics at Finalize")
}

// Detail 8 (Inferable: no): projection is memoized by (type hash, view); an
// identifier already carrying the requested view param returns unchanged.
func TestDetail08(t *testing.T) {
	// identifier already carrying the view param returns the type unchanged
	rt := bbMkRT("BbV8", "application/vnd.bb.v8;view=tiny", "a")
	rt.Views = []*expr.ViewExpr{{
		Name:          "tiny",
		AttributeExpr: &expr.AttributeExpr{Type: rt.Type},
	}}
	p, err := expr.Project(rt, "tiny")
	require.NoError(t, err)
	require.Same(t, rt, p, "already-viewed identifier returns the same type")

	// two fields of the same nested type under the same view share the
	// projection (memoized)
	var Inner, Outer *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		Inner = ResultType("application/vnd.bb.in8", func() {
			TypeName("BbIn8")
			Attributes(func() { Attribute("a", String) })
			View("tiny", func() { Attribute("a") })
		})
		Outer = ResultType("application/vnd.bb.out8", func() {
			TypeName("BbOut8")
			Attributes(func() {
				Attribute("in1", Inner)
				Attribute("in2", Inner)
			})
			View("tiny", func() {
				Attribute("in1", func() { View("tiny") })
				Attribute("in2", func() { View("tiny") })
			})
		})
	})
	po, err := expr.Project(Outer, "tiny")
	require.NoError(t, err)
	a1 := bbFieldAttr(po, "in1")
	a2 := bbFieldAttr(po, "in2")
	require.NotNil(t, a1)
	require.NotNil(t, a2)
	require.Same(t, a1.Type, a2.Type,
		"same (type, view) projects to a single shared type")
}

// Detail 9 (Inferable: no): a projected single type — unknown view errors;
// Required filtered to view fields; description gains a " (<view> view)"
// suffix; TypeName gains Title(view) unless the view is default; the new
// type defines only a default view.
func TestDetail09(t *testing.T) {
	var Outer *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		Outer = ResultType("application/vnd.bb.t9", func() {
			TypeName("BbT9")
			Description("bb original doc")
			Attributes(func() {
				Attribute("bbx", String)
				Attribute("bby", String)
				Required("bbx", "bby")
			})
			View("tiny", func() { Attribute("bbx") })
		})
	})
	_, err := expr.Project(Outer, "bbnoview")
	require.Error(t, err, "unknown view errors")

	p, err := expr.Project(Outer, "tiny")
	require.NoError(t, err)
	require.Equal(t, "BbT9Tiny", p.TypeName, "TypeName gains Title(view)")
	require.Contains(t, p.Description, "tiny view",
		"description gains the view suffix")
	require.Equal(t, []string{"default"}, bbViewNames(p),
		"projected type defines only the default view")
	if p.Validation != nil {
		require.Contains(t, p.Validation.Required, "bbx")
		require.NotContains(t, p.Validation.Required, "bby",
			"required filtered to view fields")
	}
	require.Contains(t, p.Identifier, "view=tiny")

	// the default view does not gain a suffix
	pd, err := expr.Project(Outer, "default")
	require.NoError(t, err)
	require.Equal(t, "BbT9", pd.TypeName)
}

// Detail 10 (Inferable: no): the projection registers itself BEFORE
// recursing into fields so recursive references terminate.
func TestDetail10(t *testing.T) {
	var RT *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		RT = ResultType("application/vnd.bb.rec", func() {
			TypeName("BbRec")
			Attributes(func() {
				Attribute("bbname", String)
				Attribute("bbnext", "BbRec")
			})
			View("tiny", func() {
				Attribute("bbname")
				Attribute("bbnext", func() { View("tiny") })
			})
		})
	})
	require.NotPanics(t, func() {
		p, err := expr.Project(RT, "tiny")
		require.NoError(t, err, "recursive type projection terminates")
		require.Equal(t, "BbRecTiny", p.TypeName)
		// the recursive field resolves to the in-flight projection itself
		next := bbFieldAttr(p, "bbnext")
		require.NotNil(t, next)
		require.Same(t, p, next.Type,
			"recursive reference resolves to the in-flight projection")
	})
}

// Detail 11 (Inferable: no): projected collections — the element type must
// be a result type; the projected name is "<ElemTypeName>Collection"; the
// collection's stored DSL executes during projection.
func TestDetail11(t *testing.T) {
	var Elem, C *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		Elem = ResultType("application/vnd.bb.elc", func() {
			TypeName("BbElC")
			Attributes(func() {
				Attribute("a", String)
				Attribute("b", Int)
			})
			View("tiny", func() { Attribute("a") })
		})
		C = CollectionOf(Elem)
	})
	p, err := expr.Project(C, "tiny")
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(p.TypeName, "Collection"),
		"projected collection name %q keeps the Collection suffix", p.TypeName)
	arr, ok := p.Type.(*expr.Array)
	require.True(t, ok, "projected collection stays an array")
	elem, ok := arr.ElemType.Type.(*expr.ResultTypeExpr)
	require.True(t, ok, "element remains a result type")
	require.Contains(t, elem.Identifier, "view=tiny",
		"element type projected with the requested view")
}

// Detail 12 (Inferable: no): synthesized types reuse the source's
// generated-example identity when present; otherwise they take the
// projected identifier as UID.
func TestDetail12(t *testing.T) {
	var RT *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		RT = ResultType("application/vnd.bb.u12", func() {
			TypeName("BbU12")
			Attributes(func() { Attribute("a", String) })
			View("tiny", func() { Attribute("a") })
		})
	})
	p, err := expr.Project(RT, "tiny")
	require.NoError(t, err)
	require.NotEmpty(t, p.UID)
	require.Equal(t, p.Identifier, p.UID,
		"without a generated-example identity the UID is the projected identifier")
}

// Detail 13 (Inferable: no): field projection always duplicates the
// attribute — per-field meta never leaks across fields sharing a type.
// Nested result types pick the view from the VIEW attribute meta, else the
// field meta, else default.
func TestDetail13(t *testing.T) {
	var Inner, Outer *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		Inner = ResultType("application/vnd.bb.in13", func() {
			TypeName("BbIn13")
			Attributes(func() {
				Attribute("a", String)
				Attribute("b", Int)
			})
			View("tiny", func() { Attribute("a") })
		})
		Outer = ResultType("application/vnd.bb.out13", func() {
			TypeName("BbOut13")
			Attributes(func() {
				Attribute("inT", Inner, func() { View("tiny") })
				Attribute("inD", Inner)
			})
			View("tiny", func() {
				Attribute("inT")
				Attribute("inD")
			})
		})
	})
	p, err := expr.Project(Outer, "tiny")
	require.NoError(t, err)
	aT := bbFieldAttr(p, "inT")
	aD := bbFieldAttr(p, "inD")
	require.NotNil(t, aT)
	require.NotNil(t, aD)
	require.NotSame(t, aT, aD,
		"field attributes are duplicated, never shared")
	tRT, ok := aT.Type.(*expr.ResultTypeExpr)
	require.True(t, ok)
	require.Contains(t, tRT.Identifier, "view=tiny",
		"field-level view meta selects the nested view")
	dRT, ok := aD.Type.(*expr.ResultTypeExpr)
	require.True(t, ok)
	require.NotContains(t, dRT.Identifier, "view=tiny",
		"no view meta falls back to the default view")
}

// Detail 14 (Inferable: partially): the projected identifier sets or
// overwrites the "view" media-type parameter, preserving base and other
// params.
func TestDetail14(t *testing.T) {
	var RT *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		RT = ResultType("application/vnd.bb.t14;version=9", func() {
			TypeName("BbT14")
			Attributes(func() { Attribute("a", String) })
			View("tiny", func() { Attribute("a") })
		})
	})
	p, err := expr.Project(RT, "tiny")
	require.NoError(t, err)
	require.Contains(t, p.Identifier, "application/vnd.bb.t14",
		"base identifier preserved")
	require.Contains(t, p.Identifier, "view=tiny", "view param set")
	require.Contains(t, p.Identifier, "version=9", "other params preserved")
}

// Detail 15 (Inferable: partially): view eval name is 'view "<name>"' or
// "unnamed view", plus " of <parent>" when parented.
func TestDetail15(t *testing.T) {
	v := &expr.ViewExpr{Name: "bbv"}
	n := v.EvalName()
	require.Contains(t, n, "view")
	require.Contains(t, n, "bbv")

	unnamed := (&expr.ViewExpr{}).EvalName()
	require.Contains(t, unnamed, "unnamed")

	rt := bbMkRT("BbPar", "application/vnd.bb.par", "a")
	pv := &expr.ViewExpr{Name: "bbv", Parent: rt}
	require.Contains(t, pv.EvalName(), "of", "parented view names its parent")
}
