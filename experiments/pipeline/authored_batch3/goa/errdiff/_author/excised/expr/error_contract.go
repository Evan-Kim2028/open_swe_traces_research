// This file builds the error definition that a method inherits from its
// service or API. HTTP and gRPC settings may change how the error is sent, but
// they do not change the service error value.
package expr

import (
	_ "reflect"
	_ "slices"
)

type (
	// attributePair identifies two nodes already compared while traversing
	// recursive error types.
	attributePair struct {
		first  *AttributeExpr
		second *AttributeExpr
	}

	// effectiveErrorCopier copies an inherited error without changing the
	// evaluated design. The maps reconnect recursive types to their copies.
	effectiveErrorCopier struct {
		attributes map[*AttributeExpr]*AttributeExpr
		userTypes  map[UserType]UserType
	}
)

// equivalentErrorAttributes reports whether two error attributes generate the
// same service value contract. Descriptions and examples are documentation;
// types, validations, defaults, and metadata affect generated code or runtime
// behavior and must match.
func equivalentErrorAttributes(first, second *AttributeExpr) bool {
	panic("excised: equivalentErrorAttributes")
}

// differingErrorQualifierSettings lists error settings that would change the
// generated service error returned to callers.
func differingErrorQualifierSettings(first, second *AttributeExpr) []string {
	panic("excised: differingErrorQualifierSettings")
}

// effectiveErrorAttribute returns a detached copy with References and Bases
// applied by AttributeExpr.Finalize. Validation can therefore compare the
// value contracts code generation will see without mutating evaluated design.
func effectiveErrorAttribute(source *AttributeExpr) *AttributeExpr {
	panic("excised: effectiveErrorAttribute")
}

// attribute copies one attribute shell before following its type and
// inheritance edges so self-recursive graphs terminate on the copied shell.
func (c *effectiveErrorCopier) attribute(source *AttributeExpr) *AttributeExpr {
	panic("excised: effectiveErrorCopier.attribute")
}

// cloneErrorValidation detaches slices and scalar pointers that
// ValidationExpr.Dup deliberately shares with its source.
func cloneErrorValidation(source *ValidationExpr) *ValidationExpr {
	panic("excised: cloneErrorValidation")
}

// cloneErrorContractValue copies the collection values accepted by defaults,
// enum validations, and examples. Primitive values are immutable and can be
// shared safely.
func cloneErrorContractValue(source any) any {
	panic("excised: cloneErrorContractValue")
}

// dataTypes reconnects inheritance declarations to the same copied graph used
// by attribute types.
func (c *effectiveErrorCopier) dataTypes(source []DataType) []DataType {
	panic("excised: effectiveErrorCopier.dataTypes")
}

// dataType copies each concrete type without registering generated result
// types. User-type shells are installed before their attributes are followed.
func (c *effectiveErrorCopier) dataType(source DataType) DataType {
	panic("excised: effectiveErrorCopier.dataType")
}

// equivalentErrorAttributeNodes compares every contract-bearing node while
// stopping when recursive user types revisit the same declaration pair.
func equivalentErrorAttributeNodes(first, second *AttributeExpr, seen map[attributePair]struct{}) bool {
	panic("excised: equivalentErrorAttributeNodes")
}

// equivalentErrorValidation compares validation behavior independently of the
// authored order of required fields and enum values.
func equivalentErrorValidation(first, second *ValidationExpr) bool {
	panic("excised: equivalentErrorValidation")
}

// equivalentStringSet reports whether both slices contain the same distinct
// strings; validation order does not affect runtime behavior.
func equivalentStringSet(first, second []string) bool {
	panic("excised: equivalentStringSet")
}

// equivalentValueSet reports whether both enum lists contain the same values
// regardless of declaration order.
func equivalentValueSet(first, second []any) bool {
	panic("excised: equivalentValueSet")
}

// equivalentErrorMetadata compares metadata keys and ordered values while
// treating nil and empty maps or value slices as the same absent content.
func equivalentErrorMetadata(first, second MetaExpr) bool {
	panic("excised: equivalentErrorMetadata")
}
