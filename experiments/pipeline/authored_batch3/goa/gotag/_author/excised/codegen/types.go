package codegen

import (
	_ "fmt"
	_ "sort"
	_ "strings"

	"example.internal/apikit/v3/expr"
)

// GoNativeTypeName returns the Go built-in type corresponding to the given
// primitive type. GoNativeType panics if t is not a primitive type.
func GoNativeTypeName(t expr.DataType) string {
	panic("excised: GoNativeTypeName")
}

// IsNilable reports whether the Go type generated for t can be nil.
func IsNilable(t expr.DataType) bool {
	panic("excised: IsNilable")
}

// arrayElementIsPointer reports whether validation needs a pointer to tell a
// null element apart from the element type's zero value.
func arrayElementIsPointer(array *expr.Array, enabled bool) bool {
	panic("excised: arrayElementIsPointer")
}

// goFieldIsPointer reports whether a field in a generated Apikit service struct
// uses a pointer. Unions remain values; their discriminator represents
// presence after the transport boundary has validated the wire shape.
func goFieldIsPointer(parent *expr.AttributeExpr, name string, pointer, useDefault bool) bool {
	panic("excised: goFieldIsPointer")
}

// AttributeTags computes the struct field tags from the attribute's explicit
// struct:tag:* metadata, ignoring struct:tag:json:name which only renames the
// computed json tag (see AttributeTagsWithName).
func AttributeTags(att *expr.AttributeExpr) string {
	panic("excised: AttributeTags")
}

// AttributeTagsWithName computes the struct field tags from its metadata,
// interpreting the "struct:tag:json:name" key when present.
//
// The "struct:tag:json" meta key always takes precedence and is treated as a
// complete tag override value. When only "struct:tag:json:name" is set, Apikit
// computes the json tag and appends ",omitempty" when the field is not
// required by its parent object.
func AttributeTagsWithName(parent *expr.AttributeExpr, fieldName string, att *expr.AttributeExpr) string {
	panic("excised: AttributeTagsWithName")
}
