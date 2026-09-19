package eval

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unicode"
)

// RunDSL iterates through the root expressions and calls WalkSets on each to
// retrieve the expression sets. It iterates over the expression sets multiple
// times to first execute the DSL, then validate the resulting expressions and
// lastly to finalize them. The executed DSL may register additional roots
// during initial execution via Register to have them be executed (last) in the
// same run.
func RunDSL() error { panic("excised: RunDSL") }

// Execute runs the given DSL to initialize the given expression. It returns
// true on success. It returns false and appends to Context.Errors on failure.
// Note that Run takes care of calling Execute on all expressions that implement
// Source. This function is intended for use by expressions that run the DSL at
// declaration time rather than store the DSL for execution by the dsl engine
// (usually simple independent expressions). The DSL should use ReportError to
// record DSL execution errors.
func Execute(fn func(), def Expression) bool { panic("excised: Execute") }

// Current returns the expression whose DSL is currently being executed.
// As a special case Current returns Top when the execution stack is empty.
func Current() Expression { panic("excised: Current") }

// ReportError records a DSL error for reporting post DSL execution. It accepts
// a format and values a la fmt.Printf.
func ReportError(fm string, vals ...any) { panic("excised: ReportError") }

// IncompatibleDSL should be called by DSL functions when they are invoked in an
// incorrect context (e.g. "Params" in "Service").
func IncompatibleDSL() { panic("excised: IncompatibleDSL") }

// InvalidArgError records an invalid argument error. It is used by DSL
// functions that take dynamic arguments.
func InvalidArgError(expected string, actual any) { panic("excised: InvalidArgError") }

// TooFewArgError records a too few arguments error. It is used by DSL
// functions that take dynamic arguments.
func TooFewArgError() { panic("excised: TooFewArgError") }

// TooManyArgError records a too many arguments error. It is used by DSL
// functions that take dynamic arguments.
func TooManyArgError() { panic("excised: TooManyArgError") }

// ValidationErrors records the errors encountered when running Validate.
type ValidationErrors struct {
	Errors      []error
	Expressions []Expression
}

// Error implements the error interface.
func (verr *ValidationErrors) Error() string {
	msg := make([]string, len(verr.Errors))
	for i, err := range verr.Errors {
		expr := verr.Expressions[i]
		if file, line, ok := validationErrorLocation(expr); ok {
			msg[i] = fmt.Sprintf("[%s:%d] %s: %s", file, line, expr.EvalName(), err)
		} else {
			msg[i] = fmt.Sprintf("%s: %s", expr.EvalName(), err)
		}
	}
	return strings.Join(msg, "\n")
}

// Merge merges validation errors into the target.
func (verr *ValidationErrors) Merge(err *ValidationErrors) {
	if err == nil {
		return
	}
	verr.Errors = append(verr.Errors, err.Errors...)
	verr.Expressions = append(verr.Expressions, err.Expressions...)
}

// Add adds a validation error to the target.
func (verr *ValidationErrors) Add(def Expression, format string, vals ...any) {
	verr.AddError(def, fmt.Errorf(format, vals...))
}

// AddError adds a validation error to the target. It "flattens" validation
// errors so that the recorded errors are never ValidationErrors themselves.
func (verr *ValidationErrors) AddError(def Expression, err error) {
	var v *ValidationErrors
	if errors.As(err, &v) {
		verr.Errors = append(verr.Errors, v.Errors...)
		verr.Expressions = append(verr.Expressions, v.Expressions...)
		return
	}
	verr.Errors = append(verr.Errors, err)
	verr.Expressions = append(verr.Expressions, def)
}

// runSet executes the DSL for all expressions in the given set. The expression
// DSLs may append to the set as they execute.
func runSet(set ExpressionSet) { panic("excised: runSet") }

// prepareSet runs the pre validation steps on all the set expressions that
// define one.
func prepareSet(set ExpressionSet) { panic("excised: prepareSet") }

// validateSet runs the validation on all the set expressions that define one.
func validateSet(set ExpressionSet) { panic("excised: validateSet") }

// finalizeSet runs the finalizer on all the set expressions that define one.
func finalizeSet(set ExpressionSet) { panic("excised: finalizeSet") }

// caller returns the name of calling function.
func caller() string {
	var latest string
	for skip := 2; skip <= 5; skip++ {
		pc, _, _, ok := runtime.Caller(skip)
		if !ok {
			break
		}
		name := runtime.FuncForPC(pc).Name()
		if !strings.HasPrefix(name, "example.internal/apikit/v3/dsl.") {
			break
		}
		caller := strings.Split(strings.TrimPrefix(name, "example.internal/apikit/v3/dsl."), ".")[0]
		for _, first := range caller {
			if unicode.IsUpper(first) {
				latest = caller
			}
			break
		}
	}
	if latest != "" {
		return latest
	}
	return "<unknown>"
}

func _keepExcisedImports() {
	_ = reflect.TypeOf
}
