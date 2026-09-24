// This file renders design values as Go expressions while type names and
// pointer choices are still available to the generator. Generated programs
// receive only the declarations and expression needed for the selected value.
package codegen

import (
	_ "fmt"
	_ "math"
	"reflect"
	_ "sort"
	_ "strconv"
	_ "strings"

	"example.internal/apikit/v3/expr"
)

type (
	// GoValueCode contains the Go source needed to build one design value.
	// Declarations must be written before Expression in the same block.
	GoValueCode struct {
		// Declarations initializes values whose addresses appear in Expression.
		Declarations []string
		// Expression is the final typed Go value.
		Expression string
	}

	// UnionConstructorResolver returns the planned constructor for one OneOf
	// branch. The returned name includes any package qualifier needed by the
	// generated file.
	UnionConstructorResolver func(attribute *expr.AttributeExpr, branch string) (string, error)

	// GoTypeLayoutResolver returns the linked Go type selected by one generated
	// package for an attribute and its pointer policy.
	GoTypeLayoutResolver interface {
		GoTypeLayout(attribute *expr.AttributeExpr, policy GoLayoutPolicy) (LinkedGoType, error)
	}

	// goValueRenderer holds the target layout and names local declarations made
	// while rendering one value.
	goValueRenderer struct {
		resolveUnion UnionConstructorResolver
		prefix       string
		nextLocal    int
		declarations []string
	}

	// goMapValue keeps one authored map entry with its stable generation order.
	goMapValue struct {
		order string
		key   reflect.Value
		value reflect.Value
	}
)

// RenderGoValue renders value using the linked Go layout selected for attribute.
// pointer reports whether the complete value is stored through a pointer.
// resolveUnion is required only when attribute contains a OneOf value.
func RenderGoValue(attribute *expr.AttributeExpr, value any, layout LinkedGoType, pointer bool, resolveUnion UnionConstructorResolver, localPrefix string) (GoValueCode, error) {
	panic("excised: RenderGoValue")
}

// planGoTypeWithAttributor records the final names selected by an attribute
// scope. Pointer choices still come only from GoTypePlan.
func planGoTypeWithAttributor(attribute *expr.AttributeExpr, policy GoLayoutPolicy, attributor Attributor) (LinkedGoType, error) {
	panic("excised: planGoTypeWithAttributor")
}

// render writes the expression for one value and records any local values
// whose addresses are required by the target layout.
func (r *goValueRenderer) render(attribute *expr.AttributeExpr, value reflect.Value, layout LinkedGoType, pointer bool) (string, error) {
	panic("excised: goValueRenderer.render")
}

// renderPrimitive writes a primitive literal and converts it to the planned
// named type when the design or field metadata requires one.
func (r *goValueRenderer) renderPrimitive(attribute *expr.AttributeExpr, value reflect.Value, layout LinkedGoType) (string, error) {
	panic("excised: goValueRenderer.renderPrimitive")
}

// renderArray writes every element using the element layout selected by the
// same generator context.
func (r *goValueRenderer) renderArray(attribute *expr.AttributeExpr, value reflect.Value, layout, shape LinkedGoType) (string, error) {
	panic("excised: goValueRenderer.renderArray")
}

// renderMap writes map entries in source order independent form so generated
// files remain stable across runs.
func (r *goValueRenderer) renderMap(attribute *expr.AttributeExpr, value reflect.Value, layout, shape LinkedGoType) (string, error) {
	panic("excised: goValueRenderer.renderMap")
}

// renderObject writes only the fields supplied by the authored value. Pointer
// fields use local declarations when Go cannot take a scalar literal address.
func (r *goValueRenderer) renderObject(attribute *expr.AttributeExpr, value reflect.Value, layout, shape LinkedGoType, pointer bool) (string, error) {
	panic("excised: goValueRenderer.renderObject")
}

// renderUnion selects the authored branch and calls the constructor retained
// by the generated service package.
func (r *goValueRenderer) renderUnion(attribute *expr.AttributeExpr, value reflect.Value, layout LinkedGoType) (string, error) {
	panic("excised: goValueRenderer.renderUnion")
}

// localName returns a name that is unique inside the caller-provided block.
func (r *goValueRenderer) localName() string {
	panic("excised: goValueRenderer.localName")
}

// orderedMapValues sorts authored map keys before rendering either side of an
// entry. This keeps local variable names stable when map values need pointers.
func orderedMapValues(attribute *expr.AttributeExpr, value reflect.Value) ([]goMapValue, error) {
	panic("excised: orderedMapValues")
}

// concreteGoValue removes interface and pointer wrappers while rejecting nil,
// which cannot satisfy a non-nil authored default.
func concreteGoValue(value reflect.Value) (reflect.Value, error) {
	panic("excised: concreteGoValue")
}

// renderPrimitiveLiteral writes a literal accepted by the primitive's native
// Go type.
func renderPrimitiveLiteral(primitive expr.Primitive, value reflect.Value) (string, error) {
	panic("excised: renderPrimitiveLiteral")
}

// renderAnyValue writes JSON-compatible Go values with explicit container
// types so the generated expression does not depend on a design-time Go type.
func renderAnyValue(value reflect.Value) (string, error) {
	panic("excised: renderAnyValue")
}

// goValueIsNil reports whether an authored Any value is nil after removing
// interface and pointer wrappers.
func goValueIsNil(value reflect.Value) bool {
	panic("excised: goValueIsNil")
}

// renderTypedCustomValue writes an authored value that already uses the exact
// custom Go type from the design.
func renderTypedCustomValue(typeName string, value reflect.Value) (string, error) {
	panic("excised: renderTypedCustomValue")
}

// validateCustomDefault checks an already-typed default against the declared
// custom Go type. Native values use the existing struct:field:type contract:
// the declared type must accept conversion from the Apikit primitive.
func validateCustomDefault(custom string, spec *ImportSpec, value reflect.Value) error {
	panic("excised: validateCustomDefault")
}

// authoredObjectField returns one supplied object field from a map or struct.
func authoredObjectField(value reflect.Value, name string, attribute *expr.AttributeExpr) (reflect.Value, bool, error) {
	panic("excised: authoredObjectField")
}

// reflectedMapValue finds a string key in maps whose key type may be string or
// any, as used by evaluated design defaults.
func reflectedMapValue(value reflect.Value, key string) (reflect.Value, bool) {
	panic("excised: reflectedMapValue")
}

// underlyingPrimitive returns the primitive beneath a named primitive type.
func underlyingPrimitive(dataType expr.DataType) expr.Primitive {
	panic("excised: underlyingPrimitive")
}
