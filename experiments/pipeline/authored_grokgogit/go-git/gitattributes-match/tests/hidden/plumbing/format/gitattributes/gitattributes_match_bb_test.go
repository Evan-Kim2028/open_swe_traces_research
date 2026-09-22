package gitattributes

import "testing"

func gmMatch(p string, domain, path []string) bool {
	return ParsePattern(p, domain).Match(path)
}

// TestDetail01 (yes): if len(path) <= len(domain) the match fails.
func TestDetail01(t *testing.T) {
	if gmMatch("x", []string{"a"}, []string{"a"}) {
		t.Fatal("path as short as domain matched")
	}
	if gmMatch("x", []string{"a", "b"}, []string{"a", "b"}) {
		t.Fatal("path equal to domain matched")
	}
	if gmMatch("*", []string{"a"}, nil) {
		t.Fatal("empty path matched a non-empty domain")
	}
	if !gmMatch("x", []string{"a"}, []string{"a", "x"}) {
		t.Fatal("path longer than domain did not match")
	}
}

// TestDetail02 (yes): the path must start with the domain components,
// compared exactly.
func TestDetail02(t *testing.T) {
	if !gmMatch("*", []string{"a", "b"}, []string{"a", "b", "x"}) {
		t.Fatal("path under domain did not match")
	}
	if gmMatch("*", []string{"a", "b"}, []string{"a", "c", "x"}) {
		t.Fatal("path outside domain matched")
	}
	if gmMatch("*", []string{"a", "b"}, []string{"b", "a", "x"}) {
		t.Fatal("reordered domain prefix matched")
	}
}

// TestDetail03 (shape — Inferable: no): a single-segment pattern is
// matched only against the last component of path.
func TestDetail03(t *testing.T) {
	if !gmMatch("foo", nil, []string{"a", "b", "foo"}) {
		t.Fatal("single segment did not match last component")
	}
	if gmMatch("foo", nil, []string{"foo", "bar"}) {
		t.Fatal("single segment matched a non-last component")
	}
	if !gmMatch("foo", nil, []string{"foo"}) {
		t.Fatal("single segment did not match sole component")
	}
}

// TestDetail04 (partially): a multi-segment pattern is matched against
// path[len(domain):].
func TestDetail04(t *testing.T) {
	if !gmMatch("b/c", []string{"a"}, []string{"a", "b", "c"}) {
		t.Fatal("multi-segment did not match under domain")
	}
	if gmMatch("b/c", []string{"a"}, []string{"x", "b", "c"}) {
		t.Fatal("multi-segment matched outside domain")
	}
	if gmMatch("b/c", nil, []string{"a", "b", "c"}) {
		t.Fatal("multi-segment matched a suffix of the path")
	}
	if !gmMatch("b/c", nil, []string{"b", "c"}) {
		t.Fatal("multi-segment failed on equal-depth path")
	}
}

// TestDetail05 (shape — Inferable: no): empty pattern segments are
// skipped while path components remain. (A trailing empty segment on an
// exhausted path is governed by rule 10 and deliberately not asserted.)
func TestDetail05(t *testing.T) {
	if !gmMatch("a//b", nil, []string{"a", "b"}) {
		t.Fatal("empty interior segment was not skipped")
	}
	// Skipping the empty segment must not consume a path component: the
	// next segment still has to match the next component in lockstep.
	if gmMatch("a//b", nil, []string{"a", "x", "b"}) {
		t.Fatal("skipping an empty segment desynced the lockstep")
	}
}

// TestDetail06 (partially): a `**` segment eats itself and, when last,
// succeeds immediately.
func TestDetail06(t *testing.T) {
	if !gmMatch("a/**", nil, []string{"a", "b", "c", "d"}) {
		t.Fatal("trailing ** did not match deep path")
	}
	if !gmMatch("a/**", nil, []string{"a", "b"}) {
		t.Fatal("trailing ** did not match shallow path")
	}
	if !gmMatch("**", nil, []string{"a", "b"}) {
		t.Fatal("lone ** did not match")
	}
	if !gmMatch("**", nil, []string{"a"}) {
		t.Fatal("lone ** did not match one component")
	}
}

// TestDetail07 (shape — Inferable: no): `**` as a substring of a larger
// token (not a whole segment) fails the whole match.
func TestDetail07(t *testing.T) {
	if gmMatch("a**b", nil, []string{"a**b"}) {
		t.Fatal("embedded ** matched its own literal text")
	}
	if gmMatch("a**b", nil, []string{"axxb"}) {
		t.Fatal("embedded ** behaved like a glob star")
	}
	if gmMatch("x**/y", nil, []string{"x1", "y"}) {
		t.Fatal("embedded ** in a segment matched")
	}
}

// TestDetail08 (partially): after a `**`, later path components are
// scanned until the next pattern segment matches.
func TestDetail08(t *testing.T) {
	if !gmMatch("a/**/c", nil, []string{"a", "b1", "b2", "c"}) {
		t.Fatal("** did not scan across components")
	}
	if !gmMatch("a/**/c", nil, []string{"a", "c"}) {
		t.Fatal("** did not eat zero components")
	}
	if gmMatch("a/**/c", nil, []string{"a", "b", "c", "d"}) {
		t.Fatal("** swallowed the trailing segment")
	}
	if gmMatch("a/**/c", nil, []string{"a", "x", "y"}) {
		t.Fatal("pattern without a c tail matched")
	}
}

// TestDetail09 (yes): without a pending `**`, each path component must
// filepath.Match the next pattern segment in lockstep.
func TestDetail09(t *testing.T) {
	if !gmMatch("a/*", nil, []string{"a", "b"}) {
		t.Fatal("* segment did not match one component")
	}
	if gmMatch("a/*", nil, []string{"a", "b", "c"}) {
		t.Fatal("* segment matched two components")
	}
	if !gmMatch("a/?c", nil, []string{"a", "bc"}) {
		t.Fatal("? did not match a single char")
	}
	if gmMatch("a/?c", nil, []string{"a", "bxc"}) {
		t.Fatal("? matched two chars")
	}
	if !gmMatch("vul?ano", nil, []string{"vulcano"}) {
		t.Fatal("? failed on vulcano")
	}
}

// TestDetail10 (yes): if path components run out while pattern segments
// remain, the match fails.
func TestDetail10(t *testing.T) {
	if gmMatch("a/b/c", nil, []string{"a", "b"}) {
		t.Fatal("short path matched a longer pattern")
	}
	if gmMatch("a/b", nil, []string{"a"}) {
		t.Fatal("one-component path matched two-segment pattern")
	}
	// Path exhausted while the "**" segment remains: rule 6's immediate
	// success only applies once the ** is reached, so rule 10 fails it.
	if gmMatch("a/**", nil, []string{"a"}) {
		t.Fatal("trailing ** matched an already-exhausted path")
	}
}

// TestDetail11 (shape — Inferable: no): a filepath.Match error (malformed
// pattern) makes Match return false — not an error, not a panic.
func TestDetail11(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("malformed pattern panicked: %v", r)
		}
	}()
	if gmMatch("a/[", nil, []string{"a", "["}) {
		t.Fatal("malformed char class matched")
	}
	if gmMatch("a/[", nil, []string{"a", "b"}) {
		t.Fatal("malformed char class matched other input")
	}
}

// TestDetail12 (shape — Inferable: no): simple patterns do not match a
// component in the middle of the path, only the last one.
func TestDetail12(t *testing.T) {
	if gmMatch("b", nil, []string{"a", "b", "c"}) {
		t.Fatal("simple pattern matched a middle component")
	}
	if !gmMatch("b", nil, []string{"a", "b"}) {
		t.Fatal("simple pattern did not match the last component")
	}
}
