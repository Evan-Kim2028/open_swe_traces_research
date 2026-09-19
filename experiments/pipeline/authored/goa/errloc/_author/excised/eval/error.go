package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

type (
	// Error represents an error that occurred while evaluating the DSL.
	// It contains the name of the file and line number of where the error
	// occurred as well as the original Go error.
	Error struct {
		// GoError is the original error returned by the DSL function.
		GoError error
		// File is the path to the file containing the user code that
		// caused the error.
		File string
		// Line is the line number  that caused the error.
		Line int
	}

	// MultiError collects multiple DSL errors. It implements error.
	MultiError []*Error
)

// Error returns the error message.
func (m MultiError) Error() string { panic("excised: Error") }

// Error returns the underlying error message.
func (e *Error) Error() string { panic("excised: Error") }

// normalizeFileForPackageMatch strips @version segments from module cache paths
// so that package matching works regardless of where the module is cached.
// For example: ".../apikit/v3@v3.23.2/dsl/..." becomes ".../apikit/v3/dsl/...".
func normalizeFileForPackageMatch(file string) string { panic("excised: normalizeFileForPackageMatch") }

// computeErrorLocation implements a heuristic to find the location in the user
// code where the error occurred. It walks back the callstack until the file
// doesn't match "/apikit/design/*.go" or one of the DSL package paths.
// When successful it returns the file name and line number, empty string and
// 0 otherwise.
func computeErrorLocation() (file string, line int) { panic("excised: computeErrorLocation") }

// isGoaSourceFile reports whether file points into the Apikit module sources.
//
// This is used to robustly skip internal Apikit frames even when the runtime
// reports a call location (file:line) inside an inlined Apikit function but the
// corresponding frame Function name does not include an import path.
func isGoaSourceFile(file string) bool { panic("excised: isGoaSourceFile") }

// validationErrorLocation returns the location of the DSL that declared the
// given expression when available.
//
// The location is derived from the expression DSL function pointer and is used
// to annotate validation errors (i.e. errors returned by Validate()).
func validationErrorLocation(expr Expression) (file string, line int, ok bool) { panic("excised: validationErrorLocation") }

// dslFuncLocation returns the file and line where the given DSL function is
// declared.
//
// The returned file is relative to the current working directory when possible.
func dslFuncLocation(fn func()) (file string, line int, ok bool) { panic("excised: dslFuncLocation") }

// relativeToWorkdir returns file relative to the current working directory when
// possible, otherwise it returns file unchanged.
func relativeToWorkdir(file string) string { panic("excised: relativeToWorkdir") }

func _keepExcisedImports() {
	_ = fmt.Sprintf
	_ = os.Getwd
	_ = filepath.Separator
	_ = reflect.TypeOf
	_ = runtime.Caller
	_ = strings.TrimSpace
}
