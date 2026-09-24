// This file stores alternate OpenAPI text and examples for one specification
// build. Builders read these values without changing the evaluated Apikit design.
package openapi

import (
	_ "maps"

	"example.internal/apikit/v3/eval"
	"example.internal/apikit/v3/expr"
)

type (
	// storedExample keeps an immutable example value with the exact design
	// expression used to find its translated description.
	storedExample struct {
		value  *expr.ExampleExpr
		source *expr.ExampleExpr
	}

	// Values contains alternate titles, descriptions, and examples for one
	// OpenAPI build. The zero value uses the evaluated Apikit design unchanged.
	// Methods that add values return a new independent Values.
	Values struct {
		titles       map[eval.Expression]string
		descriptions map[eval.Expression]string
		examples     map[*expr.AttributeExpr][]storedExample
	}
)

// WithTitle returns a copy of v that uses title for target.
func (v Values) WithTitle(target eval.Expression, title string) Values {
	panic("excised: Values.WithTitle")
}

// WithDescription returns a copy of v that uses description for target.
func (v Values) WithDescription(target eval.Expression, description string) Values {
	panic("excised: Values.WithDescription")
}

// WithExamples returns a copy of v that uses examples for attribute. Copies
// made from attribute by Apikit use the same examples.
func (v Values) WithExamples(attribute *expr.AttributeExpr, examples []*expr.ExampleExpr) Values {
	panic("excised: Values.WithExamples")
}

// Title returns the title stored for target or fallback when none was stored.
func (v Values) Title(target eval.Expression, fallback string) string {
	panic("excised: Values.Title")
}

// Description returns the description stored for target or fallback when none
// was stored.
func (v Values) Description(target eval.Expression, fallback string) string {
	panic("excised: Values.Description")
}

// Examples returns the examples stored for attribute or a copy of fallback
// when none were stored.
func (v Values) Examples(attribute *expr.AttributeExpr, fallback []*expr.ExampleExpr) []*expr.ExampleExpr {
	panic("excised: Values.Examples")
}

// Example returns the last stored or authored example, or generates one when
// none exists. A generator configured to suppress examples returns nil.
func (v Values) Example(attribute *expr.AttributeExpr, generator *expr.ExampleGenerator) any {
	panic("excised: Values.Example")
}

// copy returns independent maps while retaining the immutable values they
// contain. Example lists are copied again when they are changed or read.
func (v Values) copy() Values {
	panic("excised: Values.copy")
}

// storeExamples copies a complete example list while retaining the exact
// expression used to look up each translated description.
func storeExamples(examples []*expr.ExampleExpr) []storedExample {
	panic("excised: storeExamples")
}

// materializeExamples returns a fresh example list with every replacement
// description applied from its exact source expression.
func (v Values) materializeExamples(examples []storedExample) []*expr.ExampleExpr {
	panic("excised: Values.materializeExamples")
}
