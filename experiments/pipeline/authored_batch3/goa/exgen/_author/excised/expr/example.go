// This file generates JSON-compatible examples from evaluated attributes. Each
// service, method, type, and field uses its own repeatable sequence so changes
// elsewhere do not change its example values.
package expr

import (
	_ "fmt"
	_ "math"
	_ "regexp"
	"regexp/syntax"
	_ "strings"
	_ "time"
)

const (
	maxAttempts = 500 // Max number of retries to generate valid example.
	maxLength   = 3   // Max length for array and map examples.
)

// Example returns the example set on the attribute at design time. If there
// isn't such a value then Example computes a random value for the attribute
// using the given random value producer.
func (a *AttributeExpr) Example(r *ExampleGenerator) any {
	panic("excised: AttributeExpr.Example")
}

// NewLength returns an int that validates the generator attribute length
// validations if any.
func NewLength(a *AttributeExpr, r *ExampleGenerator) int {
	panic("excised: NewLength")
}

func hasLengthValidation(a *AttributeExpr) bool {
	panic("excised: hasLengthValidation")
}

func hasEnumValidation(a *AttributeExpr) bool {
	panic("excised: hasEnumValidation")
}

func hasFormatValidation(a *AttributeExpr) bool {
	panic("excised: hasFormatValidation")
}

func hasPatternValidation(a *AttributeExpr) bool {
	panic("excised: hasPatternValidation")
}

func hasMinMaxValidation(a *AttributeExpr) bool {
	panic("excised: hasMinMaxValidation")
}

// byLength generates a random size array of examples based on what's given.
func byLength(a *AttributeExpr, r *ExampleGenerator) any {
	panic("excised: byLength")
}

// byEnum returns a random selected enum value.
func byEnum(a *AttributeExpr, r *ExampleGenerator) any {
	panic("excised: byEnum")
}

// byFormat returns a random example based on the format the user asks.
func byFormat(a *AttributeExpr, r *ExampleGenerator) any {
	panic("excised: byFormat")
}

// byPattern generates a random value that satisfies the pattern.
//
// Note: if multiple patterns are given, only one of them is used.
func byPattern(a *AttributeExpr, r *ExampleGenerator) any {
	panic("excised: byPattern")
}

func patgen(re *syntax.Regexp, r *ExampleGenerator) string {
	panic("excised: patgen")
}

func byMinMax(a *AttributeExpr, r *ExampleGenerator) any {
	panic("excised: byMinMax")
}

func checkPattern(a *AttributeExpr, example any) bool {
	panic("excised: checkPattern")
}

func checkMinMaxValue(a *AttributeExpr, example any) bool {
	panic("excised: checkMinMaxValue")
}
