package pathutil

import "testing"

// TestDetail01 (partially): IsHFSDot(part, needle) is true iff, after
// dropping ignored code points and folding remaining ASCII letters to
// lower case, the result equals "."+needle.
func TestDetail01(t *testing.T) {
	for _, c := range []struct {
		part, needle string
		want         bool
	}{
		{".git", "git", true},
		{".GIT", "git", true},
		{"git", "git", false},
		{".gi", "git", false},
		{".gitx", "git", false},
		{".gitmodules", "gitmodules", true},
	} {
		if got := IsHFSDot(c.part, c.needle); got != c.want {
			t.Fatalf("IsHFSDot(%q, %q) = %v, want %v", c.part, c.needle, got, c.want)
		}
	}
}

// TestDetail02 (shape — Inferable: no): ignored code points may appear
// before the dot, between letters of the needle, and after the last
// letter; they are skipped rather than matched.
func TestDetail02(t *testing.T) {
	for _, part := range []string{
		"\u200d.git", // ignored before the dot
		".g\u200dit", // ignored inside the needle
		".git\u200e", // ignored after the last letter
		"\u200d.g\u200di\u200et\u200d",
	} {
		if !IsHFSDotGit(part) {
			t.Fatalf("IsHFSDotGit(%q) = false, want true", part)
		}
	}
}

// TestDetail03 (shape — Inferable: no): the ignored set is the HFS+
// ignorable list — members are skipped, code points just outside it are
// not.
func TestDetail03(t *testing.T) {
	for _, cp := range []string{
		"\u200c", "\u200d", "\u200e", "\u200f", // 200C-200F
		"\u202a", "\u202b", "\u202c", "\u202d", "\u202e", // 202A-202E
		"\u206a", "\u206b", "\u206c", "\u206d", "\u206e", "\u206f", // 206A-206F
		"\ufeff", // FEFF
	} {
		if !IsHFSDotGit(".g" + cp + "it") {
			t.Fatalf("code point %U not ignored", []rune(cp)[0])
		}
	}
	for _, cp := range []string{
		"\u200b", // 200B zero-width space — just below the set
		"\u00ad", // 00AD soft hyphen — not in the set
		"\u2070", // 2070 — just past 206F
	} {
		if IsHFSDotGit(".g" + cp + "it") {
			t.Fatalf("code point %U was ignored but is not in the set", []rune(cp)[0])
		}
	}
}

// TestDetail04 (yes): after skipping ignored code points, the next
// remaining rune must be '.'.
func TestDetail04(t *testing.T) {
	if IsHFSDot("\u200cgit", "git") {
		t.Fatal("no dot after skipped prefix, still matched")
	}
	if !IsHFSDot("\u200c.git", "git") {
		t.Fatal("ignored prefix then dot did not match")
	}
	if IsHFSDot("x.git", "git") {
		t.Fatal("non-ignored rune before dot matched")
	}
}

// TestDetail05 (yes): each needle rune is matched against the next
// non-ignored rune of part.
func TestDetail05(t *testing.T) {
	if !IsHFSDot(".g\u200ci\u200ct", "git") {
		t.Fatal("needle runes interleaved with ignored cps did not match")
	}
	if IsHFSDot(".g\u200cxt", "git") {
		t.Fatal("wrong non-ignored rune matched the needle")
	}
}

// TestDetail06 (shape — Inferable: no): a non-ignored rune above 127 in a
// needle position makes the match fail.
func TestDetail06(t *testing.T) {
	for _, part := range []string{
		".g\u0416t", // Cyrillic ZHE
		".g\u0130t", // İ has a lowercase fold — still fails (non-ASCII)
		".gi\u0442", // Cyrillic te in last position
	} {
		if IsHFSDot(part, "git") {
			t.Fatalf("IsHFSDot(%q) matched a non-ASCII needle position", part)
		}
	}
}

// TestDetail07 (partially): ASCII case folding uses unicode.ToLower on the
// remaining rune; the rune is required to be ASCII.
func TestDetail07(t *testing.T) {
	for _, part := range []string{".GIT", ".GiT", ".gIt"} {
		if !IsHFSDot(part, "git") {
			t.Fatalf("IsHFSDot(%q) = false, want true", part)
		}
	}
	// Non-ASCII uppercase is never folded in.
	if IsHFSDot(".\u0120it", "git") {
		t.Fatal("non-ASCII uppercase folded into the needle")
	}
}

// TestDetail08 (yes): after the needle is consumed, only ignored code
// points may remain; any other leftover rune fails.
func TestDetail08(t *testing.T) {
	if IsHFSDot(".gitx", "git") {
		t.Fatal("trailing non-ignored rune matched")
	}
	if IsHFSDot(".git\u200cx", "git") {
		t.Fatal("ignored rune then trailing junk matched")
	}
	if !IsHFSDot(".git\u200c", "git") {
		t.Fatal("trailing ignored rune failed")
	}
}

// TestDetail09 (yes): an empty part, a lone ".", or a needle that does not
// consume the whole spelling fails.
func TestDetail09(t *testing.T) {
	for _, c := range [][2]string{
		{"", "git"},
		{".", "git"},
		{".gi", "git"},
		{".gitmodules", "git"},
		{".git", "gitmodules"},
	} {
		if IsHFSDot(c[0], c[1]) {
			t.Fatalf("IsHFSDot(%q, %q) = true, want false", c[0], c[1])
		}
	}
}

// TestDetail10 (yes): IsHFSDotGit(".GIT") is true; IsHFSDotGit(".gitmodules")
// is false.
func TestDetail10(t *testing.T) {
	if !IsHFSDotGit(".GIT") {
		t.Fatal("IsHFSDotGit(.GIT) = false")
	}
	if IsHFSDotGit(".gitmodules") {
		t.Fatal("IsHFSDotGit(.gitmodules) = true")
	}
	if !IsHFSDotGitmodules(".GITMODULES") || !IsHFSDotGitattributes(".gitattributes") ||
		!IsHFSDotGitignore(".gitignore") || !IsHFSDotMailmap(".mailmap") {
		t.Fatal("a needle wrapper failed on its plain spelling")
	}
}
