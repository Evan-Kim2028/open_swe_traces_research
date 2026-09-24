// This file verifies pointer/value semantics recorded for generated union
// branch fields.
package service

import (
	_ "testing"

	_ "github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"

	_ "example.internal/apikit/v3/codegen"
	"example.internal/apikit/v3/expr"
)


func unionWithBranchTypes() *expr.Union {
	return &expr.Union{
		TypeName: "value",
		Values: []*expr.NamedAttributeExpr{
			{Name: "array", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}},
			{Name: "bool", Attribute: &expr.AttributeExpr{Type: expr.Boolean}},
			{Name: "bytes", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
			{Name: "map", Attribute: &expr.AttributeExpr{Type: &expr.Map{
				KeyType:  &expr.AttributeExpr{Type: expr.String},
				ElemType: &expr.AttributeExpr{Type: expr.String},
			}}},
			{Name: "object", Attribute: &expr.AttributeExpr{Type: &expr.Object{}}},
			{Name: "string", Attribute: &expr.AttributeExpr{Type: expr.String}},
		},
	}
}
