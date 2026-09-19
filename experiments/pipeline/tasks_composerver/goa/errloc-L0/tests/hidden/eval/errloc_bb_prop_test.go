// Black-box property suite for errloc (evaluation error formatting and locations).
// Exported API: eval.Error.Error, eval.MultiError.Error; ReportError integration per contract.
// Seed 20260919; >=10k cases.
//
// Coverage table (contract sentence -> property):
//   "Error prints [file:line] message when File set, else underlying message"
//       -> TestErrlocErrorFormatProperty / TestErrlocErrorFormatRandom
//   "MultiError joins Error lines with newlines"
//       -> TestErrlocMultiErrorProperty / TestErrlocMultiErrorRandom
//   "ReportError records on context with location from user frames"
//       -> TestErrlocReportErrorIntegrationProperty / TestErrlocReportErrorRandom
//   "validation errors include [file:line] when source function exists"
//       -> TestErrlocValidationWithSourceProperty
//   "validation without source prints name and message only"
//       -> TestErrlocValidationNoSourceProperty
package eval_test

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"example.internal/apikit/v3/eval"
)

const (
	errlocSeed  = 20260919
	errlocCases = 10000
)

func errlocOracleError(e *eval.Error) string {
	if e.GoError != nil {
		if e.File == "" {
			return e.GoError.Error()
		}
		return fmt.Sprintf("[%s:%d] %s", e.File, e.Line, e.GoError.Error())
	}
	return ""
}

func errlocOracleMulti(errs []*eval.Error) string {
	parts := make([]string, len(errs))
	for i, e := range errs {
		parts[i] = errlocOracleError(e)
	}
	return strings.Join(parts, "\n")
}

type errlocExpr struct {
	name string
	dsl  func()
}

func (e *errlocExpr) EvalName() string { return e.name }
func (e *errlocExpr) DSL() func()      { return e.dsl }

func errlocRelFile(t *testing.T, file string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	rel, err := filepath.Rel(wd, file)
	if err != nil {
		return file
	}
	return rel
}

func TestErrlocErrorFormatProperty(t *testing.T) {
	cases := []*eval.Error{
		{GoError: errors.New("plain")},
		{GoError: errors.New("tagged"), File: "a.go", Line: 10},
		{GoError: errors.New("zero"), File: "", Line: 0},
	}
	for _, e := range cases {
		got := e.Error()
		want := errlocOracleError(e)
		if got != want {
			t.Fatalf("Error()=%q want %q", got, want)
		}
	}
}

func TestErrlocErrorFormatRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(errlocSeed))
	for i := 0; i < errlocCases; i++ {
		msg := fmt.Sprintf("err-%d", i)
		e := &eval.Error{GoError: errors.New(msg)}
		if rng.Intn(2) == 0 {
			e.File = fmt.Sprintf("f%d.go", rng.Intn(20))
			e.Line = rng.Intn(500) + 1
		}
		if e.Error() != errlocOracleError(e) {
			t.Fatalf("case %d: mismatch", i)
		}
	}
}

func TestErrlocMultiErrorProperty(t *testing.T) {
	errs := eval.MultiError{
		{GoError: errors.New("a"), File: "x.go", Line: 1},
		{GoError: errors.New("b")},
	}
	want := errlocOracleMulti(errs)
	if errs.Error() != want {
		t.Fatalf("MultiError=%q want %q", errs.Error(), want)
	}
}

func TestErrlocMultiErrorRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(errlocSeed + 1))
	for i := 0; i < errlocCases; i++ {
		n := rng.Intn(6) + 1
		slice := make([]*eval.Error, n)
		for j := 0; j < n; j++ {
			slice[j] = &eval.Error{GoError: errors.New(fmt.Sprintf("%d:%d", i, j))}
			if rng.Intn(3) != 0 {
				slice[j].File = fmt.Sprintf("p%d.go", j)
				slice[j].Line = j + 10
			}
		}
		me := eval.MultiError(slice)
		if me.Error() != errlocOracleMulti(slice) {
			t.Fatalf("case %d", i)
		}
	}
}

func TestErrlocReportErrorIntegrationProperty(t *testing.T) {
	eval.Reset()
	eval.Context.Stack = append(eval.Context.Stack, &errlocExpr{name: "svc"})
	eval.ReportError("boom")
	msg := eval.Context.Error()
	if !strings.Contains(msg, "boom") || !strings.Contains(msg, "svc") {
		t.Fatalf("ReportError message: %q", msg)
	}
}

func TestErrlocReportErrorRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(errlocSeed + 2))
	for i := 0; i < errlocCases; i++ {
		eval.Reset()
		expr := &errlocExpr{name: fmt.Sprintf("expr%d", i)}
		eval.Context.Stack = append(eval.Context.Stack, expr)
		text := fmt.Sprintf("fail-%d", rng.Intn(1000))
		eval.ReportError("%s", text)
		got := eval.Context.Error()
		if !strings.Contains(got, text) || !strings.Contains(got, expr.name) {
			t.Fatalf("case %d: %q", i, got)
		}
	}
}

func TestErrlocValidationWithSourceProperty(t *testing.T) {
	var verr eval.ValidationErrors
	dsl := func() {}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	pc := reflect.ValueOf(dsl).Pointer()
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		t.Fatal("FuncForPC nil")
	}
	_, line := fn.FileLine(pc)
	rel := errlocRelFile(t, file)

	expr := &errlocExpr{name: "field", dsl: dsl}
	verr.AddError(expr, errors.New("bad"))
	want := fmt.Sprintf("[%s:%d] field: bad", rel, line)
	if verr.Error() != want {
		t.Fatalf("got %q want %q", verr.Error(), want)
	}
}

func TestErrlocValidationNoSourceProperty(t *testing.T) {
	var verr eval.ValidationErrors
	verr.AddError(eval.Top, errors.New("bad"))
	if verr.Error() != "top-level: bad" {
		t.Fatalf("top: %q", verr.Error())
	}
	expr := &errlocExpr{name: "orphan", dsl: nil}
	verr = eval.ValidationErrors{}
	verr.AddError(expr, errors.New("bad"))
	if verr.Error() != "orphan: bad" {
		t.Fatalf("nil dsl: %q", verr.Error())
	}
}
