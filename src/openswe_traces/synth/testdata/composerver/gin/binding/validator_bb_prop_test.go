// Black-box property suite for the validator unit.
// Exported API only: binding.Validator, binding.SliceValidationError.
// Seed 20260919; >=10k cases; contract + default_validator_test.go coverage.
//
// Coverage table (contract sentence -> property):
//   "nil input returns nil without panic"
//       -> TestBBValidateNil / TestBBValidateNilRandom
//   "non-struct kinds are skipped with nil"
//       -> TestBBValidatePrimitives / TestBBValidatePrimitivesRandom
//   "pointer to non-struct re-validates by kind"
//       -> TestBBValidatePointerToNonStruct
//   "structs and pointers-to-struct reach the engine"
//       -> TestBBValidateStructRules / TestBBValidateStructRandom
//   "slices validated element-wise; all-pass returns nil"
//       -> TestBBValidateSlicePass / TestBBValidateSliceRandomPass
//   "per-element errors aggregate into SliceValidationError with [i]: prefixes"
//       -> TestBBSliceValidationErrorFormat / TestBBValidateSliceFail
//   "Engine() returns live engine for custom validations"
//       -> TestBBValidatorEngineCustom / TestBBValidatorEngineLazyInit
package binding_test

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"

	binding "example.internal/httprouter/binding"
	"github.com/go-playground/validator/v10"
)

// bbSeed and bbCases are in bb_const_test.go.

type bbExampleStruct struct {
	A string `binding:"max=8"`
	B int    `binding:"gt=0"`
}

type bbModifyStruct struct {
	Integer int
}

func bbOracleSliceValidationError(errs []error) string {
	if len(errs) == 0 {
		return ""
	}
	var parts []string
	for i, e := range errs {
		if e != nil {
			parts = append(parts, fmt.Sprintf("[%d]: %s", i, e.Error()))
		}
	}
	return strings.Join(parts, "\n")
}

func TestBBValidateNil(t *testing.T) {
	if err := binding.Validator.ValidateStruct(nil); err != nil {
		t.Fatalf("ValidateStruct(nil)=%v want nil", err)
	}
}

func TestBBValidateNilRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		var nilPtr *bbExampleStruct
		if err := binding.Validator.ValidateStruct(nilPtr); err != nil {
			t.Fatalf("case %d: nil pointer struct: %v", i, err)
		}
		if rng.Intn(2) == 0 {
			if err := binding.Validator.ValidateStruct(nil); err != nil {
				t.Fatalf("case %d: nil: %v", i, err)
			}
		}
	}
}

func TestBBValidatePrimitives(t *testing.T) {
	cases := []any{
		3,
		&[]int{1, 2, 3},
		map[string]int{"a": 1},
		"text",
		[]map[string]any{{"x": 1}},
	}
	for _, c := range cases {
		if err := binding.Validator.ValidateStruct(c); err != nil {
			t.Fatalf("ValidateStruct(%T)=%v want nil", c, err)
		}
	}
}

func TestBBValidatePrimitivesRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		var err error
		switch rng.Intn(5) {
		case 0:
			err = binding.Validator.ValidateStruct(rng.Int())
		case 1:
			s := fmt.Sprintf("s%d", i)
			err = binding.Validator.ValidateStruct(s)
		case 2:
			s := fmt.Sprintf("s%d", i)
			err = binding.Validator.ValidateStruct(&s)
		case 3:
			err = binding.Validator.ValidateStruct(map[string]int{"k": i})
		case 4:
			err = binding.Validator.ValidateStruct([]int{i, i + 1})
		}
		if err != nil {
			t.Fatalf("case %d: primitive validation should be nil: %v", i, err)
		}
	}
}

func TestBBValidatePointerToNonStruct(t *testing.T) {
	n := 42
	if err := binding.Validator.ValidateStruct(&n); err != nil {
		t.Fatalf("pointer to int: %v", err)
	}
	s := "hello"
	if err := binding.Validator.ValidateStruct(&s); err != nil {
		t.Fatalf("pointer to string: %v", err)
	}
}

func TestBBValidateStructRules(t *testing.T) {
	if err := binding.Validator.ValidateStruct(bbExampleStruct{A: "123456789", B: 1}); err == nil {
		t.Fatal("expected validation error for A too long")
	}
	if err := binding.Validator.ValidateStruct(bbExampleStruct{A: "12345678", B: 0}); err == nil {
		t.Fatal("expected validation error for B not gt 0")
	}
	if err := binding.Validator.ValidateStruct(&bbExampleStruct{A: "ok", B: 1}); err != nil {
		t.Fatalf("valid struct: %v", err)
	}
}

func TestBBValidateStructRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	for i := 0; i < bbCases; i++ {
		aLen := rng.Intn(12)
		b := rng.Intn(5) - 1
		s := bbExampleStruct{A: strings.Repeat("x", aLen), B: b}
		err := binding.Validator.ValidateStruct(&s)
		wantErr := aLen > 8 || b <= 0
		if wantErr && err == nil {
			t.Fatalf("case %d: expected error for A len %d B %d", i, aLen, b)
		}
		if !wantErr && err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}
	}
}

func TestBBValidateSlicePass(t *testing.T) {
	slice := []bbExampleStruct{{A: "ok", B: 1}, {A: "fine", B: 2}}
	if err := binding.Validator.ValidateStruct(slice); err != nil {
		t.Fatalf("all-pass slice: %v", err)
	}
}

func TestBBValidateSliceRandomPass(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 5))
	for i := 0; i < bbCases; i++ {
		n := rng.Intn(5) + 1
		slice := make([]bbExampleStruct, n)
		for j := range slice {
			slice[j] = bbExampleStruct{A: "ok", B: rng.Intn(10) + 1}
		}
		if err := binding.Validator.ValidateStruct(slice); err != nil {
			t.Fatalf("case %d: valid slice failed: %v", i, err)
		}
	}
}

func TestBBSliceValidationErrorFormat(t *testing.T) {
	cases := []struct {
		errs binding.SliceValidationError
		want string
	}{
		{binding.SliceValidationError{}, ""},
		{binding.SliceValidationError{errors.New("one")}, "[0]: one"},
		{
			binding.SliceValidationError{errors.New("a"), errors.New("b")},
			"[0]: a\n[1]: b",
		},
		{
			binding.SliceValidationError{errors.New("first"), nil, errors.New("last")},
			"[0]: first\n[2]: last",
		},
	}
	for _, tc := range cases {
		got := tc.errs.Error()
		if got != tc.want {
			t.Fatalf("SliceValidationError.Error()=%q want %q", got, tc.want)
		}
	}
}

func TestBBValidateSliceFail(t *testing.T) {
	slice := []bbExampleStruct{
		{A: "ok", B: 1},
		{A: "toolongvalue", B: 2},
		{A: "ok", B: 0},
	}
	err := binding.Validator.ValidateStruct(slice)
	if err == nil {
		t.Fatal("expected slice validation error")
	}
	sve, ok := err.(binding.SliceValidationError)
	if !ok {
		t.Fatalf("expected SliceValidationError, got %T", err)
	}
	if len(sve) == 0 {
		t.Fatal("expected non-empty SliceValidationError")
	}
	msg := sve.Error()
	if !strings.Contains(msg, "[1]:") {
		t.Fatalf("expected [1]: prefix in %q", msg)
	}
}

func TestBBValidatorEngineCustom(t *testing.T) {
	engine, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok || engine == nil {
		t.Fatal("Engine() must return *validator.Validate")
	}

	type customStruct struct {
		N int `binding:"notone"`
	}
	notOne := func(fl validator.FieldLevel) bool {
		if v, ok := fl.Field().Interface().(int); ok {
			return v != 1
		}
		return false
	}
	if err := engine.RegisterValidation("notone", notOne); err != nil {
		t.Fatalf("RegisterValidation: %v", err)
	}
	if err := binding.Validator.ValidateStruct(customStruct{N: 1}); err == nil {
		t.Fatal("custom validation should fail for N=1")
	}
	if err := binding.Validator.ValidateStruct(customStruct{N: 2}); err != nil {
		t.Fatalf("custom validation should pass for N=2: %v", err)
	}
}

func TestBBValidatorEngineLazyInit(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			eng := binding.Validator.Engine()
			if eng == nil {
				t.Error("Engine() returned nil")
			}
			_ = binding.Validator.ValidateStruct(bbExampleStruct{A: "ok", B: 1})
		}()
	}
	wg.Wait()
}
