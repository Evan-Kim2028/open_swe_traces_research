// This file copies complete attribute graphs for generators that must keep a
// private expression snapshot and still resolve every copied node back to the
// exact input node used during planning.
package expr

import (
	_ "fmt"
	"reflect"
	_ "slices"
)

type (
	// AttributeGraphCopier copies one connected attribute graph without merging
	// distinct user types or breaking recursive links. Copies retain the authored
	// declaration and finalized state of every attribute. Reuse one copier for
	// every root that must share copied nodes.
	AttributeGraphCopier struct {
		attributes map[*AttributeExpr]*AttributeExpr
		originals  map[*AttributeExpr]*AttributeExpr
		types      map[DataType]DataType
	}

	// attributeValueReference identifies mutable data on the active copy path.
	// Slice length and capacity distinguish separate views of one array.
	attributeValueReference struct {
		typeOf   reflect.Type
		pointer  uintptr
		length   int
		capacity int
	}
)

// NewAttributeGraphCopier creates a copier for one expression graph. Call Copy
// for each root that belongs to that graph, then use Original when a generator
// must resolve a copied node through declarations planned from the input graph.
func NewAttributeGraphCopier() *AttributeGraphCopier {
	panic("excised: NewAttributeGraphCopier")
}

// Copy returns a deep copy of attribute. Shared and recursive nodes remain
// shared in the result. Calling Copy with a result made by this copier returns
// that result unchanged.
func (c *AttributeGraphCopier) Copy(attribute *AttributeExpr) *AttributeExpr {
	panic("excised: AttributeGraphCopier.Copy")
}

// Original returns the input node copied to create attribute. It returns
// attribute unchanged when this copier did not create it.
func (c *AttributeGraphCopier) Original(attribute *AttributeExpr) *AttributeExpr {
	panic("excised: AttributeGraphCopier.Original")
}

// dataTypes copies a list through the same graph so repeated and recursive
// declarations remain shared.
func (c *AttributeGraphCopier) dataTypes(dataTypes []DataType) []DataType {
	panic("excised: AttributeGraphCopier.dataTypes")
}

// dataType installs each new type before copying its children so a recursive
// child points back to the same copied type.
func (c *AttributeGraphCopier) dataType(dataType DataType) DataType {
	panic("excised: AttributeGraphCopier.dataType")
}

// copyAttributeValidation copies every slice, pointer, and value held by one
// validation so changing the copy cannot change the input graph.
func copyAttributeValidation(validation *ValidationExpr) *ValidationExpr {
	panic("excised: copyAttributeValidation")
}

// copyAttributeMeta copies both the metadata map and each value slice.
func copyAttributeMeta(meta MetaExpr) MetaExpr {
	panic("excised: copyAttributeMeta")
}

// copyAttributeValue copies values used by defaults, validations, and examples
// without changing their concrete Go type.
func copyAttributeValue(value any) any {
	panic("excised: copyAttributeValue")
}

// copyAttributeReflectValue copies every mutable field reachable from value.
// Unsupported mutable values are rejected instead of being shared.
func copyAttributeReflectValue(value reflect.Value, active map[attributeValueReference]struct{}) reflect.Value {
	panic("excised: copyAttributeReflectValue")
}

// enterAttributeValue rejects a value that points back to one already being
// copied. Repeated values outside the active path are copied separately.
func enterAttributeValue(value reflect.Value, active map[attributeValueReference]struct{}) attributeValueReference {
	panic("excised: enterAttributeValue")
}

// attributeTypeContainsReference reports whether a private field could share
// mutable state with the input value.
func attributeTypeContainsReference(valueType reflect.Type) bool {
	panic("excised: attributeTypeContainsReference")
}
