package config

import "testing"

// TestDetail01 (yes): ApplyInsteadOf rewrites using only that one URL's
// InsteadOfs.
func TestDetail01(t *testing.T) {
	u := &URL{Name: "https://new/", InsteadOfs: []string{"https://old/"}}
	if got := u.ApplyInsteadOf("https://old/repo"); got != "https://new/repo" {
		t.Fatalf("ApplyInsteadOf = %q", got)
	}
	// A rule living on a different URL value does not leak in.
	u2 := &URL{Name: "https://other/", InsteadOfs: []string{"git@x:"}}
	if got := u2.ApplyInsteadOf("https://old/repo"); got != "https://old/repo" {
		t.Fatalf("foreign rule rewrote: %q", got)
	}
}

// TestDetail02 (yes): a candidate matches when the remote URL has that
// insteadOf string as a prefix.
func TestDetail02(t *testing.T) {
	u := &URL{Name: "n:", InsteadOfs: []string{"a:b"}}
	if got := u.ApplyInsteadOf("a:b/c"); got != "n:/c" {
		t.Fatalf("prefix match = %q", got)
	}
	if got := u.ApplyInsteadOf("xa:b"); got != "xa:b" {
		t.Fatalf("non-prefix rewritten: %q", got)
	}
}

// TestDetail03 (doc): if several prefixes match, the longest prefix wins.
func TestDetail03(t *testing.T) {
	u := &URL{Name: "N/", InsteadOfs: []string{"a:", "a:b:"}}
	if got := u.ApplyInsteadOf("a:b:c"); got != "N/c" {
		t.Fatalf("longest prefix = %q, want %q", got, "N/c")
	}
	u = &URL{Name: "N/", InsteadOfs: []string{"a:b:", "a:"}} // reversed input order
	if got := u.ApplyInsteadOf("a:b:c"); got != "N/c" {
		t.Fatalf("reversed order: longest prefix = %q", got)
	}
}

// TestDetail04 (shape — Inferable: no): equal-length matches keep the
// first in slice order — exercised across two URL values, where the
// tie-break is observable.
func TestDetail04(t *testing.T) {
	urls := []*URL{
		{Name: "first/", InsteadOfs: []string{"a:b:"}},
		{Name: "second/", InsteadOfs: []string{"a:b:"}},
	}
	got, matched := applyLongestInsteadOfMatch("a:b:c", urls)
	if !matched || got != "first/c" {
		t.Fatalf("equal-length tie = %q matched=%v, want first/c", got, matched)
	}
}

// TestDetail05 (partially): the rewrite is Name + remoteURL[len(prefix):].
func TestDetail05(t *testing.T) {
	u := &URL{Name: "https://mirror/", InsteadOfs: []string{"https://host/"}}
	if got := u.ApplyInsteadOf("https://host/org/repo.git"); got != "https://mirror/org/repo.git" {
		t.Fatalf("rewrite = %q", got)
	}
}

// TestDetail06 (yes): no match returns the original URL and matched=false
// from the helper.
func TestDetail06(t *testing.T) {
	urls := []*URL{{Name: "n:", InsteadOfs: []string{"a:"}}}
	got, matched := applyLongestInsteadOfMatch("z:z", urls)
	if matched || got != "z:z" {
		t.Fatalf("no-match = %q matched=%v", got, matched)
	}
	u := &URL{Name: "n:", InsteadOfs: []string{"a:"}}
	if got := u.ApplyInsteadOf("z:z"); got != "z:z" {
		t.Fatalf("ApplyInsteadOf rewrote a non-match: %q", got)
	}
}

// TestDetail07 (shape — Inferable: no): across several *URL values the
// longest prefix still wins even when it belongs to a later URL.
func TestDetail07(t *testing.T) {
	urls := []*URL{
		{Name: "short/", InsteadOfs: []string{"a:"}},
		{Name: "long/", InsteadOfs: []string{"a:b:"}},
	}
	got, matched := applyLongestInsteadOfMatch("a:b:c", urls)
	if !matched || got != "long/c" {
		t.Fatalf("cross-URL longest = %q matched=%v, want long/c", got, matched)
	}
}

// TestDetail08 (shape — Inferable: no): a match of length 0 is treated as
// no match.
func TestDetail08(t *testing.T) {
	urls := []*URL{{Name: "X/", InsteadOfs: []string{""}}}
	got, matched := applyLongestInsteadOfMatch("a:b", urls)
	if matched || got != "a:b" {
		t.Fatalf("empty insteadOf: got=%q matched=%v, want unchanged", got, matched)
	}
	u := &URL{Name: "X/", InsteadOfs: []string{""}}
	if g := u.ApplyInsteadOf("a:b"); g != "a:b" {
		t.Fatalf("empty insteadOf rewrote to %q", g)
	}
}
