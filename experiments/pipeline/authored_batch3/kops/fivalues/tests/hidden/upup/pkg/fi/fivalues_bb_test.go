// Package fi_test is the hidden black-box suite for fivalues.
// One TestDetailNN per DETAILS.md commitment. Exported API only.
package fi_test

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	fi "example.internal/clustkit/upup/pkg/fi"
)

// Detail 1 (Inferable: yes): ValueOf returns the zero value on nil and
// dereferences otherwise.
func TestDetail01(t *testing.T) {
	if got := fi.ValueOf((*int)(nil)); got != 0 {
		t.Fatalf("ValueOf(nil int) = %v", got)
	}
	x := 42
	if got := fi.ValueOf(&x); got != 42 {
		t.Fatalf("ValueOf(&42) = %v", got)
	}
	if got := fi.ValueOf((*string)(nil)); got != "" {
		t.Fatalf("ValueOf(nil string) = %q", got)
	}
}

// Detail 2 (Inferable: partially): StringSliceValue skips nil elements and
// returns nil (not an empty slice) when nothing survives.
func TestDetail02(t *testing.T) {
	a, b := "a", "b"
	got := fi.StringSliceValue([]*string{&a, nil, &b})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v", got)
	}
	if got := fi.StringSliceValue([]*string{nil, nil}); got != nil {
		t.Fatalf("all-nil input: got %#v, want nil", got)
	}
	if got := fi.StringSliceValue(nil); got != nil {
		t.Fatalf("nil input: got %#v, want nil", got)
	}
	if got := fi.StringSliceValue([]*string{}); got != nil {
		t.Fatalf("empty input: got %#v, want nil", got)
	}
}

// Detail 3 (Inferable: yes): StringSlice returns pointers INTO the input
// slice's elements — mutating a target mutates the input.
func TestDetail03(t *testing.T) {
	s := []string{"x", "y"}
	ps := fi.StringSlice(s)
	if len(ps) != 2 {
		t.Fatalf("len %d", len(ps))
	}
	*ps[0] = "mutated"
	if s[0] != "mutated" {
		t.Fatal("StringSlice did not alias the input slice")
	}
}

// Detail 4 (Inferable: yes): IsNilOrEmpty is true for nil and "".
func TestDetail04(t *testing.T) {
	if !fi.IsNilOrEmpty(nil) {
		t.Fatal("nil not empty")
	}
	e := ""
	if !fi.IsNilOrEmpty(&e) {
		t.Fatal(`"" not empty`)
	}
	s := "x"
	if fi.IsNilOrEmpty(&s) {
		t.Fatal(`"x" reported empty`)
	}
}

// Detail 5 (Inferable: no): DebugPrint sentinels — nil and typed-nil render
// identically; a Stringer renders via String(); anything else via
// fmt.Sprint; a Resource renders its contents (truncated when long).
// Sentinal spellings are not pinned.
type bbStringer struct{ s string }

func (b bbStringer) String() string { return "STR:" + b.s }

type bbResource struct{ s string }

func (r bbResource) Open() (io.Reader, error) { return strings.NewReader(r.s), nil }

func TestDetail05(t *testing.T) {
	plain := fi.DebugPrint(nil)
	var sptr *string
	typedNil := fi.DebugPrint(sptr)
	if plain == "" || typedNil == "" {
		t.Fatal("nil renders empty")
	}
	if plain != typedNil {
		t.Fatalf("nil %q vs typed-nil %q differ", plain, typedNil)
	}
	if got := fi.DebugPrint(bbStringer{"abc"}); got != "STR:abc" {
		t.Fatalf("Stringer: got %q", got)
	}
	if got := fi.DebugPrint(42); got != fmt.Sprint(42) {
		t.Fatalf("fallback: got %q want %q", got, fmt.Sprint(42))
	}
	long := strings.Repeat("z", 400)
	got := fi.DebugPrint(bbResource{long})
	if got == long {
		t.Fatal("long resource content not truncated")
	}
	if len(got) >= len(long) {
		t.Fatalf("rendered resource not shorter than source (%d >= %d)", len(got), len(long))
	}
	if !strings.HasPrefix(got, strings.Repeat("z", 200)[:100]) {
		t.Fatal("rendered resource does not contain content prefix")
	}
}

// Detail 6 (Inferable: partially): marshal failure is returned as a STRING
// describing the failure, not propagated as an error — the functions have no
// error return. Shape: non-empty output that is not valid JSON, mentioning
// failure.
func TestDetail06(t *testing.T) {
	for _, fn := range []func(interface{}) string{fi.DebugAsJsonString, fi.DebugAsJsonStringIndent} {
		out := fn(map[string]int{"k": 1})
		var v interface{}
		if err := json.Unmarshal([]byte(out), &v); err != nil {
			t.Fatalf("success output not valid JSON: %q", out)
		}
		out = fn(make(chan int)) // unmarshalable
		if out == "" {
			t.Fatal("marshal failure produced empty string")
		}
		if err := json.Unmarshal([]byte(out), &v); err == nil {
			t.Fatalf("marshal failure produced valid JSON: %q", out)
		}
		if !strings.Contains(strings.ToLower(out), "error") {
			t.Fatalf("failure string does not mention error: %q", out)
		}
	}
}

// Detail 7 (Inferable: yes): ToInt64/ToString propagate nil and swallow
// conversion errors.
func TestDetail07(t *testing.T) {
	if fi.ToInt64(nil) != nil {
		t.Fatal("ToInt64(nil) non-nil")
	}
	junk := "junk"
	if got := fi.ToInt64(&junk); got != nil {
		t.Fatalf("ToInt64(junk) = %v, want nil", got)
	}
	n := "42"
	if got := fi.ToInt64(&n); got == nil || *got != 42 {
		t.Fatalf("ToInt64(42) = %v", got)
	}
	if fi.ToString(nil) != nil {
		t.Fatal("ToString(nil) non-nil")
	}
	v := int64(7)
	if got := fi.ToString(&v); got == nil || *got != "7" {
		t.Fatalf("ToString(7) = %v", got)
	}
}

// Detail 8 (Inferable: yes): ArrayContains is exact-match.
func TestDetail08(t *testing.T) {
	arr := []string{"alpha", "beta", ""}
	if !fi.ArrayContains(arr, "alpha") {
		t.Fatal("alpha not found")
	}
	if fi.ArrayContains(arr, "alph") {
		t.Fatal("prefix matched")
	}
	if !fi.ArrayContains(arr, "") {
		t.Fatal("empty string not found though present")
	}
	if fi.ArrayContains(nil, "x") {
		t.Fatal("found in nil slice")
	}
}
