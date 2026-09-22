package object

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	fdiff "example.internal/gitkit/v6/plumbing/format/diff"
	"example.internal/gitkit/v6/storage/memory"
)

func patchStore(t *testing.T) *memory.Storage {
	t.Helper()
	return memory.NewStorage()
}

func storeObjBlob(t *testing.T, st *memory.Storage, content string) plumbing.Hash {
	t.Helper()
	obj := st.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	w, err := obj.Writer()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	h, err := st.SetEncodedObject(obj)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func storeObjTree(t *testing.T, st *memory.Storage, entries ...TreeEntry) *Tree {
	t.Helper()
	tr := &Tree{Entries: entries}
	obj := st.NewEncodedObject()
	if err := tr.Encode(obj); err != nil {
		t.Fatal(err)
	}
	h, err := st.SetEncodedObject(obj)
	if err != nil {
		t.Fatal(err)
	}
	gt, err := GetTree(st, h)
	if err != nil {
		t.Fatal(err)
	}
	return gt
}

// fileChange builds a modify-change over two stored blob contents.
func fileChange(t *testing.T, st *memory.Storage, name, from, to string) *Change {
	t.Helper()
	fromH := storeObjBlob(t, st, from)
	toH := storeObjBlob(t, st, to)
	tr := storeObjTree(t, st,
		TreeEntry{Name: name, Mode: filemode.Regular, Hash: toH})
	trFrom := storeObjTree(t, st,
		TreeEntry{Name: name, Mode: filemode.Regular, Hash: fromH})
	return &Change{
		From: ChangeEntry{Name: name, Tree: trFrom,
			TreeEntry: TreeEntry{Name: name, Mode: filemode.Regular, Hash: fromH}},
		To: ChangeEntry{Name: name, Tree: tr,
			TreeEntry: TreeEntry{Name: name, Mode: filemode.Regular, Hash: toH}},
	}
}

// stagedCtx keeps Done() open for the first n calls, then reports
// cancellation — used to land a cancel inside a mid-patch loop.
type stagedCtx struct {
	ch    chan struct{}
	calls int
	open  int
}

func newStagedCtx(open int) *stagedCtx {
	return &stagedCtx{ch: make(chan struct{}), open: open}
}

func (c *stagedCtx) Done() <-chan struct{} {
	c.calls++
	if c.calls > c.open {
		select {
		case <-c.ch:
		default:
			close(c.ch)
		}
	}
	return c.ch
}

func (c *stagedCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *stagedCtx) Err() error                  { return nil }
func (c *stagedCtx) Value(any) any               { return nil }

// 1. An empty change list still produces a Patch — carrying the message
//    and zero file patches, not an error. Inferable: partially.
func TestDetail01(t *testing.T) {
	p, err := getPatch("the commit message")
	if err != nil {
		t.Fatalf("getPatch with no changes: %v", err)
	}
	if p == nil {
		t.Fatal("nil patch")
	}
	if p.Message() != "the commit message" {
		t.Fatalf("message = %q", p.Message())
	}
	if n := len(p.FilePatches()); n != 0 {
		t.Fatalf("file patches = %d, want 0", n)
	}
}

// 2. Submodule (gitlink) entries never read blob content — the diff body
//    is the synthetic `Subproject commit <hash>` line on whichever side
//    is a submodule. Inferable: doc.
func TestDetail02(t *testing.T) {
	st := patchStore(t)
	h := plumbing.NewHash("abcdabcdabcdabcdabcdabcdabcdabcdabcdabcd")
	tr := storeObjTree(t, st,
		TreeEntry{Name: "sub", Mode: filemode.Submodule, Hash: h})
	e := ChangeEntry{Name: "sub", Tree: tr,
		TreeEntry: TreeEntry{Name: "sub", Mode: filemode.Submodule, Hash: h}}

	got := submoduleContent(e)
	want := "Subproject commit " + h.String() + "\n"
	if got != want {
		t.Fatalf("submoduleContent = %q, want %q", got, want)
	}

	// Non-submodule and empty entries produce no synthetic content.
	reg := ChangeEntry{TreeEntry: TreeEntry{Mode: filemode.Regular}}
	if s := submoduleContent(reg); s != "" {
		t.Fatalf("regular entry content = %q", s)
	}
	if s := submoduleContent(ChangeEntry{}); s != "" {
		t.Fatalf("empty entry content = %q", s)
	}
	if !isSubmodule(e) || isSubmodule(reg) || isSubmodule(ChangeEntry{}) {
		t.Fatal("isSubmodule misclassified")
	}
}

// 3. A change where either side is a submodule is diffed as text over
//    the synthetic lines — added or removed subprojects diff against
//    empty content. Inferable: partially.
func TestDetail03(t *testing.T) {
	st := patchStore(t)
	hA := plumbing.NewHash("1111111111111111111111111111111111111111")
	hB := plumbing.NewHash("2222222222222222222222222222222222222222")
	sub := func(h plumbing.Hash) ChangeEntry {
		tr := storeObjTree(t, st,
			TreeEntry{Name: "sub", Mode: filemode.Submodule, Hash: h})
		return ChangeEntry{Name: "sub", Tree: tr,
			TreeEntry: TreeEntry{Name: "sub", Mode: filemode.Submodule, Hash: h}}
	}

	// Add: the to-side synthetic line appears as an addition.
	c := &Change{To: sub(hB)}
	p, err := getPatch("", c)
	if err != nil {
		t.Fatal(err)
	}
	fp := p.FilePatches()[0]
	var addText, delText string
	for _, ch := range fp.Chunks() {
		switch ch.Type() {
		case fdiff.Add:
			addText += ch.Content()
		case fdiff.Delete:
			delText += ch.Content()
		}
	}
	if !strings.Contains(addText, "Subproject commit "+hB.String()) || delText != "" {
		t.Fatalf("submodule add chunks: add=%q del=%q", addText, delText)
	}

	// Bump: old line deleted, new line added — a text diff over the two
	// synthetic lines, no blob reads involved.
	c = &Change{From: sub(hA), To: sub(hB)}
	p, err = getPatch("", c)
	if err != nil {
		t.Fatal(err)
	}
	addText, delText = "", ""
	for _, ch := range p.FilePatches()[0].Chunks() {
		switch ch.Type() {
		case fdiff.Add:
			addText += ch.Content()
		case fdiff.Delete:
			delText += ch.Content()
		}
	}
	if !strings.Contains(delText, hA.String()) || !strings.Contains(addText, hB.String()) {
		t.Fatalf("submodule bump chunks: add=%q del=%q", addText, delText)
	}
}

// 4. Binary content suppresses chunks entirely — the file patch still
//    names both sides but carries no diff body. Inferable: partially.
func TestDetail04(t *testing.T) {
	st := patchStore(t)
	c := fileChange(t, st, "f.bin", "text\n", "a\x00b\x00c\n")
	p, err := getPatch("", c)
	if err != nil {
		t.Fatal(err)
	}
	fp := p.FilePatches()[0]
	if n := len(fp.Chunks()); n != 0 {
		t.Fatalf("binary change produced %d chunks", n)
	}
	if !fp.IsBinary() {
		t.Fatal("chunkless patch not reported binary")
	}
	from, to := fp.Files()
	if from == nil || to == nil || from.Path() != "f.bin" || to.Path() != "f.bin" {
		t.Fatalf("binary patch lost its file sides: %v %v", from, to)
	}
}

// 5. Every per-change and per-chunk loop honours context cancellation
//    and surfaces the same ErrCanceled — checked between items, not
//    inside reads. Inferable: partially.
func TestDetail05(t *testing.T) {
	st := patchStore(t)
	c := fileChange(t, st, "f.txt", "a\n", "b\n")

	// Already-canceled context: refused before the first item.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := getPatchContext(ctx, "", c); err != ErrCanceled {
		t.Fatalf("canceled ctx: %v", err)
	}

	// Cancellation landing inside the single change's chunk loop —
	// the first check (per-change) passes, the second call cancels.
	if _, err := getPatchContext(newStagedCtx(1), "", c); err != ErrCanceled {
		t.Fatalf("staged ctx: %v", err)
	}
}

// 6. Diff operation mapping is one-to-one with the dmp triple — equal,
//    delete, insert — in that order. Inferable: doc.
func TestDetail06(t *testing.T) {
	st := patchStore(t)
	c := fileChange(t, st, "f.txt", "alpha\nkeep\n", "beta\nkeep\n")
	p, err := getPatch("", c)
	if err != nil {
		t.Fatal(err)
	}
	chunks := p.FilePatches()[0].Chunks()
	if len(chunks) == 0 {
		t.Fatal("no chunks")
	}
	var fromB, toB strings.Builder
	var sawAdd, sawDel, sawEq bool
	for _, ch := range chunks {
		switch ch.Type() {
		case fdiff.Add:
			sawAdd = true
			toB.WriteString(ch.Content())
		case fdiff.Delete:
			sawDel = true
			fromB.WriteString(ch.Content())
		case fdiff.Equal:
			sawEq = true
			fromB.WriteString(ch.Content())
			toB.WriteString(ch.Content())
		default:
			t.Fatalf("unexpected chunk op %d", ch.Type())
		}
	}
	if !sawAdd || !sawDel || !sawEq {
		t.Fatalf("expected add+delete+equal chunks, got %+v", chunks)
	}
	if fromB.String() != "alpha\nkeep\n" {
		t.Fatalf("delete+equal reconstructs %q, want %q", fromB.String(), "alpha\nkeep\n")
	}
	if toB.String() != "beta\nkeep\n" {
		t.Fatalf("add+equal reconstructs %q, want %q", toB.String(), "beta\nkeep\n")
	}
}

// 7. A change entry is "empty" (no file side) when its mode is not a
//    file — EXCEPT submodules, which always count as present so their
//    gitlink is diffed. Inferable: doc.
func TestDetail07(t *testing.T) {
	entry := func(m filemode.FileMode) *changeEntryWrapper {
		return &changeEntryWrapper{ChangeEntry{
			Name:      "x",
			TreeEntry: TreeEntry{Name: "x", Mode: m, Hash: plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
		}}
	}
	for _, m := range []filemode.FileMode{filemode.Regular, filemode.Deprecated, filemode.Executable, filemode.Symlink, filemode.Submodule} {
		if entry(m).Empty() {
			t.Fatalf("mode %o reported empty", m)
		}
	}
	for _, m := range []filemode.FileMode{filemode.Dir, filemode.Empty} {
		if !entry(m).Empty() {
			t.Fatalf("mode %o not reported empty", m)
		}
	}
	// Empty entries expose neither path nor hash.
	e := entry(filemode.Dir)
	if e.Path() != "" || e.Hash() != plumbing.ZeroHash {
		t.Fatalf("empty entry leaked path=%q hash=%v", e.Path(), e.Hash())
	}
}

// 8. FileStats ignore patches with no chunks — binary files produce no
//    stat row at all. Inferable: partially.
func TestDetail08(t *testing.T) {
	st := patchStore(t)
	bin := fileChange(t, st, "b.bin", "x\n", "\x00\x01\x02\x00")
	txt := fileChange(t, st, "a.txt", "x\n", "y\n")

	p, err := getPatch("", bin, txt)
	if err != nil {
		t.Fatal(err)
	}
	stats := p.Stats()
	if len(stats) != 1 || stats[0].Name != "a.txt" {
		t.Fatalf("stats = %+v — binary change produced a row", stats)
	}
}

// 9. Stat names pick the surviving path for add/delete, and
//    `from => to` for a rename — the arrow with spaces is fixed.
//    Inferable: partially.
func TestDetail09(t *testing.T) {
	st := patchStore(t)

	add := &Change{To: fileChange(t, st, "new.txt", "", "data\n").To}
	del := &Change{From: fileChange(t, st, "old.txt", "data\n", "").From}

	// Rename: same content, different path.
	h := storeObjBlob(t, st, "same\n")
	trFrom := storeObjTree(t, st, TreeEntry{Name: "a.txt", Mode: filemode.Regular, Hash: h})
	trTo := storeObjTree(t, st, TreeEntry{Name: "b.txt", Mode: filemode.Regular, Hash: h})
	ren := &Change{
		From: ChangeEntry{Name: "a.txt", Tree: trFrom,
			TreeEntry: TreeEntry{Name: "a.txt", Mode: filemode.Regular, Hash: h}},
		To: ChangeEntry{Name: "b.txt", Tree: trTo,
			TreeEntry: TreeEntry{Name: "b.txt", Mode: filemode.Regular, Hash: h}},
	}

	p, err := getPatch("", add, del, ren)
	if err != nil {
		t.Fatal(err)
	}
	stats := p.Stats()
	if len(stats) != 3 {
		t.Fatalf("stats rows = %d", len(stats))
	}
	names := map[string]bool{}
	for _, s := range stats {
		names[s.Name] = true
	}
	for _, want := range []string{"new.txt", "old.txt", "a.txt => b.txt"} {
		if !names[want] {
			t.Fatalf("missing stat row %q in %+v", want, stats)
		}
	}
}

// 10. Additions and deletions count newline characters, plus one when
//     the chunk doesn't end in a newline — an unterminated last line
//     counts. Inferable: no — assert the counting rule's observable
//     effect, not internals.
func TestDetail10(t *testing.T) {
	st := patchStore(t)

	// Two added lines, last unterminated → 1 newline + 1 partial = 2.
	add := &Change{To: fileChange(t, st, "n.txt", "", "x\ny").To}
	p, err := getPatch("", add)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Stats()[0].Addition; got != 2 {
		t.Fatalf("unterminated two-line add counted %d, want 2", got)
	}

	// Same two lines terminated → 2 newlines, same total.
	add2 := &Change{To: fileChange(t, st, "n.txt", "", "x\ny\n").To}
	p, err = getPatch("", add2)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Stats()[0].Addition; got != 2 {
		t.Fatalf("terminated two-line add counted %d, want 2", got)
	}

	// Single unterminated line → 0 newlines + 1 = 1.
	add3 := &Change{To: fileChange(t, st, "n.txt", "", "z").To}
	p, err = getPatch("", add3)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Stats()[0].Addition; got != 1 {
		t.Fatalf("unterminated single-line add counted %d, want 1", got)
	}
}

// 11. printStat scales the +/- graph linearly only when a row's total
//     exceeds the width cap — `1 + it*(width-1)/max`, so nonzero counts
//     always show at least one mark. Inferable: doc.
func TestDetail11(t *testing.T) {
	// Under the cap: marks equal the literal counts.
	small := FileStats{{Name: "f", Addition: 2, Deletion: 1}}.String()
	if strings.Count(small, "+") != 2 || strings.Count(small, "-") != 1 {
		t.Fatalf("unscaled row = %q", small)
	}
	if !strings.Contains(small, "| 3 ") {
		t.Fatalf("total column missing: %q", small)
	}

	// Over the cap (total 104 > 53): scaled by 1 + it*52/104 →
	// 51 '+' and 3 '-'; each nonzero component keeps ≥1 mark.
	big := FileStats{{Name: "f", Addition: 100, Deletion: 4}}.String()
	plus := strings.Count(big, "+")
	minus := strings.Count(big, "-")
	if plus != 51 || minus != 3 {
		t.Fatalf("scaled row: %d '+' %d '-' in %q", plus, minus, big)
	}
}

// 12. Stats rows align name and change-count columns to the widest
//     entry, with the graph starting after a ` | ` separator.
//     Inferable: no — assert column alignment shape, not exact layout.
func TestDetail12(t *testing.T) {
	s := FileStats{
		{Name: "a", Addition: 1, Deletion: 0},
		{Name: "a_much_longer_name", Addition: 10, Deletion: 5},
	}.String()
	rows := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(rows) != 2 {
		t.Fatalf("rows = %d in %q", len(rows), s)
	}
	col := -1
	for i, r := range rows {
		j := strings.Index(r, "|")
		if j < 0 {
			t.Fatalf("row %d lacks separator: %q", i, r)
		}
		if col < 0 {
			col = j
		} else if j != col {
			t.Fatalf("misaligned separators at cols %d and %d", col, j)
		}
		// After the separator: change count, then the graph marks.
		rest := strings.TrimSpace(r[j+1:])
		if rest == "" {
			t.Fatalf("row %d empty after separator", i)
		}
	}
}

// 13. Patch.String returns the same bytes Encode produces for a
//     well-formed patch; its internal buffer cannot fail, so the
//     malformed-patch branch is unreachable through the public surface.
//     Inferable: no — assert the reachable equivalence only.
func TestDetail13(t *testing.T) {
	st := patchStore(t)
	c := fileChange(t, st, "f.txt", "a\n", "b\n")
	p, err := getPatch("", c)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := p.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	if p.String() != buf.String() {
		t.Fatal("String() diverges from Encode output")
	}
	if !strings.HasPrefix(p.String(), "diff --git ") {
		t.Fatalf("encoded patch lacks header: %q", p.String())
	}
}

// 14. Patch.Encode always uses the default three context lines through
//     the unified encoder. Inferable: partially.
func TestDetail14(t *testing.T) {
	st := patchStore(t)
	nine := "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\n"
	chg := strings.Replace(nine, "l5\n", "X5\n", 1)
	c := fileChange(t, st, "f.txt", nine, chg)
	p, err := getPatch("", c)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := p.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// A mid-file change in a 9-line file with 3 context lines covers
	// exactly lines 2..8 on both sides.
	if !strings.Contains(out, "@@ -2,7 +2,7 @@") {
		t.Fatalf("no 3-context hunk header in:\n%s", out)
	}
}
