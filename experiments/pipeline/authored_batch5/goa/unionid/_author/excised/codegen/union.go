// This file builds a repeatable key from every detail that changes a generated
// union's Go or JSON definition.
package codegen

import (
	_ "strconv"
	"strings"

	"example.internal/apikit/v3/expr"
)

type (
	// UnionDeclarationID identifies one authored OneOf declaration and the Go
	// definition emitted for the current copy. Copies of the same authored
	// attribute share an identity, while separate OneOf declarations do not.
	UnionDeclarationID struct {
		authored   *expr.AttributeExpr
		definition UnionTypeID
	}

	// UnionTypeID identifies the Go and JSON definition emitted for a union.
	// It is distinct from expr.Union.Hash, which describes design compatibility.
	UnionTypeID string
)

// NewUnionDeclarationID returns the identity of the OneOf stored in attribute.
// Transport copies keep the authored attribute, so they find the same
// declaration when their emitted definitions also match.
func NewUnionDeclarationID(attribute *expr.AttributeExpr) UnionDeclarationID {
	panic("excised: NewUnionDeclarationID")
}

// NewUnionTypeID returns a repeatable key for union's generated Go and JSON
// definitions. The key includes the effective JSON envelope keys and every
// detail that changes a generated Go branch type: package location, field type
// metadata, and whether the value may be nil.
func NewUnionTypeID(union *expr.Union) UnionTypeID {
	panic("excised: NewUnionTypeID")
}

// writeUnionTypeID appends one union definition using length-prefixed values
// so different inputs cannot produce an ambiguous concatenation.
func writeUnionTypeID(key *strings.Builder, union *expr.Union, objects map[*expr.Object]int, unions map[*expr.Union]int, userTypes map[expr.UserType]int) {
	panic("excised: writeUnionTypeID")
}

// writeUnionAttributeID appends every attribute detail that changes generated
// Go code.
func writeUnionAttributeID(key *strings.Builder, att *expr.AttributeExpr, objects map[*expr.Object]int, unions map[*expr.Union]int, userTypes map[expr.UserType]int) {
	panic("excised: writeUnionAttributeID")
}

// writeUnionObjectID appends the inline Go struct emitted for an object.
func writeUnionObjectID(key *strings.Builder, parent *expr.AttributeExpr, object *expr.Object, objects map[*expr.Object]int, unions map[*expr.Union]int, userTypes map[expr.UserType]int) {
	panic("excised: writeUnionObjectID")
}

// writeUnionIDPart appends one unambiguous string component to key.
func writeUnionIDPart(key *strings.Builder, value string) {
	panic("excised: writeUnionIDPart")
}
