package server

import (
	"errors"
	"strings"
	"testing"
)

// TestDetail01: isValidSubject — empty subject and empty tokens fail;
// whitespace in a token fails; '>' only as a whole final token; '*'/'>'
// embedded in a longer token are literal; checkRunes rejects NUL and
// invalid UTF-8.
func TestDetail01(t *testing.T) {
	valid := []string{"foo", "foo.bar", "foo.>", ">", "foo.*", "*", "foo*", "foo**", "foo.*bar", "*.*"}
	for _, s := range valid {
		if !isValidSubject(s, false) {
			t.Fatalf("%q should be valid", s)
		}
	}
	invalid := []string{"", "foo..bar", ".foo", "foo.", "foo.>.bar", ">.>", "foo.>.>", "f oo", "f\too"}
	for _, s := range invalid {
		if isValidSubject(s, false) {
			t.Fatalf("%q should be invalid", s)
		}
	}
	if isValidSubject("a\x00b", true) || isValidSubject("a\xffb", true) {
		t.Fatal("checkRunes must reject NUL and invalid UTF-8")
	}
	if !isValidSubject("a\x00b", false) {
		t.Fatal("without checkRunes, NUL passes the token rules")
	}
}

// TestDetail02: subjectIsLiteral is byte-scanning — a wildcard char is
// non-literal only as a complete token bounded by dots or string edges.
func TestDetail02(t *testing.T) {
	literal := []string{"foo", "foo.bar", "foo*bar", "foo.*bar", "foo.bar*", "foo.bar>", "a.*b.c", "*a", "a*"}
	for _, s := range literal {
		if !subjectIsLiteral(s) {
			t.Fatalf("%q should be literal", s)
		}
	}
	for _, s := range []string{"foo.*", "*", ">", "foo.>", "a.*.b"} {
		if subjectIsLiteral(s) {
			t.Fatalf("%q should be non-literal", s)
		}
	}
}

// TestDetail03: isValidLiteralSubject rejects empty tokens and any
// single-char wildcard token but does not check whitespace.
func TestDetail03(t *testing.T) {
	for _, s := range []string{"foo", "foo.bar", "foo*bar", "a b", "f\tb"} {
		if !isValidLiteralSubject(strings.SplitSeq(s, ".")) {
			t.Fatalf("%q should be literal-valid", s)
		}
		if !IsValidLiteralSubject(s) {
			t.Fatalf("exported %q should be literal-valid", s)
		}
	}
	for _, s := range []string{"foo.*", "*", "foo.>", "foo..bar", ""} {
		if isValidLiteralSubject(strings.SplitSeq(s, ".")) {
			t.Fatalf("%q should not be literal-valid", s)
		}
	}
}

// TestDetail04: tokenAt is 1-based; index 0 and out-of-range return empty.
func TestDetail04(t *testing.T) {
	if got := tokenAt("foo.bar.baz", 0); got != "" {
		t.Fatalf("index 0 = %q", got)
	}
	for i, want := range []string{"foo", "bar", "baz"} {
		if got := tokenAt("foo.bar.baz", uint8(i+1)); got != want {
			t.Fatalf("index %d = %q, want %q", i+1, got, want)
		}
	}
	if got := tokenAt("foo.bar.baz", 4); got != "" {
		t.Fatalf("out of range = %q", got)
	}
	if got := tokenAt("foo.", 2); got != "" {
		t.Fatalf("trailing empty token = %q", got)
	}
	if got := tokenAt(".foo", 1); got != "" {
		t.Fatalf("leading empty token = %q", got)
	}
}

// TestDetail05: ValidateMapping — empty dest ok; dest token rules per
// subject validity reported as a mapping-destination error; {{...}}
// tokens must match a mapping function; the result must build a valid
// transform.
func TestDetail05(t *testing.T) {
	if err := ValidateMapping("foo", ""); err != nil {
		t.Fatalf("empty dest: %v", err)
	}
	if err := ValidateMapping("foo", "bar"); err != nil {
		t.Fatalf("plain dest: %v", err)
	}
	if err := ValidateMapping("foo.*", "bar.{{wildcard(1)}}"); err != nil {
		t.Fatalf("valid function token: %v", err)
	}
	if err := ValidateMapping("foo.*", "bar.{{nope(1)}}"); !errors.Is(err, ErrInvalidMappingDestination) ||
		!strings.Contains(err.Error(), "unknown function") {
		t.Fatalf("unknown function should report as a mapping-destination error naming it: %v", err)
	}
	for _, dest := range []string{"bar..baz", "bar.>.x"} {
		if err := ValidateMapping("foo", dest); !errors.Is(err, ErrInvalidMappingDestination) {
			t.Fatalf("dest %q should be a mapping-destination error: %v", dest, err)
		}
	}
	// Wildcard dest against a literal src passes token rules and fails in
	// the transform stage — assert only that an error is reported.
	if err := ValidateMapping("foo", "bar.>"); err == nil {
		t.Fatal("wildcard dest with literal src must fail")
	}
}

// TestDetail06: SubjectsCollide — equal literal subjects collide;
// one-literal uses subset matching; both-wildcard applies the token-count
// rules (partials-only must be equal length; shorter-without-'>' never
// collides) then pairwise token matching.
func TestDetail06(t *testing.T) {
	collide := [][2]string{
		{"a.b", "a.b"}, {"a.*", "a.b"}, {"a.>", "a.b.c"}, {"a.*", "a.>"},
		{"a.>", "a.b.>"}, {"a.*.c", "a.b.*"}, {">", "a.b"}, {"a.*.*", "a.b.>"},
		{"a.*.d", "a.>"}, {"a.b.*", "a.*.c"},
	}
	for _, p := range collide {
		if !SubjectsCollide(p[0], p[1]) {
			t.Fatalf("%q and %q should collide", p[0], p[1])
		}
	}
	nocollide := [][2]string{
		{"a.b", "a.c"}, {"a.*", "a.*.b"}, {"*.a", "*.b"}, {"a.*", "b.*"},
		{"a.*", "a.*.d"},
	}
	for _, p := range nocollide {
		if SubjectsCollide(p[0], p[1]) {
			t.Fatalf("%q and %q should not collide", p[0], p[1])
		}
	}
}

// TestDetail07: tokensCanMatch — empty is false; a leading wildcard char
// is true (position-only); else equality.
func TestDetail07(t *testing.T) {
	for _, p := range [][2]string{{"", "a"}, {"a", ""}, {"foo*", "bar"}, {"a", "b"}} {
		if tokensCanMatch(p[0], p[1]) {
			t.Fatalf("%q/%q should not match", p[0], p[1])
		}
	}
	for _, p := range [][2]string{{"*", "a"}, {">", "a"}, {"*foo", "bar"}, {">x", "y"}, {"a", "a"}} {
		if !tokensCanMatch(p[0], p[1]) {
			t.Fatalf("%q/%q should match", p[0], p[1])
		}
	}
}

// TestDetail08: numTokens counts separators+1 (empty → 0).
func TestDetail08(t *testing.T) {
	for s, want := range map[string]int{
		"": 0, "foo": 1, "foo.bar": 2, "foo..bar": 3, ".": 2, "a.b.c.d": 4,
	} {
		if got := numTokens(s); got != want {
			t.Fatalf("numTokens(%q) = %d, want %d", s, got, want)
		}
	}
}
