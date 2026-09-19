package eval

import (
	"fmt"
)

// Context contains the state used by the engine to execute the DSL.
var Context *DSLContext

type (
	// DSLContext is the data structure that contains the DSL execution state.
	DSLContext struct {
		// Stack represents the current execution stack.
		Stack Stack
		// Errors contains the DSL execution errors for the current expression set.
		// Errors is an instance of MultiError.
		Errors MultiError

		// roots is the list of DSL roots as registered by all loaded DSLs.
		roots []Root
		// dslPackages keeps track of the DSL package import paths so the initiator
		// may skip any callstack frame that belongs to them when computing error
		// locations.
		dslPackages []string
	}

	// Stack represents the expression evaluation stack. The stack is appended to
	// each time the initiator executes an expression source DSL.
	Stack []Expression
)

func init() {
	Reset()
}

// Reset resets the eval context, mostly useful for tests.
func Reset() {
	Context = &DSLContext{dslPackages: []string{"example.internal/apikit/v3/eval"}}
}

// Register appends a root expression to the current Context root expressions.
// Each root expression may only be registered once.
func Register(r Root) error { panic("excised: Register") }

// Current returns current evaluation context, i.e. object being currently built
// by DSL.
func (s Stack) Current() Expression { panic("excised: Current") }

// Error builds the error message from the current context errors.
func (c *DSLContext) Error() string { panic("excised: Error") }

// Roots orders the DSL roots making sure dependencies are last. It returns an
// error if there is a dependency cycle.
func (c *DSLContext) Roots() ([]Root, error) { panic("excised: Roots") }

// Record appends an error to the context Errors field.
func (c *DSLContext) Record(err *Error) { panic("excised: Record") }

// sortDependencies sorts the depencies of the given root in the given slice.
func sortDependencies(roots []Root, root Root, depFunc func(Root) []Root) []Root { panic("excised: sortDependencies") }

// sortDependenciesR sorts the depencies of the given root in the given slice.
func sortDependenciesR(root Root, seen map[string]bool, sorted *[]Root, depFunc func(Root) []Root) { panic("excised: sortDependenciesR") }

func _keepExcisedImports() {
	_ = fmt.Sprintf
}
