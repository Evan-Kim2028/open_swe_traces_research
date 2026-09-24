// This file checks that each Swagger union choice pairs its name with the
// schema for the matching value.
package openapiv2

import (
	_ "encoding/json"
	_ "sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/http/codegen/openapi"
)

func TestAttributeTypeSchemaCorrelatesUnionDiscriminatorAndValue(t *testing.T) {
	method := &expr.MethodExpr{Name: "union", Service: &expr.ServiceExpr{Name: "test"}}
	generator := expr.NewExampleGenerator(expr.NewFakerRandomizerFactory("test")).At(
		expr.MethodPayloadExampleIdentity(method),
	)
	schema := newSchemaBuilder(openapi.Values{}).attributeTypeSchemaWithPrefix(
		&expr.APIExpr{},
		unionAttribute(),
		"",
		generator,
	)

	require.Len(t, schema.AnyOf, 2)
	assertUnionSchemaBranch(t, schema.AnyOf[0], "text", openapi.String)
	assertUnionSchemaBranch(t, schema.AnyOf[1], "count", openapi.Integer)
	assert.Empty(t, schema.Properties)
}


func TestBuildAttributeSchemaKeepsValuesForNamedStringMapKeys(t *testing.T) {
	key := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{Type: expr.String},
		TypeName:      "ConnectionKind",
	}
	value := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
			&expr.NamedAttributeExpr{
				Name:      "connected",
				Attribute: &expr.AttributeExpr{Type: expr.Boolean},
			},
		}},
		TypeName: "ConnectionStatus",
	}
	attribute := &expr.AttributeExpr{Type: &expr.Map{
		KeyType:  &expr.AttributeExpr{Type: key},
		ElemType: &expr.AttributeExpr{Type: value},
	}}
	root := &expr.UserTypeExpr{AttributeExpr: attribute, TypeName: "Connections"}
	generator := expr.NewExampleGenerator(expr.NewFakerRandomizerFactory("named-string-map-key")).At(
		expr.UserTypeExampleIdentity(root),
	)

	schema := BuildAttributeSchema(&expr.APIExpr{}, attribute, generator)

	values, ok := schema.AdditionalProperties.(*openapi.Schema)
	require.True(t, ok)
	assert.Equal(t, "#/$defs/ConnectionStatus", values.Ref)
	require.Contains(t, schema.Defs, "ConnectionStatus")
	assert.Equal(t, openapi.Type(openapi.Boolean), schema.Defs["ConnectionStatus"].Properties["connected"].Type)
}

// assertUnionSchemaBranch checks one generated union branch's tag, required
// fields, and value type.
func assertUnionSchemaBranch(t *testing.T, branch *openapi.Schema, tag string, valueType openapi.Type) {
	t.Helper()
	assert.Equal(t, openapi.Type(openapi.Object), branch.Type)
	assert.Equal(t, []string{"type", "value"}, branch.Required)
	require.Contains(t, branch.Properties, "type")
	assert.Equal(t, []any{tag}, branch.Properties["type"].Enum)
	require.Contains(t, branch.Properties, "value")
	assert.Equal(t, valueType, branch.Properties["value"].Type)
}

// unionAttribute returns the string-or-integer union used by these schema tests.
func unionAttribute() *expr.AttributeExpr {
	return &expr.AttributeExpr{
		Type: &expr.Union{
			TypeName: "outcome",
			Values: []*expr.NamedAttributeExpr{
				{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
				{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int}},
			},
		},
	}
}

// namedObjectAttribute returns a named object with one field.
func namedObjectAttribute(field string) *expr.AttributeExpr {
	object := expr.Object{
		&expr.NamedAttributeExpr{
			Name:      field,
			Attribute: &expr.AttributeExpr{Type: expr.String},
		},
	}
	return &expr.AttributeExpr{
		Type: &expr.UserTypeExpr{
			AttributeExpr: &expr.AttributeExpr{Type: &object},
			TypeName:      "Shared",
		},
	}
}
