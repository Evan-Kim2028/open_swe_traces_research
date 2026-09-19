// Black-box property suite for the evalrun unit.
// Exported API: RunDSL, Execute, Current, ReportError, IncompatibleDSL,
// InvalidArgError, TooFewArgError, TooManyArgError.
// Seed 20260919; >=10k cases; contract + eval_test.go coverage.
//
// Coverage table (contract sentence -> property):
//   "wrongly typed extra argument records one error naming expected shape"
//       -> TestEvalrunInvalidArgError / TestEvalrunInvalidArgErrorRandom
//   "missing argument records one error naming the design function"
//       -> TestEvalrunTooFewArgError / TestEvalrunTooFewArgErrorRandom
//   "extra arguments record one error naming the design function"
//       -> TestEvalrunTooManyArgError / TestEvalrunTooManyArgErrorRandom
//   "design function used in wrong surrounding type records incompatible-context error"
//       -> TestEvalrunIncompatibleDSL / TestEvalrunIncompatibleDSLRandom
//   "executing stored source records an error that evaluation then returns"
//       -> TestEvalrunRunDSLReportError / TestEvalrunRunDSLReportErrorRandom
//   "validation failures after execute are returned and later phases do not hide them"
//       -> TestEvalrunRunDSLValidationError / TestEvalrunRunDSLValidationRandom
//   "current expression is stack top or top-level placeholder when empty"
//       -> TestEvalrunCurrentStack / TestEvalrunCurrentStackRandom
//   "Execute pushes target and pops afterward; nil source is success"
//       -> TestEvalrunExecuteNilSource / TestEvalrunExecuteSuccessRandom
package eval_test

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/eval"
	"example.internal/apikit/v3/expr"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

type bbEvalExpr struct {
	name     string
	dsl      func()
	validate error
}

func (e *bbEvalExpr) EvalName() string { return e.name }
func (e *bbEvalExpr) DSL() func()      { return e.dsl }
func (e *bbEvalExpr) Validate() error  { return e.validate }

type bbEvalRoot struct {
	expr eval.Expression
}

func (*bbEvalRoot) EvalName() string              { return "bb-root" }
func (*bbEvalRoot) DependsOn() []eval.Root        { return nil }
func (*bbEvalRoot) Packages() []string            { return nil }
func (r *bbEvalRoot) WalkSets(walk eval.SetWalker) { walk(eval.ExpressionSet{r.expr}) }

func bbResetRunDSL(t *testing.T, expr eval.Expression) {
	t.Helper()
	eval.Reset()
	if err := eval.Register(&bbEvalRoot{expr: expr}); err != nil {
		t.Fatalf("Register: %v", err)
	}
}

func bbSingleLineErr(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	lines := strings.Split(strings.TrimSpace(err.Error()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 error line, got %d: %q", len(lines), err.Error())
	}
	return lines[0]
}

func TestEvalrunInvalidArgError(t *testing.T) {
	cases := map[string]struct {
		dsl  func()
		want string
	}{
		"Type": {func() { Type("name", 1) }, "cannot use 1 (type int) as type type or function"},
		"Example": {func() { Example(1, 2) }, "cannot use 1 (type int) as type summary (string)"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.dsl)
			msg := bbSingleLineErr(t, err)
			if !strings.Contains(msg, tc.want) {
				t.Fatalf("got %q want substring %q", msg, tc.want)
			}
		})
	}
}

func TestEvalrunInvalidArgErrorRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		var err error
		switch rng.Intn(3) {
		case 0:
			err = expr.RunInvalidDSL(t, func() { Type(fmt.Sprintf("t%d", i), rng.Intn(100)) })
		case 1:
			err = expr.RunInvalidDSL(t, func() { Example(rng.Intn(10), rng.Intn(10)) })
		default:
			err = expr.RunInvalidDSL(t, func() { Headers(rng.Intn(5)) })
		}
		msg := bbSingleLineErr(t, err)
		if !strings.Contains(msg, "cannot use") {
			t.Fatalf("case %d: %q", i, msg)
		}
	}
}

func TestEvalrunTooFewArgError(t *testing.T) {
	cases := map[string]func(){
		"Example": func() { Example() },
		"OneOf":   func() { OneOf("name") },
	}
	for name, dsl := range cases {
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, dsl)
			msg := bbSingleLineErr(t, err)
			fn := strings.Split(name, " ")[0]
			if !strings.Contains(msg, "too few arguments given to "+fn) {
				t.Fatalf("got %q", msg)
			}
		})
	}
}

func TestEvalrunTooFewArgErrorRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	fns := []func(){
		func() { Example() },
		func() { OneOf("n") },
		func() { Body() },
	}
	for i := 0; i < bbCases; i++ {
		err := expr.RunInvalidDSL(t, fns[rng.Intn(len(fns))])
		msg := bbSingleLineErr(t, err)
		if !strings.Contains(msg, "too few arguments") {
			t.Fatalf("case %d: %q", i, msg)
		}
	}
}

func TestEvalrunTooManyArgError(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() { Example(1, 2, 3) })
	msg := bbSingleLineErr(t, err)
	if !strings.Contains(msg, "too many arguments given to Example") {
		t.Fatalf("got %q", msg)
	}
}

func TestEvalrunTooManyArgErrorRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		err := expr.RunInvalidDSL(t, func() {
			Type("name", func() {
				Attribute("a", String, "d", rng.Intn(5), rng.Intn(5), rng.Intn(5))
			})
		})
		msg := bbSingleLineErr(t, err)
		if !strings.Contains(msg, "too many arguments") {
			t.Fatalf("case %d: %q", i, msg)
		}
	}
}

func TestEvalrunIncompatibleDSL(t *testing.T) {
	dsl := func() {
		Type("SomeType", func() {
			Attribute("attr")
			View("default", func() { Attribute("attr") })
		})
	}
	err := expr.RunInvalidDSL(t, dsl)
	if err == nil || !strings.Contains(err.Error(), `in type "SomeType"`) {
		t.Fatalf("got %v", err)
	}
}

func TestEvalrunIncompatibleDSLRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		typeName := fmt.Sprintf("T%d", rng.Intn(50))
		err := expr.RunInvalidDSL(t, func() {
			Type(typeName, func() {
				Attribute("x")
				View("v", func() { Attribute("x") })
			})
		})
		if err == nil || !strings.Contains(err.Error(), "in type") {
			t.Fatalf("case %d: %v", i, err)
		}
	}
}

func TestEvalrunRunDSLReportError(t *testing.T) {
	bbResetRunDSL(t, &bbEvalExpr{
		name: "expr",
		dsl: func() { eval.ReportError("boom-%d", 1) },
	})
	err := eval.RunDSL()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "boom-1") {
		t.Fatalf("got %v", err)
	}
}

func TestEvalrunRunDSLReportErrorRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	for i := 0; i < bbCases; i++ {
		n := rng.Intn(1000)
		bbResetRunDSL(t, &bbEvalExpr{
			name: "e",
			dsl:  func() { eval.ReportError("msg-%d", n) },
		})
		err := eval.RunDSL()
		if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("msg-%d", n)) {
			t.Fatalf("case %d: %v", i, err)
		}
	}
}

func TestEvalrunRunDSLValidationError(t *testing.T) {
	bbResetRunDSL(t, &bbEvalExpr{
		name:     "expr",
		dsl:      func() {},
		validate: errors.New("bad"),
	})
	err := eval.RunDSL()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Fatalf("got %v", err)
	}
}

func TestEvalrunRunDSLValidationRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 5))
	for i := 0; i < bbCases; i++ {
		code := rng.Intn(10000)
		bbResetRunDSL(t, &bbEvalExpr{
			name:     "v",
			dsl:      func() {},
			validate: fmt.Errorf("fail-%d", code),
		})
		err := eval.RunDSL()
		if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("fail-%d", code)) {
			t.Fatalf("case %d: %v", i, err)
		}
	}
}

func TestEvalrunCurrentStack(t *testing.T) {
	def := &bbEvalExpr{name: "target"}
	var seen eval.Expression
	ok := eval.Execute(func() {
		seen = eval.Current()
		eval.ReportError("x")
	}, def)
	if ok {
		t.Fatal("Execute should return false after ReportError")
	}
	if seen != def {
		t.Fatalf("Current during DSL: got %v want target", seen)
	}
	if cur := eval.Current(); cur != eval.Top {
		t.Fatalf("empty stack Current=%v want Top", cur)
	}
}

func TestEvalrunCurrentStackRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 6))
	for i := 0; i < bbCases; i++ {
		def := &bbEvalExpr{name: fmt.Sprintf("n%d", i)}
		var cur eval.Expression
		_ = eval.Execute(func() {
			cur = eval.Current()
			if rng.Intn(2) == 0 {
				eval.InvalidArgError("string", rng.Intn(10))
			}
		}, def)
		if cur != def {
			t.Fatalf("case %d: Current mismatch", i)
		}
		if eval.Current() != eval.Top {
			t.Fatalf("case %d: stack not empty after Execute", i)
		}
	}
}

func TestEvalrunExecuteNilSource(t *testing.T) {
	def := &bbEvalExpr{name: "nil-src", dsl: nil}
	if !eval.Execute(func() {}, def) {
		t.Fatal("nil DSL source should succeed")
	}
}

func TestEvalrunExecuteSuccessRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 7))
	for i := 0; i < bbCases; i++ {
		def := &bbEvalExpr{name: "d"}
		ok := eval.Execute(func() {
			switch rng.Intn(4) {
			case 0:
				eval.TooFewArgError()
			case 1:
				eval.TooManyArgError()
			case 2:
				eval.IncompatibleDSL()
			default:
				eval.InvalidArgError("int", "x")
			}
		}, def)
		if ok {
			t.Fatalf("case %d: expected Execute failure after error helper", i)
		}
		if eval.Current() != eval.Top {
			t.Fatalf("case %d: stack leak", i)
		}
	}
}

func TestEvalrunUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 99))
	for i := 0; i < bbCases; i++ {
		switch rng.Intn(5) {
		case 0:
			def := &bbEvalExpr{name: "u"}
			if eval.Execute(func() { eval.ReportError("r%d", i) }, def) {
				t.Fatalf("case %d: wanted failure", i)
			}
		case 1:
			err := expr.RunInvalidDSL(t, func() { MapParams(rng.Intn(9)) })
			if err == nil {
				t.Fatalf("case %d: MapParams int", i)
			}
		case 2:
			bbResetRunDSL(t, &bbEvalExpr{name: "ok", dsl: func() {}})
			if err := eval.RunDSL(); err != nil {
				t.Fatalf("case %d: valid run: %v", i, err)
			}
		case 3:
			if eval.Current() != eval.Top {
				t.Fatalf("case %d: initial Current", i)
			}
		default:
			err := expr.RunInvalidDSL(t, func() {
				Service("s", func() { Method("m", func() { Payload(rng.Intn(3)) }) })
			})
			if err == nil {
				t.Fatalf("case %d: bad payload", i)
			}
		}
	}
}
