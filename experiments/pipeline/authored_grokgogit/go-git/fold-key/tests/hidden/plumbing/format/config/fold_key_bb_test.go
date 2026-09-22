package config

import (
	"strings"
	"testing"
)

// TestDetail01 (doc): foldKey(a) == foldKey(b) iff strings.EqualFold(a, b).
func TestDetail01(t *testing.T) {
	pairs := [][2]string{
		{"Core", "core"},
		{"CORE", "core"},
		{"A", "a"},
		{"Sub.Section", "sub.section"},
		{"x", "x"},
	}
	for _, p := range pairs {
		if !strings.EqualFold(p[0], p[1]) {
			t.Fatalf("test bug: %q and %q not EqualFold", p[0], p[1])
		}
		if foldKey(p[0]) != foldKey(p[1]) {
			t.Fatalf("foldKey(%q) = %q != foldKey(%q) = %q", p[0], foldKey(p[0]), p[1], foldKey(p[1]))
		}
	}

	nonPairs := [][2]string{
		{"core", "corg"},
		{"ab", "ba"},
		{"s", "ss"},
		{"a", ""},
	}
	for _, p := range nonPairs {
		if strings.EqualFold(p[0], p[1]) {
			t.Fatalf("test bug: %q and %q are EqualFold", p[0], p[1])
		}
		if foldKey(p[0]) == foldKey(p[1]) {
			t.Fatalf("foldKey(%q) == foldKey(%q) but not EqualFold", p[0], p[1])
		}
	}
}

// TestDetail02 (doc): an all-ASCII name with no uppercase letters is
// returned unchanged and must not allocate.
func TestDetail02(t *testing.T) {
	for _, name := range []string{"core", "core.filemode", "a.b-c_1", ""} {
		if got := foldKey(name); got != name {
			t.Fatalf("foldKey(%q) = %q, want unchanged", name, got)
		}
	}
	if allocs := testing.AllocsPerRun(100, func() { _ = foldKey("core.filemode") }); allocs != 0 {
		t.Fatalf("foldKey of lowercase ASCII allocated %v times", allocs)
	}
}

// TestDetail03 (shape — Inferable: no): each rune maps to the smallest
// rune of its unicode.SimpleFold orbit, lowercased when that smallest rune
// is an ASCII letter. Asserted through the committed consequences on known
// orbits.
func TestDetail03(t *testing.T) {
	for _, c := range []struct {
		in   rune
		want rune
	}{
		{'a', 'a'},
		{'A', 'a'},
		{'Z', 'z'},
		{'5', '5'},
		{'.', '.'},
		{'\u017f', 's'}, // long s: orbit {ſ,S,s} min 'S' -> 's'
		{'\u212a', 'k'}, // kelvin: orbit {K,k,K} min 'K' -> 'k'
	} {
		if got := foldRune(c.in); got != c.want {
			t.Fatalf("foldRune(%U) = %U, want %U", c.in, got, c.want)
		}
	}
}

// TestDetail04 (shape — Inferable: no): ASCII uppercase folds to lowercase.
// (The +('a'-'A') mechanism and an orbit walk agree on outputs; only the
// observable consequence is asserted.)
func TestDetail04(t *testing.T) {
	for r := 'A'; r <= 'Z'; r++ {
		if got := foldRune(r); got != r+('a'-'A') {
			t.Fatalf("foldRune(%c) = %c, want %c", r, got, r+('a'-'A'))
		}
	}
	if got := foldKey("CORE"); got != "core" {
		t.Fatalf("foldKey(CORE) = %q", got)
	}
}

// TestDetail05 (doc): foldKey("Core") == foldKey("core") == "core".
func TestDetail05(t *testing.T) {
	if foldKey("Core") != "core" || foldKey("core") != "core" {
		t.Fatalf("foldKey(Core)=%q foldKey(core)=%q", foldKey("Core"), foldKey("core"))
	}
	if foldKey("Core") == "CORE" {
		t.Fatal("foldKey returned uppercase")
	}
}

// TestDetail06 (shape — Inferable: no): U+017F (long s) shares a key with
// "s" and "S".
func TestDetail06(t *testing.T) {
	if foldKey("\u017f") != foldKey("s") || foldKey("\u017f") != foldKey("S") {
		t.Fatalf("long-s does not share a key with s/S: %q vs %q vs %q",
			foldKey("\u017f"), foldKey("s"), foldKey("S"))
	}
	if strings.ToLower("\u017f") == "s" {
		t.Skip("stdlib changed; the distinguishing property is gone")
	}
}

// TestDetail07 (shape — Inferable: no): U+212A (kelvin sign) shares a key
// with "k" and "K".
func TestDetail07(t *testing.T) {
	if foldKey("\u212a") != foldKey("k") || foldKey("\u212a") != foldKey("K") {
		t.Fatalf("kelvin does not share a key with k/K: %q vs %q vs %q",
			foldKey("\u212a"), foldKey("k"), foldKey("K"))
	}
}

// TestDetail08 (shape — Inferable: no): invalid UTF-8 decodes the way
// strings.EqualFold decodes it (U+FFFD per bad byte), so names that
// EqualFold considers equal still share a key.
func TestDetail08(t *testing.T) {
	// "\xff" and "\xfe" both decode to a single U+FFFD under EqualFold.
	if !strings.EqualFold("\xff", "\xfe") {
		t.Skip("stdlib no longer EquaFolds single invalid bytes")
	}
	if foldKey("\xff") != foldKey("\xfe") {
		t.Fatal("invalid single bytes do not share a key")
	}
	if !strings.EqualFold("a\xffb", "a\xfeb") {
		t.Skip("stdlib no longer EqualFolds embedded invalid bytes")
	}
	if foldKey("a\xffb") != foldKey("a\xfeb") {
		t.Fatal("names with embedded invalid bytes do not share a key")
	}
	if foldKey("\xff\xff") == foldKey("\xff") {
		t.Fatal("two bad bytes collide with one")
	}
}
