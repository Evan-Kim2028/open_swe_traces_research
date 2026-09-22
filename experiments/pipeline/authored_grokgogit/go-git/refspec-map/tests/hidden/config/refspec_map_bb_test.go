package config

import (
	"errors"
	"testing"

	"example.internal/gitkit/v6/plumbing"
)

// TestDetail01 (partially): Validate requires exactly one ":" and a
// non-empty destination after it; otherwise ErrRefSpecMalformedSeparator.
func TestDetail01(t *testing.T) {
	if err := RefSpec("refs/heads/a:refs/heads/b").Validate(); err != nil {
		t.Fatalf("plain spec: %v", err)
	}
	if err := RefSpec("+refs/heads/a:refs/heads/b").Validate(); err != nil {
		t.Fatalf("force spec: %v", err)
	}
	if err := RefSpec(":refs/heads/b").Validate(); err != nil {
		t.Fatalf("delete spec: %v", err)
	}
	for _, bad := range []string{"refs/heads/a", "", "a:b:c", "refs/heads/a:"} {
		err := RefSpec(bad).Validate()
		if !errors.Is(err, ErrRefSpecMalformedSeparator) {
			t.Fatalf("Validate(%q) = %v, want ErrRefSpecMalformedSeparator", bad, err)
		}
	}
}

// TestDetail02 (shape — Inferable: no): source and destination must carry
// the same number of "*" and that number is 0 or 1; otherwise the
// wildcard sentinel.
func TestDetail02(t *testing.T) {
	for _, good := range []string{"a:b", "a*:b*"} {
		if err := RefSpec(good).Validate(); err != nil {
			t.Fatalf("Validate(%q) = %v, want nil", good, err)
		}
	}
	for _, bad := range []string{"a*:b", "a:b*", "a:b*c*d", "*a*b:c"} {
		err := RefSpec(bad).Validate()
		if !errors.Is(err, ErrRefSpecMalformedWildcard) {
			t.Fatalf("Validate(%q) = %v, want ErrRefSpecMalformedWildcard", bad, err)
		}
	}
}

// TestDetail03 (yes): a leading "+" is a force flag and not part of the
// source.
func TestDetail03(t *testing.T) {
	s := RefSpec("+refs/heads/a:refs/heads/b")
	if !s.IsForceUpdate() {
		t.Fatal("IsForceUpdate = false for + spec")
	}
	if got := s.Src(); got != "refs/heads/a" {
		t.Fatalf("Src = %q, want refs/heads/a (no '+')", got)
	}
	if RefSpec("refs/heads/a:refs/heads/b").IsForceUpdate() {
		t.Fatal("IsForceUpdate = true for plain spec")
	}
}

// TestDetail04 (yes): a spec whose first character is ":" is a delete
// (empty source).
func TestDetail04(t *testing.T) {
	s := RefSpec(":refs/heads/x")
	if !s.IsDelete() {
		t.Fatal("IsDelete = false for :dst spec")
	}
	if got := s.Src(); got != "" {
		t.Fatalf("delete Src = %q, want empty", got)
	}
	if RefSpec("a:b").IsDelete() {
		t.Fatal("IsDelete = true for a:b")
	}
}

// TestDetail05 (yes): non-wildcard Match is exact equality of Src() and
// the reference name.
func TestDetail05(t *testing.T) {
	s := RefSpec("refs/heads/a:refs/heads/b")
	if !s.Match(plumbing.ReferenceName("refs/heads/a")) {
		t.Fatal("exact match failed")
	}
	for _, n := range []string{"refs/heads/ab", "refs/heads/b", "refs/heads", "xrefs/heads/a"} {
		if s.Match(plumbing.ReferenceName(n)) {
			t.Fatalf("matched %q", n)
		}
	}
	if !RefSpec("+refs/heads/a:refs/heads/b").Match(plumbing.ReferenceName("refs/heads/a")) {
		t.Fatal("forced spec did not match its source")
	}
}

// TestDetail06 (partially): wildcard Match splits Src() on the first "*"
// into prefix+suffix; the name must be at least prefix+suffix long, start
// with prefix, and end with suffix.
func TestDetail06(t *testing.T) {
	s := RefSpec("refs/heads/*:refs/remotes/o/*")
	for _, n := range []string{"refs/heads/x", "refs/heads/", "refs/heads/a/b"} {
		if !s.Match(plumbing.ReferenceName(n)) {
			t.Fatalf("did not match %q", n)
		}
	}
	for _, n := range []string{"refs/other/x", "refs/head", "refs/headsx"} {
		if s.Match(plumbing.ReferenceName(n)) {
			t.Fatalf("matched %q", n)
		}
	}
	// Suffix side: "refs/heads/*bc" — prefix "refs/heads/", suffix "bc".
	s2 := RefSpec("refs/heads/*bc:refs/tags/*")
	if !s2.Match(plumbing.ReferenceName("refs/heads/abc")) {
		t.Fatal("did not match refs/heads/abc")
	}
	if !s2.Match(plumbing.ReferenceName("refs/heads/bc")) {
		t.Fatal("empty star-slice should still match")
	}
	if s2.Match(plumbing.ReferenceName("refs/heads/ab")) {
		t.Fatal("matched a name not ending in suffix")
	}
}

// TestDetail07 (yes): non-wildcard Dst returns the text after ":".
func TestDetail07(t *testing.T) {
	if got := RefSpec("refs/heads/a:refs/heads/b").Dst(plumbing.ReferenceName("refs/heads/a")); got != "refs/heads/b" {
		t.Fatalf("Dst = %q", got)
	}
	if got := RefSpec("+a:b").Dst(plumbing.ReferenceName("a")); got != "b" {
		t.Fatalf("forced Dst = %q, want b", got)
	}
}

// TestDetail08 (shape — Inferable: no): wildcard Dst copies the slice of
// the name that sat under the source "*" into the destination "*".
func TestDetail08(t *testing.T) {
	// The committed example: refs/heads/*bc vs refs/heads/abc -> "a".
	got := RefSpec("refs/heads/*bc:refs/tags/*").Dst(plumbing.ReferenceName("refs/heads/abc"))
	if got != "refs/tags/a" {
		t.Fatalf("Dst = %q, want refs/tags/a", got)
	}
	got = RefSpec("refs/heads/*:refs/remotes/o/*").Dst(plumbing.ReferenceName("refs/heads/feat"))
	if got != "refs/remotes/o/feat" {
		t.Fatalf("Dst = %q, want refs/remotes/o/feat", got)
	}
}

// TestDetail09 (yes): MatchAny is true if any spec in the list matches.
func TestDetail09(t *testing.T) {
	l := []RefSpec{"x:y", "refs/heads/*:refs/r/*"}
	if !MatchAny(l, plumbing.ReferenceName("refs/heads/z")) {
		t.Fatal("MatchAny = false with a matching member")
	}
	if MatchAny(l, plumbing.ReferenceName("refs/tags/z")) {
		t.Fatal("MatchAny = true with no matching member")
	}
	if MatchAny(nil, plumbing.ReferenceName("x")) {
		t.Fatal("MatchAny(nil) = true")
	}
}

// TestDetail10 (doc): malformed specs surface the two package sentinels.
func TestDetail10(t *testing.T) {
	if err := RefSpec("nocolon").Validate(); !errors.Is(err, ErrRefSpecMalformedSeparator) {
		t.Fatalf("separator sentinel = %v", err)
	}
	if err := RefSpec("a*:b").Validate(); !errors.Is(err, ErrRefSpecMalformedWildcard) {
		t.Fatalf("wildcard sentinel = %v", err)
	}
	if errors.Is(RefSpec("nocolon").Validate(), ErrRefSpecMalformedWildcard) {
		t.Fatal("separator error aliased the wildcard sentinel")
	}
}
