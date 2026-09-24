package object

import (
	"sort"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	"example.internal/gitkit/v6/storage/memory"
	"example.internal/gitkit/v6/utils/merkletrie"
)

func storeBlob(t *testing.T, st *memory.Storage, content string) plumbing.Hash {
	t.Helper()
	o := st.NewEncodedObject()
	o.SetType(plumbing.BlobObject)
	w, err := o.Writer()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	h, err := st.SetEncodedObject(o)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func fileEntry(t *testing.T, st *memory.Storage, name, content string) ChangeEntry {
	t.Helper()
	h := storeBlob(t, st, content)
	tr := &Tree{s: st, Entries: []TreeEntry{{Name: name, Mode: filemode.Regular, Hash: h}}}
	return ChangeEntry{Name: name, Tree: tr,
		TreeEntry: TreeEntry{Name: name, Mode: filemode.Regular, Hash: h}}
}

func dirEntry(st *memory.Storage, name string) ChangeEntry {
	return ChangeEntry{Name: name, Tree: &Tree{s: st},
		TreeEntry: TreeEntry{Name: name, Mode: filemode.Dir}}
}

// TestDetail01: both-sides-empty is malformed; one-side-empty is insert or
// delete; both present is modify.
func TestDetail01(t *testing.T) {
	st := memory.NewStorage()
	if _, err := (&Change{}).Action(); err == nil {
		t.Fatal("empty change action: want error")
	}
	c := &Change{To: fileEntry(t, st, "f", "x")}
	if a, err := c.Action(); err != nil || a != merkletrie.Insert {
		t.Fatalf("insert action = %v, %v", a, err)
	}
	c = &Change{From: fileEntry(t, st, "f", "x")}
	if a, err := c.Action(); err != nil || a != merkletrie.Delete {
		t.Fatalf("delete action = %v, %v", a, err)
	}
	c = &Change{From: fileEntry(t, st, "f", "x"), To: fileEntry(t, st, "f", "y")}
	if a, err := c.Action(); err != nil || a != merkletrie.Modify {
		t.Fatalf("modify action = %v, %v", a, err)
	}
}

// TestDetail02: "empty" is full-struct equality — a name-only entry still
// counts as present.
func TestDetail02(t *testing.T) {
	c := &Change{From: ChangeEntry{Name: "a.txt"}}
	if a, err := c.Action(); err != nil || a != merkletrie.Delete {
		t.Fatalf("name-only From: action = %v, %v — want Delete, not malformed", a, err)
	}
	c = &Change{From: ChangeEntry{Name: "a.txt"}, To: ChangeEntry{Name: "a.txt"}}
	if a, err := c.Action(); err != nil || a != merkletrie.Modify {
		t.Fatalf("name-only both sides: action = %v, %v — want Modify", a, err)
	}
}

// TestDetail03: Files resolves only the sides the action needs; a non-file
// entry yields (nil, nil, nil) with no error.
func TestDetail03(t *testing.T) {
	st := memory.NewStorage()

	c := &Change{To: fileEntry(t, st, "f", "content")}
	from, to, err := c.Files()
	if err != nil {
		t.Fatalf("insert Files: %v", err)
	}
	if from != nil || to == nil {
		t.Fatalf("insert Files = (%v, %v), want (nil, file)", from, to)
	}

	c = &Change{From: fileEntry(t, st, "f", "content")}
	from, to, err = c.Files()
	if err != nil {
		t.Fatalf("delete Files: %v", err)
	}
	if from == nil || to != nil {
		t.Fatalf("delete Files = (%v, %v), want (file, nil)", from, to)
	}

	c = &Change{To: dirEntry(st, "d")}
	from, to, err = c.Files()
	if err != nil || from != nil || to != nil {
		t.Fatalf("non-file entry Files = (%v, %v, %v), want (nil, nil, nil)", from, to, err)
	}
}

// TestDetail04: the display name prefers the FROM path.
func TestDetail04(t *testing.T) {
	st := memory.NewStorage()
	c := &Change{From: fileEntry(t, st, "old.txt", "x")}
	s := c.String()
	if !strings.Contains(s, "old.txt") {
		t.Fatalf("delete String() = %q, want old path reported", s)
	}
	c = &Change{From: fileEntry(t, st, "before.txt", "x"), To: fileEntry(t, st, "after.txt", "y")}
	s = c.String()
	if !strings.Contains(s, "before.txt") {
		t.Fatalf("modify String() = %q, want FROM path preferred", s)
	}
}

// TestDetail05: Changes.Less sorts on the preferred (FROM) name — deletions
// sort by their old path.
func TestDetail05(t *testing.T) {
	st := memory.NewStorage()
	del := &Change{From: fileEntry(t, st, "zzz.txt", "x")}
	ins := &Change{To: fileEntry(t, st, "aaa.txt", "y")}
	cs := Changes{del, ins}
	sort.Sort(cs)
	if cs[0] != ins || cs[1] != del {
		t.Fatal("deletion did not sort by its FROM path")
	}
}

// TestDetail06: Lines splits on \n only — carriage returns are not stripped.
func TestDetail06(t *testing.T) {
	st := memory.NewStorage()
	fe := fileEntry(t, st, "f", "a\r\nb\rc\n")
	_, f, err := (&Change{To: fe}).Files()
	if err != nil {
		t.Fatal(err)
	}
	lines, err := f.Lines()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "a\r" || lines[1] != "b\rc" {
		t.Fatalf("Lines = %q, want [a\\r b\\rc] — CR must survive, even mid-line", lines)
	}
}

// TestDetail07: trailing newline yields no phantom tail; an empty file yields
// an empty slice.
func TestDetail07(t *testing.T) {
	st := memory.NewStorage()
	_, f, err := (&Change{To: fileEntry(t, st, "f", "a\nb\n")}).Files()
	if err != nil {
		t.Fatal(err)
	}
	lines, err := f.Lines()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Fatalf("Lines = %q, want [a b]", lines)
	}
	_, f, err = (&Change{To: fileEntry(t, st, "e", "")}).Files()
	if err != nil {
		t.Fatal(err)
	}
	lines, err = f.Lines()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Fatalf("empty file Lines = %q, want empty slice", lines)
	}
}

// TestDetail08: binary detection is content-based — a .txt with NUL bytes is
// binary.
func TestDetail08(t *testing.T) {
	st := memory.NewStorage()
	_, f, err := (&Change{To: fileEntry(t, st, "f.txt", "text\x00payload")}).Files()
	if err != nil {
		t.Fatal(err)
	}
	if b, err := f.IsBinary(); err != nil || !b {
		t.Fatalf("IsBinary(NUL content .txt) = %v, %v — want true", b, err)
	}
	_, f, err = (&Change{To: fileEntry(t, st, "g", "plain text\n")}).Files()
	if err != nil {
		t.Fatal(err)
	}
	if b, err := f.IsBinary(); err != nil || b {
		t.Fatalf("IsBinary(clean) = %v, %v — want false", b, err)
	}
}

// TestDetail09: String renders the action and path; a malformed change renders
// a "malformed" marker rather than failing.
func TestDetail09(t *testing.T) {
	st := memory.NewStorage()
	s := (&Change{From: fileEntry(t, st, "gone.txt", "x")}).String()
	if !strings.Contains(s, "Delete") || !strings.Contains(s, "gone.txt") {
		t.Fatalf("String() = %q, want action and path", s)
	}
	s = (&Change{}).String()
	if !strings.Contains(s, "malformed") {
		t.Fatalf("malformed change String() = %q, want a malformed marker", s)
	}
}

// TestDetail10: Contents reads the whole blob into memory.
func TestDetail10(t *testing.T) {
	st := memory.NewStorage()
	content := strings.Repeat("0123456789\n", 2000) // 22KB, past any sniff window
	_, f, err := (&Change{To: fileEntry(t, st, "big", content)}).Files()
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.Contents()
	if err != nil {
		t.Fatal(err)
	}
	if got != content {
		t.Fatalf("Contents truncated: %d of %d bytes", len(got), len(content))
	}
}
