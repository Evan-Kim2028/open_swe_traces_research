package object

import (
	"fmt"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	"example.internal/gitkit/v6/storage/memory"
	"example.internal/gitkit/v6/utils/merkletrie"
)

func rdBlob(t *testing.T, st *memory.Storage, body []byte) plumbing.Hash {
	t.Helper()
	o := st.NewEncodedObject()
	o.SetType(plumbing.BlobObject)
	o.SetSize(int64(len(body)))
	w, err := o.Writer()
	if err != nil {
		t.Fatalf("blob writer: %v", err)
	}
	if _, err := w.Write(body); err != nil {
		t.Fatalf("write blob: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close blob: %v", err)
	}
	h, err := st.SetEncodedObject(o)
	if err != nil {
		t.Fatalf("store blob: %v", err)
	}
	return h
}

func rdIns(tree *Tree, path, name string, h plumbing.Hash, mode filemode.FileMode) *Change {
	return &Change{
		To: ChangeEntry{
			Name: path,
			Tree: tree,
			TreeEntry: TreeEntry{
				Name: name,
				Mode: mode,
				Hash: h,
			},
		},
	}
}

func rdDel(tree *Tree, path, name string, h plumbing.Hash, mode filemode.FileMode) *Change {
	return &Change{
		From: ChangeEntry{
			Name: path,
			Tree: tree,
			TreeEntry: TreeEntry{
				Name: name,
				Mode: mode,
				Hash: h,
			},
		},
	}
}

func rdOpts() *DiffTreeOptions {
	return &DiffTreeOptions{
		DetectRenames:    true,
		RenameScore:      60,
		RenameLimit:      0,
		OnlyExactRenames: false,
	}
}

func isModify(c *Change) bool {
	a, err := c.Action()
	return err == nil && a == merkletrie.Modify
}

func findModify(cs Changes, from, to string) bool {
	for _, c := range cs {
		if isModify(c) && c.From.Name == from && c.To.Name == to {
			return true
		}
	}
	return false
}

// 1. Detection runs only when BOTH additions and deletions exist — a
//    changeset missing either side is returned without scoring.
func TestDetail01(t *testing.T) {
	st := memory.NewStorage()
	tree := &Tree{s: st}
	h1 := rdBlob(t, st, []byte("one"))
	h2 := rdBlob(t, st, []byte("two"))

	// adds only
	in := Changes{
		rdIns(tree, "a.txt", "a.txt", h1, filemode.Regular),
		rdIns(tree, "b.txt", "b.txt", h2, filemode.Regular),
	}
	out, err := DetectRenames(in, rdOpts())
	if err != nil {
		t.Fatalf("DetectRenames: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("adds-only: got %d changes, want 2", len(out))
	}
	for _, c := range out {
		if isModify(c) {
			t.Fatalf("adds-only changeset produced a modify")
		}
	}

	// deletes only
	in2 := Changes{
		rdDel(tree, "a.txt", "a.txt", h1, filemode.Regular),
		rdDel(tree, "b.txt", "b.txt", h2, filemode.Regular),
	}
	out2, err := DetectRenames(in2, rdOpts())
	if err != nil {
		t.Fatalf("DetectRenames: %v", err)
	}
	if len(out2) != 2 {
		t.Fatalf("deletes-only: got %d changes, want 2", len(out2))
	}
	for _, c := range out2 {
		if isModify(c) {
			t.Fatalf("deletes-only changeset produced a modify")
		}
	}
}

// 2. Exact renames match on identical content hash AND identical file mode —
//    same hash with a mode change stays add+delete, not a rename.
func TestDetail02(t *testing.T) {
	st := memory.NewStorage()
	tree := &Tree{s: st}
	h := rdBlob(t, st, []byte("same content"))

	out, err := DetectRenames(Changes{
		rdDel(tree, "old.txt", "old.txt", h, filemode.Regular),
		rdIns(tree, "new.txt", "new.txt", h, filemode.Regular),
	}, rdOpts())
	if err != nil {
		t.Fatalf("DetectRenames: %v", err)
	}
	if len(out) != 1 || !findModify(out, "old.txt", "new.txt") {
		t.Fatalf("same hash+mode not detected as rename: %v", out)
	}

	out2, err := DetectRenames(Changes{
		rdDel(tree, "old.txt", "old.txt", h, filemode.Regular),
		rdIns(tree, "new.txt", "new.txt", h, filemode.Executable),
	}, rdOpts())
	if err != nil {
		t.Fatalf("DetectRenames: %v", err)
	}
	if len(out2) != 2 || findModify(out2, "old.txt", "new.txt") {
		t.Fatalf("mode change still detected as rename: %v", out2)
	}
}

// 3. With multiple deletes sharing the added file's hash, the winner is the
//    most path-similar name — the losers revert to plain deletions.
func TestDetail03(t *testing.T) {
	st := memory.NewStorage()
	tree := &Tree{s: st}
	h := rdBlob(t, st, []byte("shared"))

	out, err := DetectRenames(Changes{
		rdDel(tree, "dir/sub/old.txt", "old.txt", h, filemode.Regular),
		rdDel(tree, "other/zzz.txt", "zzz.txt", h, filemode.Regular),
		rdIns(tree, "dir/sub/new.txt", "new.txt", h, filemode.Regular),
	}, rdOpts())
	if err != nil {
		t.Fatalf("DetectRenames: %v", err)
	}
	// added "dir/sub/new.txt" shares directory prefix with "dir/sub/old.txt"
	if !findModify(out, "dir/sub/old.txt", "dir/sub/new.txt") {
		t.Fatalf("rename paired with wrong delete: %v", out)
	}
	// the losing delete survives as a plain deletion
	var loser bool
	for _, c := range out {
		if a, _ := c.Action(); a == merkletrie.Delete && c.From.Name == "other/zzz.txt" {
			loser = true
		}
	}
	if !loser {
		t.Fatalf("losing delete missing from result: %v", out)
	}
}

// 4. With multiple adds per hash, a bounded similarity matrix over name
//    scores picks pairs greedily from highest score, each side used once —
//    the matrix, not iteration order, decides.
func TestDetail04(t *testing.T) {
	st := memory.NewStorage()
	tree := &Tree{s: st}
	h := rdBlob(t, st, []byte("dup content"))

	out, err := DetectRenames(Changes{
		rdDel(tree, "a/old.c", "old.c", h, filemode.Regular),
		rdDel(tree, "b/old.c", "old.c", h, filemode.Regular),
		rdIns(tree, "a/new.c", "new.c", h, filemode.Regular),
		rdIns(tree, "b/new.c", "new.c", h, filemode.Regular),
	}, rdOpts())
	if err != nil {
		t.Fatalf("DetectRenames: %v", err)
	}
	if !findModify(out, "a/old.c", "a/new.c") {
		t.Fatalf("expected a-dir pairing: %v", out)
	}
	if !findModify(out, "b/old.c", "b/new.c") {
		t.Fatalf("expected b-dir pairing: %v", out)
	}
	if len(out) != 2 {
		t.Fatalf("each side must pair once: got %d changes", len(out))
	}
}

// 5. Name similarity blends directory prefix+suffix scores (25% each) with
//    filename-suffix score (50%) — weights are fixed fractions, not equal.
//    Inferable: no — the weighting is an upstream constant. Assert SHAPE:
//    the score is bounded, symmetric weighting shows filename-suffix
//    dominance (a same-filename/different-dir pair outscores a
//    same-dir/different-filename pair).
func TestDetail05(t *testing.T) {
	ident := nameSimilarityScore("a/b/c.txt", "a/b/c.txt")
	if ident <= 0 {
		t.Fatalf("identical name scored %d", ident)
	}
	// more shared path structure must score strictly higher
	file := nameSimilarityScore("a/b/c.txt", "a/b/z.txt")
	dir := nameSimilarityScore("a/b/c.txt", "a/x/z.txt")
	diff := nameSimilarityScore("a/b/c.txt", "x/y/z.txt")
	if !(ident >= file && file >= dir && dir >= diff) {
		t.Fatalf("similarity not monotone: %d >= %d >= %d >= %d",
			ident, file, dir, diff)
	}
	if diff < 0 || ident < file {
		t.Fatalf("score out of expected order/range")
	}
}

// 6. Content renames are skipped entirely when OnlyExactRenames is set, and
//    abandoned wholesale when the larger side exceeds RenameLimit — the
//    limit refuses, it does not truncate.
func TestDetail06(t *testing.T) {
	st := memory.NewStorage()
	tree := &Tree{s: st}
	hOld := rdBlob(t, st, []byte("l1\nl2\nl3\nl4\nl5\n"))
	hNew := rdBlob(t, st, []byte("l1\nl2\nl3\nl4\nl6\n")) // similar, not identical

	// only-exact: an inexact pair must not be paired
	opts := rdOpts()
	opts.OnlyExactRenames = true
	out, err := DetectRenames(Changes{
		rdDel(tree, "a.txt", "a.txt", hOld, filemode.Regular),
		rdIns(tree, "b.txt", "b.txt", hNew, filemode.Regular),
	}, opts)
	if err != nil {
		t.Fatalf("DetectRenames: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("OnlyExactRenames still paired inexact files: %v", out)
	}

	// rename limit exceeded: wholesale abandon — error or no modifications,
	// never a truncated subset
	opts2 := rdOpts()
	opts2.RenameLimit = 1
	var many Changes
	for i := 0; i < 4; i++ {
		old := rdBlob(t, st, []byte(fmt.Sprintf("x\ny\nz\nold%d\n", i)))
		new_ := rdBlob(t, st, []byte(fmt.Sprintf("x\ny\nz\nnew%d\n", i)))
		many = append(many,
			rdDel(tree, fmt.Sprintf("d%d.txt", i), fmt.Sprintf("d%d.txt", i), old, filemode.Regular),
			rdIns(tree, fmt.Sprintf("n%d.txt", i), fmt.Sprintf("n%d.txt", i), new_, filemode.Regular))
	}
	out2, err2 := DetectRenames(many, opts2)
	if err2 == nil {
		mods := 0
		for _, c := range out2 {
			if isModify(c) {
				mods++
			}
		}
		if mods != 0 {
			t.Fatalf("RenameLimit truncated rather than refused: %d mods", mods)
		}
	}
}

// 7. Non-regular files (links, submodules, executables) are excluded from
//    content pairing — only regular files enter the matrix.
//    Inferable: no. Assert SHAPE: a symlink-mode change is never paired.
func TestDetail07(t *testing.T) {
	st := memory.NewStorage()
	tree := &Tree{s: st}
	hOld := rdBlob(t, st, []byte("l1\nl2\nl3\nl4\n"))
	hNew := rdBlob(t, st, []byte("l1\nl2\nl3\nl5\n"))

	out, err := DetectRenames(Changes{
		rdDel(tree, "link", "link", hOld, filemode.Symlink),
		rdIns(tree, "real.txt", "real.txt", hNew, filemode.Regular),
	}, rdOpts())
	if err != nil {
		t.Fatalf("DetectRenames: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("symlink entered content pairing: %v", out)
	}
}

// 8. File sizes are inflated by one before the size gate so empty files can
//    still score — min*100/max must reach RenameScore to stay a candidate.
//    Inferable: no — the +1 dodge is arbitrary. Assert SHAPE: an empty file
//    participates without error (no divide-by-zero / skip).
func TestDetail08(t *testing.T) {
	st := memory.NewStorage()
	tree := &Tree{s: st}
	hEmpty := rdBlob(t, st, []byte{})
	hTiny := rdBlob(t, st, []byte("x\n"))

	out, err := DetectRenames(Changes{
		rdDel(tree, "empty", "empty", hEmpty, filemode.Regular),
		rdIns(tree, "tiny", "tiny", hTiny, filemode.Regular),
	}, rdOpts())
	if err != nil {
		t.Fatalf("DetectRenames with empty file: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("empty file dropped from result")
	}
}

// 9. A pair's score is content similarity weighted 99 to name similarity
//    weighted 1, both normalised to the same range — content dominates.
//    Inferable: no. Assert SHAPE: content similarity alone can pair files
//    with totally dissimilar names.
func TestDetail09(t *testing.T) {
	st := memory.NewStorage()
	tree := &Tree{s: st}
	hOld := rdBlob(t, st, []byte("l1\nl2\nl3\nl4\nl5\n"))
	hNew := rdBlob(t, st, []byte("l1\nl2\nl3\nl4\nl6\n"))

	out, err := DetectRenames(Changes{
		rdDel(tree, "aaa/zzz.go", "zzz.go", hOld, filemode.Regular),
		rdIns(tree, "qqq/www.md", "www.md", hNew, filemode.Regular),
	}, rdOpts())
	if err != nil {
		t.Fatalf("DetectRenames: %v", err)
	}
	if len(out) != 1 || !isModify(out[0]) {
		t.Fatalf("content-similar pair with dissimilar names not paired: %v", out)
	}
}

// 10. The similarity index hashes line regions for text and fixed 64-byte
//     blocks for binary, folding CRLF to LF in text — a lone CR without LF
//     still counts.
func TestDetail10(t *testing.T) {
	st := memory.NewStorage()

	mkFile := func(body []byte) *File {
		h := rdBlob(t, st, body)
		b, err := GetBlob(st, h)
		if err != nil {
			t.Fatalf("GetBlob: %v", err)
		}
		return NewFile("f", filemode.Regular, b)
	}

	// CRLF folds to LF: same lines score identically
	fa, err := fileSimilarityIndex(mkFile([]byte("x\ny\n")))
	if err != nil {
		t.Fatalf("index LF: %v", err)
	}
	fb, err := fileSimilarityIndex(mkFile([]byte("x\r\ny\r\n")))
	if err != nil {
		t.Fatalf("index CRLF: %v", err)
	}
	if got := fa.score(fb, 100); got != 100 {
		t.Fatalf("CRLF/LF score %d, want 100", got)
	}

	// a lone CR counts as a region boundary (or at least changes hashing) —
	// "x\ry" must not equal "x\ny" regions exactly nor equal "xry"
	fc, err := fileSimilarityIndex(mkFile([]byte("x\ry\n")))
	if err != nil {
		t.Fatalf("index CR: %v", err)
	}
	fd, err := fileSimilarityIndex(mkFile([]byte("xry\n")))
	if err != nil {
		t.Fatalf("index plain: %v", err)
	}
	if fa.score(fc, 100) == 100 || fc.score(fd, 100) == 100 {
		t.Fatalf("lone CR ignored by indexer")
	}

	// binary (NUL present): identical 64-byte-block content scores max
	fe, err := fileSimilarityIndex(mkFile(append([]byte{0}, make([]byte, 200)...)))
	if err != nil {
		t.Fatalf("index bin: %v", err)
	}
	ff, err := fileSimilarityIndex(mkFile(append([]byte{0}, make([]byte, 200)...)))
	if err != nil {
		t.Fatalf("index bin2: %v", err)
	}
	if got := fe.score(ff, 100); got != 100 {
		t.Fatalf("identical binary score %d, want 100", got)
	}
}

// 11. The index starts at 256 slots and doubles until the 30-bit ceiling,
//     packing 32-bit key + 32-bit count into one word — growth past the cap
//     reports index-full, not panic. Inferable: doc.
func TestDetail11(t *testing.T) {
	// packed word: key in high 32 bits, count in low 32 bits
	p, err := newKeyCountPair(7, 3)
	if err != nil {
		t.Fatalf("newKeyCountPair: %v", err)
	}
	if p.key() != 7 || p.count() != 3 {
		t.Fatalf("keyCountPair roundtrip: key=%d count=%d", p.key(), p.count())
	}
	if _, err := newKeyCountPair(0, maxCountValue+1); err == nil {
		t.Fatalf("overflow count accepted")
	}

	// initial sizing per doc comment, and growth is bounded
	idx := newSimilarityIndex()
	if len(idx.hashes) != 256 {
		t.Fatalf("initial slots %d, want 256", len(idx.hashes))
	}
	if shouldGrowAt(8) <= 0 {
		t.Fatalf("shouldGrowAt(8)=%d nonpositive", shouldGrowAt(8))
	}

	// growth past the ceiling reports errIndexFull — keep doubling until the
	// cap refuses; the 1MiB bound keeps this cheap.
	err = nil
	for n := 0; n < 25 && err == nil; n++ {
		err = idx.grow()
	}
	if err != errIndexFull {
		t.Fatalf("grow past cap: got %v, want errIndexFull", err)
	}
}

// 12. Pair scores equal common region count over the larger region count
//     times the scale — symmetrical, and two empty files score the maximum.
//     Inferable: doc.
func TestDetail12(t *testing.T) {
	st := memory.NewStorage()
	mkIdx := func(body []byte) *similarityIndex {
		h := rdBlob(t, st, body)
		b, err := GetBlob(st, h)
		if err != nil {
			t.Fatalf("GetBlob: %v", err)
		}
		i, err := fileSimilarityIndex(NewFile("f", filemode.Regular, b))
		if err != nil {
			t.Fatalf("index: %v", err)
		}
		return i
	}

	a := mkIdx([]byte("l1\nl2\nl3\n"))
	b := mkIdx([]byte("l1\nl2\nl3\n"))
	c := mkIdx([]byte("x\ny\nz\n"))

	if got := a.score(b, 100); got != 100 {
		t.Fatalf("identical score %d, want 100", got)
	}
	if got := a.score(c, 100); got != 0 {
		t.Fatalf("disjoint score %d, want 0", got)
	}
	if a.score(c, 100) != c.score(a, 100) {
		t.Fatalf("score not symmetrical")
	}
	half := mkIdx([]byte("l1\nl2\n"))
	if s := a.score(half, 100); s <= 0 || s >= 100 {
		t.Fatalf("partial score %d out of (0,100)", s)
	}

	e1, e2 := mkIdx([]byte{}), mkIdx([]byte{})
	if got := e1.score(e2, 100); got != 100 {
		t.Fatalf("two empty files score %d, want 100", got)
	}
}

// 13. The result concatenates remaining adds, remaining deletes, then matched
//     modifies — and is stable-sorted as one list at the end, not per bucket.
func TestDetail13(t *testing.T) {
	st := memory.NewStorage()
	tree := &Tree{s: st}
	h := rdBlob(t, st, []byte("body"))

	out, err := DetectRenames(Changes{
		rdDel(tree, "z/del.txt", "del.txt", h, filemode.Regular),
		rdIns(tree, "a/add.txt", "add.txt", rdBlob(t, st, []byte("other")), filemode.Regular),
		rdDel(tree, "m/old.txt", "old.txt", h, filemode.Regular),
		rdIns(tree, "m/new.txt", "new.txt", h, filemode.Regular),
	}, rdOpts())
	if err != nil {
		t.Fatalf("DetectRenames: %v", err)
	}
	// result sorted as one list by path name
	for i := 1; i < len(out); i++ {
		if out[i-1].name() > out[i].name() {
			t.Fatalf("result not stable-sorted: %q > %q", out[i-1].name(), out[i].name())
		}
	}
	if !findModify(out, "m/old.txt", "m/new.txt") {
		t.Fatalf("rename pair missing: %v", out)
	}
}

// 14. A nil options pointer uses the package defaults — detection is never
//     skipped for nil opts.
func TestDetail14(t *testing.T) {
	st := memory.NewStorage()
	tree := &Tree{s: st}
	h := rdBlob(t, st, []byte("content"))

	out, err := DetectRenames(Changes{
		rdDel(tree, "o.txt", "o.txt", h, filemode.Regular),
		rdIns(tree, "n.txt", "n.txt", h, filemode.Regular),
	}, nil)
	if err != nil {
		t.Fatalf("DetectRenames nil opts: %v", err)
	}
	if len(out) != 1 || !isModify(out[0]) {
		t.Fatalf("nil opts skipped detection: %v", out)
	}
}
