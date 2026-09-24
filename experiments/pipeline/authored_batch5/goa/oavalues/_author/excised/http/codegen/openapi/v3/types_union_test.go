// This file verifies OpenAPI 3 schema conversion for unions, including that a
// typed owner keeps each discriminator paired with its generated member value.
package openapiv3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/http/codegen/openapi"
)


func assertUnionSchemaBranch(t *testing.T, branch *openapi.Schema, tag string, valueType openapi.Type) {
	t.Helper()
	assert.Equal(t, openapi.Type(openapi.Object), branch.Type)
	assert.Equal(t, []string{"type", "value"}, branch.Required)
	require.Contains(t, branch.Properties, "type")
	assert.Equal(t, []any{tag}, branch.Properties["type"].Enum)
	require.Contains(t, branch.Properties, "value")
	assert.Equal(t, valueType, branch.Properties["value"].Type)
}

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
