package expr

import (
	"fmt"
)


func arrayOf(dt DataType) *Array {
	return &Array{ElemType: &AttributeExpr{Type: dt}}
}

func mapOf(dt, kt DataType) *Map {
	return &Map{ElemType: &AttributeExpr{Type: dt}, KeyType: &AttributeExpr{Type: kt}}
}

func object(dts ...DataType) *Object {
	var obj Object = make([]*NamedAttributeExpr, len(dts))
	for i, dt := range dts {
		att := &AttributeExpr{Type: dt}
		obj[i] = &NamedAttributeExpr{
			Attribute: att,
			Name:      fmt.Sprintf("att%d", i),
		}
	}
	return &obj
}

func userType(name string, dt DataType) *UserTypeExpr {
	return &UserTypeExpr{TypeName: name, AttributeExpr: &AttributeExpr{Type: dt}}
}
