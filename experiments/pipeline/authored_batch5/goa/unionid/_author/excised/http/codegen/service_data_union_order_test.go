// This file verifies deterministic HTTP wire union identity and confirms that
// detached wire expressions do not retain service package ownership.
package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	_ "example.internal/apikit/v3/codegen/testutil"
	"example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
)



func TestCollectHTTPUnionTypesRejectsBranchesWithTheSameGoName(t *testing.T) {
	union := makeUnionForOrderTest("Value", "foo-bar", "foo_bar")
	catalog, _ := testWireTypeCatalog(t)
	catalog.collect(&expr.AttributeExpr{Type: union}, wireAttribute, wireTypePolicy{})

	err := catalog.Declare()

	require.ErrorContains(t, err, `OneOf "Value" branches "foo-bar" and "foo_bar" both generate Go name "FooBar"`)
	require.ErrorContains(t, err, "rename one of the branches")
	require.NotContains(t, err.Error(), "TypeName")
}








func TestMakeHTTPTypeRemovesServicePackageOwnershipFromWireCopy(t *testing.T) {
	nested := &expr.UserTypeExpr{
		TypeName: "Nested",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "choice", Attribute: &expr.AttributeExpr{Type: makeUnionForOrderTest("Choice", "text", "number")}},
			},
			Meta: expr.MetaExpr{"struct:pkg:path": {"service/types"}},
		},
	}
	outer := &expr.UserTypeExpr{
		TypeName: "Envelope",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "nested", Attribute: &expr.AttributeExpr{Type: nested}},
			},
			Meta: expr.MetaExpr{"struct:pkg:path": {"service/types"}},
		},
	}

	wire := makeHTTPType(&expr.AttributeExpr{Type: outer})
	wireOuter := wire.Type.(expr.UserType)
	wireNested := expr.AsObject(wireOuter.Attribute().Type).Attribute("nested").Type.(expr.UserType)

	require.NotContains(t, wireOuter.Attribute().Meta, "struct:pkg:path")
	require.NotContains(t, wireNested.Attribute().Meta, "struct:pkg:path")
	require.Contains(t, outer.Attribute().Meta, "struct:pkg:path")
	require.Contains(t, nested.Attribute().Meta, "struct:pkg:path")
}

func TestStreamingHTTPTypeRemovesServicePackageOwnershipFromWireCopy(t *testing.T) {
	nested := &expr.UserTypeExpr{
		TypeName: "Nested",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}},
			},
			Meta: expr.MetaExpr{"struct:pkg:path": {"service/types"}},
		},
	}
	outer := &expr.UserTypeExpr{
		TypeName: "Envelope",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "nested", Attribute: &expr.AttributeExpr{Type: nested}},
			},
			Meta: expr.MetaExpr{"struct:pkg:path": {"service/types"}},
		},
	}
	body := &expr.AttributeExpr{Type: outer}
	endpoint := &expr.HTTPEndpointExpr{StreamingBody: body}

	wire := new(shapedBodies).streaming(endpoint)
	wireOuter := wire.Type.(expr.UserType)
	wireNested := expr.AsObject(wireOuter.Attribute().Type).Attribute("nested").Type.(expr.UserType)

	require.NotContains(t, wireOuter.Attribute().Meta, "struct:pkg:path")
	require.NotContains(t, wireNested.Attribute().Meta, "struct:pkg:path")
	require.Contains(t, outer.Attribute().Meta, "struct:pkg:path")
	require.Contains(t, nested.Attribute().Meta, "struct:pkg:path")
}

func sameShapedValueUnionDSL() {
	dsl.Attribute("bool", dsl.Boolean)
	dsl.Attribute("number", dsl.Float64)
}

// unionBranchOrderDSL returns the same authored union through response objects
// that reach its nested branch type in different orders.
func unionBranchOrderDSL() {
	identifier := dsl.Type("Identifier", dsl.String)
	label := dsl.Type("Label", func() {
		dsl.Attribute("value", dsl.String)
		dsl.Required("value")
	})
	snapshot := dsl.Type("Snapshot", func() {
		dsl.Attribute("id", identifier)
		dsl.Attribute("labels", dsl.ArrayOf(label))
		dsl.Required("id", "labels")
	})
	noEdit := dsl.Type("NoEdit", func() {})
	editable := dsl.Type("Editable", func() {
		dsl.Attribute("snapshot", snapshot)
		dsl.Required("snapshot")
	})
	edit := dsl.Type("Edit", func() {
		dsl.OneOf("relationship", func() {
			dsl.Attribute("none", noEdit)
			dsl.Attribute("editable", editable)
		})
		dsl.Required("relationship")
	})
	withEarlierSnapshot := dsl.Type("WithEarlierSnapshot", func() {
		dsl.Attribute("snapshot", snapshot)
		dsl.Attribute("edit", edit)
		dsl.Required("snapshot", "edit")
	})
	withEditOnly := dsl.Type("WithEditOnly", func() {
		dsl.Attribute("edit", edit)
		dsl.Required("edit")
	})

	dsl.Service("values", func() {
		dsl.Method("with_earlier_snapshot", func() {
			dsl.Result(withEarlierSnapshot)
			dsl.HTTP(func() {
				dsl.GET("/with-earlier-snapshot")
			})
		})
		dsl.Method("with_edit_only", func() {
			dsl.Result(withEditOnly)
			dsl.HTTP(func() {
				dsl.GET("/with-edit-only")
			})
		})
	})
}

func collectHTTPUnionTypeNames(t *testing.T, att *expr.AttributeExpr) map[string]string {
	t.Helper()
	catalog, generation := testWireTypeCatalog(t)
	catalog.collect(att, wireAttribute, wireTypePolicy{})
	linkTestWireTypeCatalog(t, generation, catalog)

	names := make(map[string]string, len(catalog.unions))
	for _, record := range catalog.unions {
		names[record.attribute.Type.Name()] = record.data.Name
	}
	return names
}

func makeUnionForOrderTest(typeName string, variants ...string) *expr.Union {
	values := make([]*expr.NamedAttributeExpr, len(variants))
	for i, variant := range variants {
		values[i] = &expr.NamedAttributeExpr{
			Name: variant,
			Attribute: &expr.AttributeExpr{
				Type: expr.String,
			},
		}
	}
	return &expr.Union{TypeName: typeName, Values: values}
}
