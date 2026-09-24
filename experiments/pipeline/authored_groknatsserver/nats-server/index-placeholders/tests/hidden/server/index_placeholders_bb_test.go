package server

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// iphErr checks the shape of a mapping error: non-nil, a
// *mappingDestinationErr wrapping the original token, matching
// ErrInvalidMappingDestination, and carrying the given inner sentinel.
func iphErr(t *testing.T, err error, token string, inner error) {
	t.Helper()
	if err == nil {
		t.Fatalf("token %q: expected error, got nil", token)
	}
	var mde *mappingDestinationErr
	if !errors.As(err, &mde) {
		t.Fatalf("token %q: error %T is not a mapping-destination error", token, err)
	}
	if mde.token != token {
		t.Fatalf("token %q: error wraps %q, not the original token", token, mde.token)
	}
	if mde.err != inner {
		t.Fatalf("token %q: inner sentinel = %v, want %v", token, mde.err, inner)
	}
	if !errors.Is(err, ErrInvalidMappingDestination) {
		t.Fatalf("token %q: error does not match the invalid-mapping-destination sentinel", token)
	}
	if !strings.Contains(err.Error(), token) {
		t.Fatalf("token %q: error text %q does not name the offending token", token, err.Error())
	}
}

func iphOK(t *testing.T, tok string, kind int16, idxs []int, scalar int32, str string) {
	t.Helper()
	k, ii, s, a, err := indexPlaceHolders(tok)
	if err != nil {
		t.Fatalf("indexPlaceHolders(%q) errored: %v", tok, err)
	}
	if k != kind {
		t.Fatalf("indexPlaceHolders(%q) kind = %d, want %d", tok, k, kind)
	}
	if !reflect.DeepEqual(ii, idxs) {
		t.Fatalf("indexPlaceHolders(%q) indexes = %v, want %v", tok, ii, idxs)
	}
	if s != scalar {
		t.Fatalf("indexPlaceHolders(%q) scalar = %d, want %d", tok, s, scalar)
	}
	if a != str {
		t.Fatalf("indexPlaceHolders(%q) str = %q, want %q", tok, a, str)
	}
}

// TestDetail01 (partially): a token of length 0 or 1 is NoTransform with
// indexes []int{-1}, scalar -1, empty string, nil error.
func TestDetail01(t *testing.T) {
	for _, tok := range []string{"", "a", "$"} {
		k, ii, s, a, err := indexPlaceHolders(tok)
		if err != nil {
			t.Fatalf("indexPlaceHolders(%q) errored: %v", tok, err)
		}
		if k != NoTransform {
			t.Fatalf("indexPlaceHolders(%q) kind = %d, want NoTransform", tok, k)
		}
		if !reflect.DeepEqual(ii, []int{-1}) {
			t.Fatalf("indexPlaceHolders(%q) indexes = %v, want [-1]", tok, ii)
		}
		if s != -1 || a != _EMPTY_ {
			t.Fatalf("indexPlaceHolders(%q) = scalar %d, str %q; want -1, \"\"", tok, s, a)
		}
	}
}

// TestDetail02 (yes): "$" plus a base-10 int is Wildcard with that index.
func TestDetail02(t *testing.T) {
	iphOK(t, "$2", Wildcard, []int{2}, -1, _EMPTY_)
	iphOK(t, "$10", Wildcard, []int{10}, -1, _EMPTY_)
}

// TestDetail03 (doc): "$" plus a non-int remainder is NoTransform with the
// [-1]/-1/"" tuple — not a parse error.
func TestDetail03(t *testing.T) {
	for _, tok := range []string{"$x", "$1.5", "$x1", "$a2"} {
		k, ii, s, a, err := indexPlaceHolders(tok)
		if err != nil {
			t.Fatalf("indexPlaceHolders(%q) errored: %v", tok, err)
		}
		if k != NoTransform || !reflect.DeepEqual(ii, []int{-1}) || s != -1 || a != _EMPTY_ {
			t.Fatalf("indexPlaceHolders(%q) = (%d, %v, %d, %q), want NoTransform tuple", tok, k, ii, s, a)
		}
	}
}

// TestDetail04 (yes): the mustache form is recognized only when length > 4
// and the token starts with "{{" and ends with "}}".
func TestDetail04(t *testing.T) {
	// "{{}}" is len 4 — not a mustache token, so it is not dispatched.
	k, _, _, _, err := indexPlaceHolders("{{}}")
	if err != nil || k != NoTransform {
		t.Fatalf("indexPlaceHolders({{}}) = kind %d err %v, want NoTransform/nil", k, err)
	}
	// Missing close or open never reaches the mustache dispatch.
	for _, tok := range []string{"{{abc", "ab}}", "{x}", "{{a}"} {
		k, _, _, _, err := indexPlaceHolders(tok)
		if err != nil || k != NoTransform {
			t.Fatalf("indexPlaceHolders(%q) = kind %d err %v, want NoTransform/nil", tok, k, err)
		}
	}
	// len > 4 with both markers does reach it (unknown name -> error).
	if _, _, _, _, err := indexPlaceHolders("{{q}}"); err == nil {
		t.Fatal("indexPlaceHolders({{q}}) did not error")
	}
}

// TestDetail05 (yes): wildcard with a single empty argument is BadTransform
// and a not-enough-args mapping error wrapping the original token.
func TestDetail05(t *testing.T) {
	k, _, _, _, err := indexPlaceHolders("{{wildcard()}}")
	if k != BadTransform {
		t.Fatalf("kind = %d, want BadTransform", k)
	}
	iphErr(t, err, "{{wildcard()}}", ErrMappingDestinationNotEnoughArgs)
}

// TestDetail06 (yes): wildcard with exactly one integer argument is Wildcard
// with that index, scalar -1.
func TestDetail06(t *testing.T) {
	iphOK(t, "{{wildcard(3)}}", Wildcard, []int{3}, -1, _EMPTY_)
}

// TestDetail07 (yes): wildcard with more than one argument is BadTransform
// and a too-many-args mapping error.
func TestDetail07(t *testing.T) {
	k, _, _, _, err := indexPlaceHolders("{{wildcard(1,2)}}")
	if k != BadTransform {
		t.Fatalf("kind = %d, want BadTransform", k)
	}
	iphErr(t, err, "{{wildcard(1,2)}}", ErrMappingDestinationTooManyArgs)
}

// TestDetail08 (doc): partition with a single int32-fitting integer is
// Partition, empty index slice, that integer as the scalar.
func TestDetail08(t *testing.T) {
	k, ii, s, a, err := indexPlaceHolders("{{partition(10)}}")
	if err != nil {
		t.Fatalf("partition(10) errored: %v", err)
	}
	if k != Partition {
		t.Fatalf("kind = %d, want Partition", k)
	}
	if len(ii) != 0 {
		t.Fatalf("indexes = %v, want empty", ii)
	}
	if s != 10 || a != _EMPTY_ {
		t.Fatalf("scalar = %d, str = %q; want 10, \"\"", s, a)
	}
}

// TestDetail09 (doc): partition with first argument N and following integer
// token indexes is Partition with those indexes in order and scalar N.
func TestDetail09(t *testing.T) {
	iphOK(t, "{{partition(10,1,2)}}", Partition, []int{1, 2}, 10, _EMPTY_)
}

// TestDetail10 (yes): a partition or random integer larger than MaxInt32 is
// BadTransform and an invalid-arg mapping error.
func TestDetail10(t *testing.T) {
	for _, tok := range []string{"{{partition(2147483648)}}", "{{random(2147483648)}}"} {
		k, _, _, _, err := indexPlaceHolders(tok)
		if k != BadTransform {
			t.Fatalf("%s kind = %d, want BadTransform", tok, k)
		}
		iphErr(t, err, tok, ErrMappingDestinationInvalidArg)
	}
}

// TestDetail11 (yes): the six index+int mapping functions dispatch through
// the shared int-args helper with the matching kind constant.
func TestDetail11(t *testing.T) {
	for _, c := range []struct {
		name string
		kind int16
	}{
		{"splitfromleft", SplitFromLeft},
		{"splitfromright", SplitFromRight},
		{"slicefromleft", SliceFromLeft},
		{"slicefromright", SliceFromRight},
		{"left", Left},
		{"right", Right},
	} {
		tok := "{{" + c.name + "(3,2)}}"
		iphOK(t, tok, c.kind, []int{3}, 2, _EMPTY_)
		// And the helper's own arity rules apply: one arg -> not enough.
		bad := "{{" + c.name + "(3)}}"
		k, _, _, _, err := indexPlaceHolders(bad)
		if k != BadTransform {
			t.Fatalf("%s kind = %d, want BadTransform", bad, k)
		}
		iphErr(t, err, bad, ErrMappingDestinationNotEnoughArgs)
	}
}

// TestDetail12 (yes): split requires exactly two arguments — an integer token
// index then the delimiter string.
func TestDetail12(t *testing.T) {
	k, ii, _, a, err := indexPlaceHolders("{{split(1,-)}}")
	if err != nil {
		t.Fatalf("split(1,-) errored: %v", err)
	}
	if k != Split {
		t.Fatalf("kind = %d, want Split", k)
	}
	if !reflect.DeepEqual(ii, []int{1}) {
		t.Fatalf("indexes = %v, want [1]", ii)
	}
	if a != "-" {
		t.Fatalf("string arg = %q, want %q", a, "-")
	}
	k, _, _, _, err = indexPlaceHolders("{{split(1)}}")
	if k != BadTransform || err == nil {
		t.Fatalf("split(1) = kind %d err %v, want BadTransform+error", k, err)
	}
}

// TestDetail13 (shape — Inferable: no): a split delimiter containing a space
// or the subject token separator is BadTransform with an invalid-arg error.
func TestDetail13(t *testing.T) {
	for _, tok := range []string{"{{split(1,- -)}}", "{{split(1,.)}}"} {
		k, _, _, _, err := indexPlaceHolders(tok)
		if k != BadTransform {
			t.Fatalf("%s kind = %d, want BadTransform", tok, k)
		}
		iphErr(t, err, tok, ErrMappingDestinationInvalidArg)
	}
}

// TestDetail14 (partially): random requires exactly one integer argument; any
// other arity is BadTransform with a mapping error — not-enough-args for the
// empty case, still a mapping-destination error for extra args.
func TestDetail14(t *testing.T) {
	k7, ii7, s7, a7, err7 := indexPlaceHolders("{{random(7)}}")
	if err7 != nil || k7 != Random || s7 != 7 || a7 != _EMPTY_ {
		t.Fatalf("{{random(7)}} = (%d, %v, %d, %q, %v), want Random+7", k7, ii7, s7, a7, err7)
	}
	if len(ii7) != 0 {
		t.Fatalf("{{random(7)}} indexes = %v, want empty", ii7)
	}

	k, _, _, _, err := indexPlaceHolders("{{random()}}")
	if k != BadTransform {
		t.Fatalf("random() kind = %d, want BadTransform", k)
	}
	// Arity errors surface as a mapping-destination error naming the token;
	// which inner sentinel carries "wrong arity" is not derivable.
	var mde0 *mappingDestinationErr
	if err == nil || !errors.As(err, &mde0) || mde0.token != "{{random()}}" ||
		!errors.Is(err, ErrInvalidMappingDestination) {
		t.Fatalf("random() error %v is not a mapping-destination error for the token", err)
	}

	k, _, _, _, err = indexPlaceHolders("{{random(1,2)}}")
	if k != BadTransform || err == nil {
		t.Fatalf("random(1,2) = kind %d err %v, want BadTransform+error", k, err)
	}
	var mde *mappingDestinationErr
	if !errors.As(err, &mde) || !errors.Is(err, ErrInvalidMappingDestination) {
		t.Fatalf("random(1,2) error %v is not a mapping-destination error", err)
	}
}

// TestDetail15 (yes): a well-formed {{...}} token matching no known function
// is BadTransform and an unknown-function mapping error wrapping the token.
func TestDetail15(t *testing.T) {
	k, _, _, _, err := indexPlaceHolders("{{nosuchfn(1)}}")
	if k != BadTransform {
		t.Fatalf("kind = %d, want BadTransform", k)
	}
	iphErr(t, err, "{{nosuchfn(1)}}", ErrUnknownMappingDestinationFunction)
}

// TestDetail16 (yes): function names match as the remaining regexes encode —
// first letter of either case, internal whitespace allowed.
func TestDetail16(t *testing.T) {
	iphOK(t, "{{Wildcard(3)}}", Wildcard, []int{3}, -1, _EMPTY_)
	iphOK(t, "{{ wildcard(3)}}", Wildcard, []int{3}, -1, _EMPTY_)
	iphOK(t, "{{wildcard( 3 )}}", Wildcard, []int{3}, -1, _EMPTY_)
	// Only the first letter is case-folded by the regexes: an all-caps name
	// matches no function and falls to the unknown-function error.
	k, _, _, _, err := indexPlaceHolders("{{WILDCARD(3)}}")
	if err == nil || k != BadTransform {
		t.Fatalf("{{WILDCARD(3)}} = kind %d err %v, want BadTransform+error", k, err)
	}
}

// TestDetail17 (yes): integer arguments are parsed after trimming spaces.
func TestDetail17(t *testing.T) {
	iphOK(t, "{{partition( 10 , 1 , 2 )}}", Partition, []int{1, 2}, 10, _EMPTY_)
}

// TestDetail18 (yes): on any successful non-split mapping the string
// argument is empty.
func TestDetail18(t *testing.T) {
	for _, tok := range []string{"$2", "{{wildcard(3)}}", "{{partition(4)}}", "{{random(7)}}", "{{left(1,2)}}"} {
		_, _, _, a, err := indexPlaceHolders(tok)
		if err != nil {
			t.Fatalf("indexPlaceHolders(%q) errored: %v", tok, err)
		}
		if a != _EMPTY_ {
			t.Fatalf("indexPlaceHolders(%q) string arg = %q, want empty", tok, a)
		}
	}
}
