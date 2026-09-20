// This file validates values written with the Default DSL before code
// generation starts. It walks the design type and applies the same validation
// rules that generated boundary validators apply to request values.
package expr

import (
	_ "fmt"
	_ "math"
	"reflect"
	_ "sort"
	_ "unicode/utf8"

	"example.internal/apikit/v3/eval"
	_ "example.internal/apikit/v3/internal/codegenname"
	_ "example.internal/apikit/v3/pkg"
)

type (
	// defaultValueValidator records errors against the expression that owns the
	// authored default.
	defaultValueValidator struct {
		parent eval.Expression
		errors *eval.ValidationErrors
	}

	// defaultMapEntry keeps map validation deterministic even though Go map
	// iteration order is not stable.
	defaultMapEntry struct {
		key   reflect.Value
		value reflect.Value
		label string
	}
)

// validateDefaultValue verifies the complete value and every supplied value
// below it. Defaults are compile-time design data, so generators may assume
// this method has already proved their type and validation rules.
func (a *AttributeExpr) validateDefaultValue(ctx string, parent eval.Expression) *eval.ValidationErrors {
	panic("excised: AttributeExpr.validateDefaultValue")
}

// validateDefaultValues finds every authored default below an attribute. The
// visited set stops recursive user types while still validating each declared
// attribute once.
func (a *AttributeExpr) validateDefaultValues(ctx string, parent eval.Expression) *eval.ValidationErrors {
	panic("excised: AttributeExpr.validateDefaultValues")
}

// validateDefaultValuesRecursive validates one attribute and descends through
// each kind that can contain another attribute.
func (a *AttributeExpr) validateDefaultValuesRecursive(ctx string, parent eval.Expression, visited map[*AttributeExpr]struct{}) *eval.ValidationErrors {
	panic("excised: AttributeExpr.validateDefaultValuesRecursive")
}

// validate checks one value and then visits the values selected by its design
// type. A type error stops only that branch because its validations cannot be
// applied safely.
func (v *defaultValueValidator) validate(attribute *AttributeExpr, value reflect.Value, path string) {
	panic("excised: defaultValueValidator.validate")
}

// validateAny accepts the JSON-shaped values that the Go value renderer can
// specialize into source. Nil is valid inside an Any collection because the
// generated service field itself has type any.
func (v *defaultValueValidator) validateAny(value reflect.Value, path string) {
	panic("excised: defaultValueValidator.validateAny")
}

// validateRules applies the validations inherited through primitive aliases
// as well as rules written directly on the attribute.
func (v *defaultValueValidator) validateRules(attribute *AttributeExpr, value reflect.Value, path string) {
	panic("excised: defaultValueValidator.validateRules")
}

// defaultEnumValueEqual compares numeric enum values after converting both to
// the primitive selected by the design. Generated Go comparisons apply this
// same conversion to untyped enum constants.
func defaultEnumValueEqual(dataType DataType, value reflect.Value, allowed any) bool {
	panic("excised: defaultEnumValueEqual")
}

// validateArray applies the element contract to every authored element.
func (v *defaultValueValidator) validateArray(attribute *AttributeExpr, value reflect.Value, path string) {
	panic("excised: defaultValueValidator.validateArray")
}

// validateMap applies both key and value contracts to every authored entry.
func (v *defaultValueValidator) validateMap(attribute *AttributeExpr, value reflect.Value, path string) {
	panic("excised: defaultValueValidator.validateMap")
}

// validateObject checks required fields, rejects unknown fields, and applies
// each supplied field's own contract.
func (v *defaultValueValidator) validateObject(attribute *AttributeExpr, value reflect.Value, path string) {
	panic("excised: defaultValueValidator.validateObject")
}

// defaultObjectFields returns values using their design names. Struct fields
// use the same Go names as generated object literals.
func defaultObjectFields(value reflect.Value, object *Object) (map[string]reflect.Value, []string) {
	panic("excised: defaultObjectFields")
}

// validateUnion requires the canonical tagged envelope and validates only the
// branch named by its discriminator.
func (v *defaultValueValidator) validateUnion(attribute *AttributeExpr, value reflect.Value, path string) {
	panic("excised: defaultValueValidator.validateUnion")
}

// add records one design error with the expression that owns the default.
func (v *defaultValueValidator) add(format string, args ...any) {
	panic("excised: defaultValueValidator.add")
}

// addTypeError reports the concrete authored Go type and expected Apikit type.
func (v *defaultValueValidator) addTypeError(path string, value reflect.Value, dataType DataType) {
	panic("excised: defaultValueValidator.addTypeError")
}

// concreteDefaultValue removes interfaces and pointers while preserving the
// concrete value used for type and validation checks.
func concreteDefaultValue(value reflect.Value) (reflect.Value, bool) {
	panic("excised: concreteDefaultValue")
}

// defaultPrimitive returns the primitive below a chain of named types.
func defaultPrimitive(dataType DataType) Primitive {
	panic("excised: defaultPrimitive")
}

// customPrimitiveDefaultValue converts a value already declared with the exact
// custom Go type into the primitive value used by design validations. The
// metadata contains the package path and type name, so this check never needs
// to load or inspect a Go package.
func customPrimitiveDefaultValue(attribute *AttributeExpr, value reflect.Value) (reflect.Value, bool) {
	panic("excised: customPrimitiveDefaultValue")
}

// defaultPrimitiveReflectType returns the native Go type used to evaluate a
// primitive's design validations.
func defaultPrimitiveReflectType(primitive Primitive) reflect.Type {
	panic("excised: defaultPrimitiveReflectType")
}

// defaultPrimitiveValueFits reports whether a compatible numeric value can be
// represented by the generated primitive type without overflow.
func defaultPrimitiveValueFits(primitive Primitive, value reflect.Value) bool {
	panic("excised: defaultPrimitiveValueFits")
}

// defaultSignedValueFits checks signed bounds. A zero bit count means the
// generated Go int width on the current generation platform.
func defaultSignedValueFits(value reflect.Value, bits int) bool {
	panic("excised: defaultSignedValueFits")
}

// defaultUnsignedValueFits checks unsigned bounds. A zero bit count means the
// generated Go uint width on the current generation platform.
func defaultUnsignedValueFits(value reflect.Value, bits int) bool {
	panic("excised: defaultUnsignedValueFits")
}

// defaultNumber converts a numeric design value to the representation used by
// numeric validation expressions.
func defaultNumber(value reflect.Value) (float64, bool) {
	panic("excised: defaultNumber")
}

// defaultLength returns the length measured by generated validators. String
// lengths count Unicode characters; collections and bytes count elements.
func defaultLength(value reflect.Value) (int, bool) {
	panic("excised: defaultLength")
}

// defaultStringMap returns an object's or OneOf's authored fields. Callers
// already proved that value is a map.
func defaultStringMap(value reflect.Value) (map[string]reflect.Value, []string) {
	panic("excised: defaultStringMap")
}

// sortedDefaultMap returns map entries ordered by their printed key so error
// order does not change between evaluations.
func sortedDefaultMap(value reflect.Value) []defaultMapEntry {
	panic("excised: sortedDefaultMap")
}

// sortedDefaultNames returns object field names in stable order.
func sortedDefaultNames(values map[string]reflect.Value) []string {
	panic("excised: sortedDefaultNames")
}
