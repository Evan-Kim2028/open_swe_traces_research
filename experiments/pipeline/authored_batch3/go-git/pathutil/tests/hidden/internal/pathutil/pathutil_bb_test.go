package pathutil

import (
	"testing"
)

// TestDetail01: `.git` matches case-insensitively and via its 8.3 alias
// `git~1` — but `git~2`/`GIT~9` are not covered by this predicate.
func TestDetail01(t *testing.T) {
	for _, yes := range []string{".git", ".GIT", ".Git", "git~1"} {
		if !IsDotGitName(yes) {
			t.Fatalf("IsDotGitName(%q) = false", yes)
		}
	}
	for _, no := range []string{"git~2", "GIT~9", ".gitx", "git", ".gi"} {
		if IsDotGitName(no) {
			t.Fatalf("IsDotGitName(%q) = true", no)
		}
	}
}

// TestDetail02: HFS matching skips ignored codepoints between expected
// bytes, is case-insensitive, and requires the component to end after the
// needle.
func TestDetail02(t *testing.T) {
	if !IsHFSDotGit(".git") || !IsHFSDotGit(".GIT") {
		t.Fatal("plain/case .git not matched")
	}
	if !IsHFSDotGit(".g\u200cit") { // U+200C is in the ignored table
		t.Fatal("ignored codepoint inside needle not skipped")
	}
	if IsHFSDotGit(".gitx") {
		t.Fatal("component not ending after needle matched")
	}
	if IsHFSDotGit(".gi") {
		t.Fatal("partial needle matched")
	}
	if !IsHFSDotGitmodules(".gitmodules") || IsHFSDotGitattributes(".gitmodules") {
		t.Fatal("needle-specific wrappers misbehave")
	}
}

// TestDetail03: NTFS pattern 1 — `.<needle>` followed only by
// spaces/periods; a `:` ends the scan early (ADS always allowed).
func TestDetail03(t *testing.T) {
	for _, yes := range []string{".git", ".git ", ".git.", ".git  ..", ".git::$INDEX_ALLOCATION", ".git:x"} {
		if !IsNTFSDotGit(yes) {
			t.Fatalf("IsNTFSDotGit(%q) = false", yes)
		}
	}
	for _, no := range []string{".gitx", ".gitx:x", ".gi", ".gitmodules"} {
		if IsNTFSDotGit(no) {
			t.Fatalf("IsNTFSDotGit(%q) = true", no)
		}
	}
}

// TestDetail04 (shape — Inferable: no): NTFS pattern 2 catches the 8.3
// shortname form `<prefix>~<digit>` for a bounded digit range — some digits
// match and some do not; the exact bound is not derivable.
func TestDetail04(t *testing.T) {
	var hits []int
	for d := '1'; d <= '9'; d++ {
		if IsNTFSDotGitmodules("gitmod~" + string(d)) {
			hits = append(hits, int(d-'0'))
		}
	}
	if len(hits) == 0 || len(hits) == 9 {
		t.Fatalf("digit hits = %v, want a bounded non-empty range", hits)
	}
}

// TestDetail05: reserved-name check compares only the base — CON, con.txt,
// NUL:x, "LPT3 " match; a mere prefix does not.
func TestDetail05(t *testing.T) {
	for _, bad := range []string{"CON", "con.txt", "NUL:x", "LPT3 ", "com1"} {
		if WindowsValidPath(bad) {
			t.Fatalf("WindowsValidPath(%q) = true for reserved name", bad)
		}
	}
	for _, ok := range []string{"CONTACT", "console.log", "nullex"} {
		if !WindowsValidPath(ok) {
			t.Fatalf("WindowsValidPath(%q) = false for non-reserved", ok)
		}
	}
}

// TestDetail06 (shape — Inferable: no): a plain `.git`/`git~1` is VALID to
// WindowsValidPath — only NTFS-disguised variants and reserved names fail.
// The committed contrast: ordinary names pass, disguised ones fail.
func TestDetail06(t *testing.T) {
	if !WindowsValidPath(".git") || !WindowsValidPath("git~1") {
		t.Fatal("plain .git/git~1 rejected by WindowsValidPath")
	}
	for _, bad := range []string{".git::$INDEX_ALLOCATION", "CON"} {
		if WindowsValidPath(bad) {
			t.Fatalf("WindowsValidPath(%q) = true", bad)
		}
	}
}

// TestDetail07: ValidTreePath rejects control bytes, empty/separator-only
// paths, `.`/`..` components, volume prefixes, and `.git` disguises at ANY
// component position — but NOT Windows device names.
func TestDetail07(t *testing.T) {
	// Volume-name prefixes are checked via filepath.VolumeName, which is a
	// no-op off Windows — not assertable on this platform.
	for _, bad := range []string{
		"", "/", "//", "a/./b", "a/../b", "..", "a/b\x01c",
		"a/.git/b", ".GIT/x", "a/git~1",
	} {
		if err := ValidTreePath(bad); err == nil {
			t.Fatalf("ValidTreePath(%q) = nil", bad)
		}
	}
	for _, ok := range []string{"a/b/c", "a/CON/b", "normal.txt", ".gitignore"} {
		if err := ValidTreePath(ok); err != nil {
			t.Fatalf("ValidTreePath(%q) = %v", ok, err)
		}
	}
}

// TestDetail08: HasUnsafeComponent scans `/` AND `\` components for control
// bytes (firing anywhere) and runs the fold checks only on components
// containing a `.` — flagging `..` components, not dot-less or benign ones.
func TestDetail08(t *testing.T) {
	for _, bad := range []string{"a/../b", "a\\..\\b", "a/b\x01c", "nodot\x7f", ".."} {
		if !HasUnsafeComponent(bad) {
			t.Fatalf("HasUnsafeComponent(%q) = false", bad)
		}
	}
	for _, ok := range []string{"a/b/c", "a/gitx", "no-dots-here", "a/..b", "a/b.."} {
		if HasUnsafeComponent(ok) {
			t.Fatalf("HasUnsafeComponent(%q) = true", ok)
		}
	}
}

// TestDetail09: the fold checks catch `..` components reached through HFS/
// NTFS canonicalisation — ignored codepoints, trailing spaces/dots, ADS
// suffixes.
func TestDetail09(t *testing.T) {
	for _, bad := range []string{"a/.. /b", "a/.‌./b", "a/..::$I/b", "a/..:/b"} {
		if !HasUnsafeComponent(bad) {
			t.Fatalf("HasUnsafeComponent(%q) = false", bad)
		}
	}
}

// TestDetail10 (shape — Inferable: no): `~` expansion — bare `~`/`~x` return
// unchanged; on lookup failure the ORIGINAL path comes back with the error.
func TestDetail10(t *testing.T) {
	for _, in := range []string{"~", "~x", "/abs/path", "rel/path"} {
		got, err := ReplaceTildeWithHome(in)
		if err == nil && got != in {
			t.Fatalf("ReplaceTildeWithHome(%q) = %q, want unchanged", in, got)
		}
	}
	got, err := ReplaceTildeWithHome("~nonexistent-user-zzz/path")
	if err != nil && got != "~nonexistent-user-zzz/path" {
		t.Fatalf("lookup failure returned %q, want original", got)
	}
}

// TestDetail11: backslash is a separator on non-Windows hosts too.
func TestDetail11(t *testing.T) {
	if err := ValidTreePath("a\\.git"); err == nil {
		t.Fatal("backslash-separated .git accepted")
	}
	if !HasUnsafeComponent("x\\..\\y") {
		t.Fatal("backslash-separated .. not caught")
	}
}

// TestDetail12: IsNTFSDot pattern 3 uses the caller-supplied short-name
// prefix — the prefix argument controls which `~N` forms match.
func TestDetail12(t *testing.T) {
	if !IsNTFSDot("gitmod~1", ".gitmodules", "gitmod") {
		t.Fatal("shortname prefix not honoured")
	}
	if IsNTFSDot("other~1", ".gitmodules", "gitmod") {
		t.Fatal("prefix-mismatched shortname matched")
	}
}
