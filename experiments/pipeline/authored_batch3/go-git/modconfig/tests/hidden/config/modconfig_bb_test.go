package config

import (
	"errors"
	"strings"
	"testing"
)

// TestDetail01: submodule names reject empty, ".", NUL, leading/trailing
// separators, drive-letter prefixes, and components resolving to ".." under
// either canonicalisation.
func TestDetail01(t *testing.T) {
	for _, bad := range []string{
		"", ".", "a\x00b", "/lead", "trail/", "\\lead", "trail\\", "x:foo", "x:y",
		"a/../b", "a\\..\\b", "a/.. /b", "a/.‌./b",
	} {
		if err := validSubmoduleName(bad); err == nil {
			t.Fatalf("validSubmoduleName(%q) = nil", bad)
		}
	}
	for _, ok := range []string{"a/b/c", "a/b..c", "sub"} {
		if err := validSubmoduleName(ok); err != nil {
			t.Fatalf("validSubmoduleName(%q) = %v", ok, err)
		}
	}
}

// TestDetail02: entries whose NAME or PATH fail validation never reach the
// map; empty path/URL do NOT skip — the submodule still lands in the map.
func TestDetail02(t *testing.T) {
	m := NewModules()
	err := m.Unmarshal([]byte(`
[submodule "a/../b"]
	path = x
	url = u
[submodule "good"]
	path = p
	url = u
`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Submodules["a/../b"]; ok {
		t.Fatal("bad-name submodule reached the map")
	}
	if _, ok := m.Submodules["good"]; !ok {
		t.Fatal("good submodule missing")
	}

	// Empty path/URL do NOT skip — the submodule still lands in the map.
	for _, input := range []string{
		"[submodule \"s\"]\n\tpath = \n\turl = u\n",
		"[submodule \"s\"]\n\tpath = p\n\turl = \n",
	} {
		m2 := NewModules()
		if err := m2.Unmarshal([]byte(input)); err != nil {
			t.Fatalf("%q: %v", input, err)
		}
		if _, ok := m2.Submodules["s"]; !ok {
			t.Fatalf("%q: submodule with empty option was skipped", input)
		}
	}
}

// TestDetail03: a `..` path component (either separator, either end) fails;
// `..` inside a component name is legal.
func TestDetail03(t *testing.T) {
	for _, bad := range []string{"a/../b", "a\\..\\b", "../x", "x/..", "x/../"} {
		s := &Submodule{Name: "n", Path: bad, URL: "u"}
		if err := s.Validate(); !errors.Is(err, ErrModuleBadPath) {
			t.Fatalf("Path %q = %v, want ErrModuleBadPath", bad, err)
		}
	}
	for _, ok := range []string{"a/x../b", "a/..x/b", "plain"} {
		s := &Submodule{Name: "n", Path: ok, URL: "u"}
		if err := s.Validate(); err != nil {
			t.Fatalf("Path %q = %v", ok, err)
		}
	}
}

// TestDetail04 (shape — Inferable: no): marshaling a submodule with an
// empty name still produces a submodule subsection — its path supplies the
// label.
func TestDetail04(t *testing.T) {
	m := NewModules()
	m.Submodules["k"] = &Submodule{Path: "libs/dep", URL: "u"}
	out, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "submodule") ||
		!strings.Contains(string(out), "libs/dep") {
		t.Fatalf("marshal = %q", out)
	}
}

// TestDetail05 (shape — Inferable: no): empty branch fields are absent from
// the marshaled subsection rather than written empty.
func TestDetail05(t *testing.T) {
	b := &Branch{Name: "b"}
	sub := b.marshal()
	for _, key := range []string{"remote", "merge", "rebase", "description"} {
		if sub.HasOption(key) {
			t.Fatalf("empty %q written", key)
		}
	}
}

// TestDetail06: rebase accepts true/interactive/false; other values fail;
// merge without refs/ prefix is accepted.
func TestDetail06(t *testing.T) {
	for _, ok := range []string{"true", "interactive", "false"} {
		b := &Branch{Name: "b", Rebase: ok}
		if err := b.Validate(); err != nil {
			t.Fatalf("rebase %q rejected: %v", ok, err)
		}
	}
	b := &Branch{Name: "b", Rebase: "sometimes"}
	if err := b.Validate(); !errors.Is(err, errBranchInvalidRebase) {
		t.Fatalf("bad rebase = %v", err)
	}
	if err := (&Branch{Name: "b", Merge: "plainname"}).Validate(); err != nil {
		t.Fatalf("non-refs merge rejected: %v", err)
	}
}

// TestDetail07: description newlines marshal to a literal `\n` escape and
// unmarshal back.
func TestDetail07(t *testing.T) {
	b := &Branch{Name: "b", Description: "line1\nline2"}
	sub := b.marshal()
	got := sub.Option("description")
	if strings.Contains(got, "\n") {
		t.Fatalf("raw newline written: %q", got)
	}
	if !strings.Contains(got, `\n`) {
		t.Fatalf("literal escape missing: %q", got)
	}
	back := &Branch{Name: "b"}
	if err := back.unmarshal(sub); err != nil {
		t.Fatal(err)
	}
	if back.Description != "line1\nline2" {
		t.Fatalf("round-trip = %q", back.Description)
	}
}

// TestDetail08: parseConfigBool — true/yes/on, false/no/off (ci), decimal
// ints (0 false, non-0 true), empty → unset.
func TestDetail08(t *testing.T) {
	trues := []string{"true", "TRUE", "yes", "on", "1", "-1", "42"}
	for _, v := range trues {
		if parseConfigBool(v) != OptBoolTrue {
			t.Fatalf("parseConfigBool(%q) != True", v)
		}
	}
	for _, v := range []string{"false", "NO", "off", "0"} {
		if parseConfigBool(v) != OptBoolFalse {
			t.Fatalf("parseConfigBool(%q) != False", v)
		}
	}
	if parseConfigBool("") != OptBoolUnset {
		t.Fatal("empty string did not map to unset")
	}
}

// TestDetail09: an unrecognised bool value is unset, not an error or false.
func TestDetail09(t *testing.T) {
	for _, v := range []string{"maybe", "2x", "tru"} {
		if o := parseConfigBool(v); o != OptBoolUnset {
			t.Fatalf("parseConfigBool(%q) = %v, want unset", v, o)
		}
	}
}

// TestDetail10 (shape — Inferable: no): the unset value renders a third,
// distinct, non-empty spelling.
func TestDetail10(t *testing.T) {
	u := OptBoolUnset.String()
	if u == "" {
		t.Fatal("unset String() empty")
	}
	if u == OptBoolTrue.String() || u == OptBoolFalse.String() {
		t.Fatalf("unset String() = %q collides with a set spelling", u)
	}
}

// TestDetail11 (shape — Inferable: no): Marshal preserves raw subsections —
// unknown options round-trip.
func TestDetail11(t *testing.T) {
	m := NewModules()
	err := m.Unmarshal([]byte(`
[submodule "s"]
	path = p
	url = u
	unknownkey = keepme
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "unknownkey") {
		t.Fatalf("unknown option lost: %q", out)
	}
}

// TestDetail12 (shape — Inferable: no): validation reports the first bad
// field in name→path-empty→url-empty→path-bad order — identified by which
// error var surfaces.
func TestDetail12(t *testing.T) {
	all := &Submodule{Name: "/bad", Path: "", URL: ""}
	if err := all.Validate(); !errors.Is(err, ErrModuleBadName) {
		t.Fatalf("all-bad = %v, want name error first", err)
	}
	noURL := &Submodule{Name: "n", Path: "p", URL: ""}
	if err := noURL.Validate(); !errors.Is(err, ErrModuleEmptyURL) {
		t.Fatalf("empty url = %v", err)
	}
	badPath := &Submodule{Name: "n", Path: "a/../b", URL: "u"}
	if err := badPath.Validate(); !errors.Is(err, ErrModuleBadPath) {
		t.Fatalf("bad path = %v", err)
	}
}
