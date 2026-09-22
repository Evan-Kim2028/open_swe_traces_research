// Package reflectutils_test is the hidden black-box suite for fieldpath.
// One TestDetailNN per DETAILS.md commitment. Exported API only.
package reflectutils_test

import (
	"os"
	"os/exec"
	"reflect"
	"testing"

	ru "example.internal/clustkit/util/pkg/reflectutils"
)

func mustParse(t *testing.T, s string) *ru.FieldPath {
	t.Helper()
	p, err := ru.ParseFieldPath(s)
	if err != nil {
		t.Fatalf("ParseFieldPath(%q): %v", s, err)
	}
	return p
}

// Detail 1 (Inferable: yes): rendering — fields join with '.', map keys
// [key], array indexes [n], wildcards [*], no leading dot.
func TestDetail01(t *testing.T) {
	cases := map[string]string{
		"a":          "a",
		"a.b":        "a.b",
		"a.b.c":      "a.b.c",
		"a[0]":       "a[0]",
		"a[12]":      "a[12]",
		"a[*]":       "a[*]",
		"a.b[0].c":   "a.b[0].c",
		"a[0][1][2]": "a[0][1][2]",
	}
	for in, want := range cases {
		p := mustParse(t, in)
		if got := p.String(); got != want {
			t.Fatalf("String(Parse(%q)) = %q, want %q", in, got, want)
		}
	}
}

// Detail 2 (Inferable: no): '/' is accepted as a separator like '.'.
// Asserted as equivalence, not the spelling: a/b parses and matches a.b.
func TestDetail02(t *testing.T) {
	slash := mustParse(t, "a/b")
	dot := mustParse(t, "a.b")
	if !slash.Matches(dot) {
		t.Fatal("a/b did not parse as a.b")
	}
	deep := mustParse(t, "a/b/c[0]/d")
	mixed := mustParse(t, "a.b.c[0].d")
	if !deep.Matches(mixed) {
		t.Fatal("mixed separators did not parse equivalently")
	}
}

// Detail 3 (Inferable: no): 'a..b' collapses to 'a.b' — repeated separators
// never produce empty elements.
func TestDetail03(t *testing.T) {
	if !mustParse(t, "a..b").Matches(mustParse(t, "a.b")) {
		t.Fatal("a..b did not collapse to a.b")
	}
	if !mustParse(t, "a...b").Matches(mustParse(t, "a.b")) {
		t.Fatal("a...b did not collapse to a.b")
	}
	if got := mustParse(t, "a..b").String(); got != "a.b" {
		t.Fatalf("a..b rendered %q", got)
	}
}

// Detail 4 (Inferable: partially): three bracket element kinds exist —
// WildcardIndex, ArrayIndex, MapKey. `[n]` and `[*]` are parseable spellings;
// a MapKey element renders `[token]` — observed via ReflectRecursive over a
// map (parse does not accept bare key brackets; see Detail 5).
func TestDetail04(t *testing.T) {
	for _, s := range []string{"a[0]", "a[*]"} {
		p := mustParse(t, s)
		if p.String() != s {
			t.Fatalf("Parse(%q).String() = %q", s, p.String())
		}
	}
	// MapKey render: walking a map produces a [key] element.
	var paths []string
	err := ru.ReflectRecursive(reflect.ValueOf(map[string]int{"k": 1}),
		func(p *ru.FieldPath, f *reflect.StructField, v reflect.Value) error {
			paths = append(paths, p.String())
			return nil
		}, nil)
	if err != nil {
		t.Fatalf("ReflectRecursive: %v", err)
	}
	found := false
	for _, p := range paths {
		if p == "[k]" {
			found = true
		}
	}
	if !found {
		t.Fatalf("map walk produced no [k] map-key element: %v", paths)
	}
}

// Detail 5 (Inferable: partially): an unclosed '[' or a non-integer, non-'*'
// bracket body is a parse error. `[k]`/`["k"]` are errors — MapKey elements
// are produced only programmatically, not by the parser.
func TestDetail05(t *testing.T) {
	for _, bad := range []string{"a[", "a.b[", "a[0", "a[k", "a[k]", `a["k"]`, "a[]", "a[1.5]"} {
		if _, err := ru.ParseFieldPath(bad); err == nil {
			t.Fatalf("ParseFieldPath(%q): expected error", bad)
		}
	}
}

// Detail 6 (Inferable: partially): Matches requires equal counts;
// HasPrefixMatch tolerates a longer receiver.
func TestDetail06(t *testing.T) {
	if !mustParse(t, "a.b").Matches(mustParse(t, "a.b")) {
		t.Fatal("identical paths do not match")
	}
	if mustParse(t, "a.b").Matches(mustParse(t, "a.b.c")) {
		t.Fatal("Matches accepted different element counts")
	}
	if !mustParse(t, "a.b.c").HasPrefixMatch(mustParse(t, "a.b")) {
		t.Fatal("HasPrefixMatch rejected a longer receiver")
	}
	if mustParse(t, "a.b").HasPrefixMatch(mustParse(t, "a.b.c")) {
		t.Fatal("HasPrefixMatch accepted a shorter receiver")
	}
}

// Detail 7 (Inferable: no): wildcard asymmetry — a WildcardIndex in the
// receiver matches an ArrayIndex in the argument; it does not match a
// MapKey; a wildcard in the argument never matches. Asserted as the stated
// partition: exactly one direction matches for array index, neither for map.
func TestDetail07(t *testing.T) {
	recv := mustParse(t, "a[*]")
	argIdx := mustParse(t, "a[0]")
	// MapKey element built via the intact Extend helper (token is unexported).
	argKey := mustParse(t, "a").Extend(ru.FieldPathElement{Type: ru.FieldPathElementTypeMapKey})
	if recv.Matches(argIdx) == argIdx.Matches(recv) {
		t.Fatalf("wildcard match is symmetric: recv->idx=%v idx->recv=%v",
			recv.Matches(argIdx), argIdx.Matches(recv))
	}
	if !recv.Matches(argIdx) {
		t.Fatal("receiver wildcard did not match argument array index")
	}
	if recv.Matches(argKey) || argKey.Matches(recv) {
		t.Fatal("wildcard matched a map key")
	}
}

// Detail 8 (Inferable: yes): element equality is exact — type, token and
// number must agree.
func TestDetail08(t *testing.T) {
	pairs := [][2]string{
		{"a[0]", "a[1]"},
		{"a.b", "a.c"},
		{"a[*]", "a[*]"},
		{"a[0]", "a[0]"},
	}
	for i, pr := range pairs {
		got := mustParse(t, pr[0]).Matches(mustParse(t, pr[1]))
		want := pr[0] == pr[1]
		if got != want {
			t.Fatalf("pair %d: Matches(%q,%q) = %v want %v", i, pr[0], pr[1], got, want)
		}
	}
	// MapKey element differs from an ArrayIndex at the same position even
	// though both occupy slot 1.
	mk := mustParse(t, "a").Extend(ru.FieldPathElement{Type: ru.FieldPathElementTypeMapKey})
	if mk.Matches(mustParse(t, "a[0]")) {
		t.Fatal("MapKey element matched ArrayIndex")
	}
}

// Detail 9 (Inferable: yes): IsEmpty for a zero-element path.
func TestDetail09(t *testing.T) {
	if !mustParse(t, "").IsEmpty() {
		t.Fatal("empty string did not parse to empty path")
	}
	if mustParse(t, "a").IsEmpty() {
		t.Fatal("non-empty path reported IsEmpty")
	}
}

// Detail 10 (Inferable: no): an unknown element type makes String() abort the
// process (klog.Fatalf), not return an error. Asserted via subprocess death.
func TestDetail10helper(t *testing.T) {
	if os.Getenv("BB_FP_DEATH") != "1" {
		return
	}
	p, err := ru.ParseFieldPath("a")
	if err != nil {
		os.Exit(3)
	}
	p = p.Extend(ru.FieldPathElement{Type: ru.FieldPathElementType(99)})
	_ = p.String()
	os.Exit(0) // reached only if String() did not abort
}

func TestDetail10(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestDetail10helper$")
	cmd.Env = append(os.Environ(), "BB_FP_DEATH=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("String() on unknown element type returned normally: %s", out)
	}
}
