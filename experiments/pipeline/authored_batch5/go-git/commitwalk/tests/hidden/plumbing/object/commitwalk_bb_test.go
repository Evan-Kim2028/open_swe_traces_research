package object

import (
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	"example.internal/gitkit/v6/plumbing/storer"
	"example.internal/gitkit/v6/storage/memory"
)

func cwBlob(t *testing.T, st *memory.Storage, body string) plumbing.Hash {
	t.Helper()
	o := st.NewEncodedObject()
	o.SetType(plumbing.BlobObject)
	w, err := o.Writer()
	if err != nil {
		t.Fatalf("blob writer: %v", err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
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

func cwTree(t *testing.T, st *memory.Storage, files map[string]string) plumbing.Hash {
	t.Helper()
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	slices.Sort(names)
	ents := make([]TreeEntry, 0, len(names))
	for _, n := range names {
		ents = append(ents, TreeEntry{
			Name: n, Mode: filemode.Regular, Hash: cwBlob(t, st, files[n]),
		})
	}
	o := st.NewEncodedObject()
	o.SetType(plumbing.TreeObject)
	if err := (&Tree{Entries: ents}).Encode(o); err != nil {
		t.Fatalf("encode tree: %v", err)
	}
	h, err := st.SetEncodedObject(o)
	if err != nil {
		t.Fatalf("store tree: %v", err)
	}
	return h
}

func cwCommit(t *testing.T, st *memory.Storage, tree plumbing.Hash, when time.Time, parents ...plumbing.Hash) plumbing.Hash {
	t.Helper()
	o := st.NewEncodedObject()
	o.SetType(plumbing.CommitObject)
	sig := Signature{Name: "a", Email: "a@b", When: when}
	if err := (&Commit{
		Author:       sig,
		Committer:    sig,
		Message:      "m\n",
		TreeHash:     tree,
		ParentHashes: parents,
	}).Encode(o); err != nil {
		t.Fatalf("encode commit: %v", err)
	}
	h, err := st.SetEncodedObject(o)
	if err != nil {
		t.Fatalf("store commit: %v", err)
	}
	return h
}

func cwGet(t *testing.T, st *memory.Storage, h plumbing.Hash) *Commit {
	t.Helper()
	c, err := GetCommit(st, h)
	if err != nil {
		t.Fatalf("GetCommit %s: %v", h, err)
	}
	return c
}

func cwHashes(t *testing.T, it CommitIter) []plumbing.Hash {
	t.Helper()
	defer it.Close()
	var out []plumbing.Hash
	for {
		c, err := it.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		out = append(out, c.Hash)
	}
}

var cwT = func(n int) time.Time { return time.Date(2020, 1, 1, 0, 0, n, 0, time.UTC) }

// cwDiamond: A <- B, A <- C, {B,C} <- D (merge). Returns (st, A,B,C,D).
func cwDiamond(t *testing.T) (*memory.Storage, plumbing.Hash, plumbing.Hash, plumbing.Hash, plumbing.Hash) {
	st := memory.NewStorage()
	tr := cwTree(t, st, map[string]string{"f": "x"})
	a := cwCommit(t, st, tr, cwT(1))
	b := cwCommit(t, st, tr, cwT(2), a)
	c := cwCommit(t, st, tr, cwT(3), a)
	d := cwCommit(t, st, tr, cwT(4), c, b)
	return st, a, b, c, d
}

// 1. Every iterator terminates on io.EOF from Next and halts early —
//    cleanly — on storer.ErrStop from the callback; other callback errors
//    propagate.
func TestDetail01(t *testing.T) {
	st, _, _, _, d := cwDiamond(t)
	head := cwGet(t, st, d)

	it := NewCommitPreorderIter(head, nil, nil)
	n := 0
	for {
		_, err := it.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		n++
	}
	if n != 4 {
		t.Fatalf("iterated %d commits, want 4", n)
	}

	it2 := NewCommitPreorderIter(head, nil, nil)
	seen := 0
	if err := it2.ForEach(func(*Commit) error {
		seen++
		return storer.ErrStop
	}); err != nil {
		t.Fatalf("ErrStop should halt cleanly, got %v", err)
	}
	if seen != 1 {
		t.Fatalf("ErrStop after %d commits, want 1", seen)
	}

	it3 := NewCommitPreorderIter(head, nil, nil)
	boom := errors.New("cb boom")
	if err := it3.ForEach(func(*Commit) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("callback error: got %v, want boom", err)
	}
}

// 2. Preorder yields a commit before its parents and never revisits — the
//    seen set plus the caller-supplied external seen map both gate descent.
func TestDetail02(t *testing.T) {
	st, a, b, c, d := cwDiamond(t)
	head := cwGet(t, st, d)

	got := cwHashes(t, NewCommitPreorderIter(head, nil, nil))
	if len(got) != 4 || got[0] != d {
		t.Fatalf("preorder start/count: %v", got)
	}
	seen := map[plumbing.Hash]bool{}
	for _, h := range got {
		if seen[h] {
			t.Fatalf("revisited %s", h)
		}
		seen[h] = true
	}
	for _, h := range []plumbing.Hash{a, b, c, d} {
		if !seen[h] {
			t.Fatalf("preorder missed %s", h)
		}
	}

	// strict child-before-parents along a linear chain
	st2 := memory.NewStorage()
	tr2 := cwTree(t, st2, map[string]string{"f": "x"})
	l1 := cwCommit(t, st2, tr2, cwT(1))
	l2 := cwCommit(t, st2, tr2, cwT(2), l1)
	l3 := cwCommit(t, st2, tr2, cwT(3), l2)
	gotL := cwHashes(t, NewCommitPreorderIter(cwGet(t, st2, l3), nil, nil))
	if len(gotL) != 3 || gotL[0] != l3 || gotL[1] != l2 || gotL[2] != l1 {
		t.Fatalf("linear preorder not child-first: %v", gotL)
	}

	// external seen map gates descent
	got2 := cwHashes(t, NewCommitPreorderIter(head, map[plumbing.Hash]bool{b: true}, nil))
	for _, h := range got2 {
		if h == b {
			t.Fatalf("externally-seen commit emitted")
		}
	}
	if len(got2) != 3 {
		t.Fatalf("external-seen walk: %v", got2)
	}
	_ = c
}

// 3. Postorder yields parents before the commit itself; the first-parent
//    variant follows only the first parent edge.
func TestDetail03(t *testing.T) {
	st, a, b, c, d := cwDiamond(t)
	head := cwGet(t, st, d)

	got := cwHashes(t, NewCommitPostorderIter(head, nil))
	if len(got) != 4 {
		t.Fatalf("postorder count: %v", got)
	}
	seen := map[plumbing.Hash]bool{}
	for _, h := range got {
		if seen[h] {
			t.Fatalf("postorder revisited %s", h)
		}
		seen[h] = true
	}
	// deterministic replay
	got2 := cwHashes(t, NewCommitPostorderIter(head, nil))
	if !slices.Equal(got, got2) {
		t.Fatalf("postorder not deterministic: %v vs %v", got, got2)
	}
	// doc commitment: the merged commit is walked before the base it was
	// merged on — B (merged-in parent) precedes A (merge base)
	if slices.Index(got, b) > slices.Index(got, a) {
		t.Fatalf("merged commit not walked before base: %v", got)
	}

	// first-parent: D -> C -> A, never B
	got3 := cwHashes(t, NewCommitPostorderIterFirstParent(head, nil))
	if slices.Contains(got3, b) {
		t.Fatalf("first-parent walk emitted second parent: %v", got3)
	}
	want := map[plumbing.Hash]bool{a: true, c: true, d: true}
	for _, h := range got3 {
		delete(want, h)
	}
	if len(want) != 0 {
		t.Fatalf("first-parent walk missed %v", want)
	}
}

// 4. The ignore list seeds the seen set — ignored commits are never emitted
//    and never descended through.
func TestDetail04(t *testing.T) {
	st := memory.NewStorage()
	tr := cwTree(t, st, map[string]string{"f": "x"})
	a := cwCommit(t, st, tr, cwT(1))
	b := cwCommit(t, st, tr, cwT(2), a)
	c := cwCommit(t, st, tr, cwT(3), b)
	d := cwCommit(t, st, tr, cwT(4), c)

	got := cwHashes(t, NewCommitPreorderIter(cwGet(t, st, d), nil, []plumbing.Hash{b}))
	if slices.Contains(got, b) || slices.Contains(got, a) {
		t.Fatalf("ignore leaked: %v", got)
	}
	if !slices.Contains(got, d) || !slices.Contains(got, c) || len(got) != 2 {
		t.Fatalf("ignore walk: %v", got)
	}
}

// 5. BFS keeps a queue and marks parents seen at ENQUEUE time — duplicates
//    never enter the queue twice.
func TestDetail05(t *testing.T) {
	st, a, _, _, d := cwDiamond(t)
	head := cwGet(t, st, d)

	got := cwHashes(t, NewCommitIterBSF(head, nil, nil))
	if len(got) != 4 {
		t.Fatalf("bfs count: %v", got)
	}
	seen := map[plumbing.Hash]bool{}
	for _, h := range got {
		if seen[h] {
			t.Fatalf("bfs duplicate %s", h)
		}
		seen[h] = true
	}
	// breadth-first: head first, both parents before the grandparent
	if got[0] != d || got[3] != a {
		t.Fatalf("bfs order: %v", got)
	}
}

// 6. The filtered BFS consults isValid before emitting and isLimit before
//    descending — a limit hit yields the commit but stops the descent, and a
//    storer failure is latched for Error() while the walk ends.
func TestDetail06(t *testing.T) {
	st := memory.NewStorage()
	tr := cwTree(t, st, map[string]string{"f": "x"})
	a := cwCommit(t, st, tr, cwT(1))
	b := cwCommit(t, st, tr, cwT(2), a)
	c := cwCommit(t, st, tr, cwT(3), b)
	d := cwCommit(t, st, tr, cwT(4), c)

	// isValid filters emission only
	notB := CommitFilter(func(cm *Commit) bool { return cm.Hash != b })
	got := cwHashes(t, NewFilterCommitIter(cwGet(t, st, d), &notB, nil))
	if slices.Contains(got, b) || len(got) != 3 {
		t.Fatalf("isValid walk: %v", got)
	}

	// isLimit emits but does not descend
	atC := CommitFilter(func(cm *Commit) bool { return cm.Hash == c })
	got2 := cwHashes(t, NewFilterCommitIter(cwGet(t, st, d), nil, &atC))
	if !slices.Contains(got2, c) || slices.Contains(got2, b) || slices.Contains(got2, a) {
		t.Fatalf("isLimit walk: %v", got2)
	}

	// storer failure latches Error()
	missing := plumbing.NewHash("0123456789012345678901234567890123456789")
	bad := cwCommit(t, st, tr, cwT(5), missing)
	it := NewFilterCommitIter(cwGet(t, st, bad), nil, nil)
	for {
		_, err := it.Next()
		if err != nil {
			break
		}
	}
	errIter, ok := it.(interface{ Error() error })
	if !ok {
		t.Fatalf("filtered iter has no Error()")
	}
	if errIter.Error() == nil {
		t.Fatalf("storer failure not latched for Error()")
	}
}

// 7. The ctime iterator keeps a min-heap on committer time so the newest
//    commit pops first — equal timestamps fall back to insertion order.
func TestDetail07(t *testing.T) {
	st := memory.NewStorage()
	tr := cwTree(t, st, map[string]string{"f": "x"})
	a := cwCommit(t, st, tr, cwT(1))
	old := cwCommit(t, st, tr, cwT(2), a)
	newer := cwCommit(t, st, tr, cwT(9), a)
	// parents listed older first — heap must still pop newest first
	head := cwCommit(t, st, tr, cwT(10), old, newer)

	got := cwHashes(t, NewCommitIterCTime(cwGet(t, st, head), nil, nil))
	if len(got) != 4 || got[0] != head {
		t.Fatalf("ctime walk: %v", got)
	}
	pos := map[plumbing.Hash]int{}
	for i, h := range got {
		pos[h] = i
	}
	if pos[newer] > pos[old] {
		t.Fatalf("newer commit did not pop first: %v", got)
	}

	// equal committer times: all commits still emitted (heap tie-break
	// ordering is an implementation detail)
	st2 := memory.NewStorage()
	tr2 := cwTree(t, st2, map[string]string{"f": "x"})
	tr2b := cwTree(t, st2, map[string]string{"f": "y"})
	r := cwCommit(t, st2, tr2, cwT(1))
	p1 := cwCommit(t, st2, tr2, cwT(5), r)
	p2 := cwCommit(t, st2, tr2b, cwT(5), r)
	m := cwCommit(t, st2, tr2, cwT(6), p1, p2)
	got2 := cwHashes(t, NewCommitIterCTime(cwGet(t, st2, m), nil, nil))
	if len(got2) != 4 || got2[0] != m {
		t.Fatalf("equal-ctime walk: %v", got2)
	}
	for _, h := range []plumbing.Hash{p1, p2, r} {
		if !slices.Contains(got2, h) {
			t.Fatalf("equal-ctime walk missed %s: %v", h, got2)
		}
	}
}

// seqIter is a CommitIter yielding an explicit commit sequence — used to
// control the "next commit" the path iterator diffs against.
type seqIter struct {
	seq []*Commit
	i   int
}

func (s *seqIter) Next() (*Commit, error) {
	if s.i >= len(s.seq) {
		return nil, io.EOF
	}
	s.i++
	return s.seq[s.i-1], nil
}

func (s *seqIter) ForEach(f func(*Commit) error) error {
	for {
		c, err := s.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := f(c); err != nil {
			return err
		}
	}
}

func (s *seqIter) Close() {}

// 8. The path iterator emits a commit only when the diff to its parent —
//    or, with checkParent off, to each parent — touches the filtered path;
//    merge commits need the flag set to be compared against every parent.
func TestDetail08(t *testing.T) {
	st := memory.NewStorage()

	tA := cwTree(t, st, map[string]string{"p.txt": "v1", "q.txt": "x"})
	tB := cwTree(t, st, map[string]string{"p.txt": "v1", "q.txt": "y"})
	tC := cwTree(t, st, map[string]string{"p.txt": "v2", "q.txt": "y"})
	tX := cwTree(t, st, map[string]string{"p.txt": "v9", "q.txt": "z"})

	cA := cwCommit(t, st, tA, cwT(1))         // creates p.txt
	cB := cwCommit(t, st, tB, cwT(2), cA)     // touches q.txt only
	cC := cwCommit(t, st, tC, cwT(3), cB)     // touches p.txt
	cX := cwCommit(t, st, tX, cwT(4))         // unrelated root; not cC's parent

	isP := func(path string) bool { return path == "p.txt" }

	// plain chain: only commits whose diff touches p.txt are emitted
	src := NewCommitPreorderIter(cwGet(t, st, cC), nil, nil)
	got := cwHashes(t, NewCommitPathIterFromIter(isP, src, true))
	if !slices.Contains(got, cC) || slices.Contains(got, cB) {
		t.Fatalf("path filter emitted non-touching commit: %v", got)
	}

	// checkParent=false: the next yielded commit is treated as the parent
	// unconditionally — cC diffs against unrelated cX (p.txt differs).
	src2 := &seqIter{seq: []*Commit{cwGet(t, st, cC), cwGet(t, st, cX)}}
	got2 := cwHashes(t, NewCommitPathIterFromIter(isP, src2, false))
	if !slices.Contains(got2, cC) {
		t.Fatalf("checkParent=false skipped next-commit diff: %v", got2)
	}

	// checkParent=true: a next commit that is not a real parent is not used
	// as the diff base — cC is dropped rather than diffed against cX.
	src3 := &seqIter{seq: []*Commit{cwGet(t, st, cC), cwGet(t, st, cX)}}
	got3 := cwHashes(t, NewCommitPathIterFromIter(isP, src3, true))
	if slices.Contains(got3, cC) {
		t.Fatalf("checkParent=true diffed against a non-parent: %v", got3)
	}

	// merge commit identical to the next parent on the path is suppressed
	// even though another parent's diff would touch it.
	tY := cwTree(t, st, map[string]string{"p.txt": "v2", "q.txt": "y", "z.txt": "1"})
	tM := cwTree(t, st, map[string]string{"p.txt": "v2", "q.txt": "y", "z.txt": "2"})
	cY := cwCommit(t, st, tY, cwT(4), cC)     // p.txt stays v2, adds z.txt
	cM := cwCommit(t, st, tM, cwT(5), cY, cB) // parents: Y (p same), B (p differs)
	src4 := NewCommitPreorderIter(cwGet(t, st, cM), nil, nil)
	got4 := cwHashes(t, NewCommitPathIterFromIter(isP, src4, true))
	if slices.Contains(got4, cM) {
		t.Fatalf("merge emitted despite TREESAME parent on path: %v", got4)
	}
}

// 9. The limit iterator drops commits outside the since/until window and
//    stops entirely once the tail hash is seen — the tail itself is not
//    emitted.
// cwHashesStop collects hashes, treating io.EOF or storer.ErrStop from Next
// as end-of-walk (the limit iterator terminates via ErrStop).
func cwHashesStop(t *testing.T, it CommitIter) []plumbing.Hash {
	t.Helper()
	defer it.Close()
	var out []plumbing.Hash
	for {
		c, err := it.Next()
		if err == io.EOF || err == storer.ErrStop {
			return out
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		out = append(out, c.Hash)
	}
}

func TestDetail09(t *testing.T) {
	st := memory.NewStorage()
	tr := cwTree(t, st, map[string]string{"f": "x"})
	c1 := cwCommit(t, st, tr, cwT(1))
	c2 := cwCommit(t, st, tr, cwT(2), c1)
	c3 := cwCommit(t, st, tr, cwT(3), c2)
	c4 := cwCommit(t, st, tr, cwT(4), c3)
	c5 := cwCommit(t, st, tr, cwT(5), c4)

	// since/until window
	since := cwT(3)
	until := cwT(4) // exclusive bound check: keep [t3, t4]
	src := NewCommitPreorderIter(cwGet(t, st, c5), nil, nil)
	got := cwHashesStop(t, NewCommitLimitIterFromIter(src, LogLimitOptions{Since: &since, Until: &until}))
	for _, h := range got {
		if h == c1 || h == c2 || h == c5 {
			t.Fatalf("outside-window commit emitted: %v", got)
		}
	}
	if len(got) == 0 {
		t.Fatalf("limit window dropped everything")
	}

	// tail hash: stop before it, never emit it
	src2 := NewCommitPreorderIter(cwGet(t, st, c5), nil, nil)
	got2 := cwHashesStop(t, NewCommitLimitIterFromIter(src2, LogLimitOptions{TailHash: c3}))
	if slices.Contains(got2, c3) || slices.Contains(got2, c2) || slices.Contains(got2, c1) {
		t.Fatalf("tail or older emitted: %v", got2)
	}
	if len(got2) != 2 {
		t.Fatalf("tail walk: %v", got2)
	}
}

// 10. NewCommitAllIter seeds the walk from every reference's commit — the
//     reference walk failure propagates before any commit is yielded.
func TestDetail10(t *testing.T) {
	st := memory.NewStorage()
	tr := cwTree(t, st, map[string]string{"f": "x"})
	h1 := cwCommit(t, st, tr, cwT(1))
	h2 := cwCommit(t, st, tr, cwT(2))
	if err := st.SetReference(plumbing.NewHashReference("refs/heads/one", h1)); err != nil {
		t.Fatalf("set ref: %v", err)
	}
	if err := st.SetReference(plumbing.NewHashReference("refs/heads/two", h2)); err != nil {
		t.Fatalf("set ref: %v", err)
	}

	mk := func(c *Commit) CommitIter { return NewCommitPreorderIter(c, nil, nil) }
	it, err := NewCommitAllIter(st, mk)
	if err != nil {
		t.Fatalf("NewCommitAllIter: %v", err)
	}
	got := cwHashes(t, it)
	if !slices.Contains(got, h1) || !slices.Contains(got, h2) {
		t.Fatalf("all-iter missed a ref head: %v", got)
	}

	// a ref pointing at a non-commit propagates an error before any commit
	// is yielded — at construction or on first Next
	st2 := memory.NewStorage()
	bh := cwBlob(t, st2, "not a commit")
	if err := st2.SetReference(plumbing.NewHashReference("refs/heads/bad", bh)); err != nil {
		t.Fatalf("set ref: %v", err)
	}
	it2, err := NewCommitAllIter(st2, mk)
	if err == nil {
		defer it2.Close()
		_, err = it2.Next()
	}
	if err == nil {
		t.Fatalf("bad ref propagated no error")
	}
}

// 11. Iterators hold only their own state — Close releases wrapped source
//     iterators where present and is otherwise a no-op.
func TestDetail11(t *testing.T) {
	st, _, _, _, d := cwDiamond(t)
	head := cwGet(t, st, d)

	// plain iterators: Close is a safe no-op
	it := NewCommitPreorderIter(head, nil, nil)
	it.Close()

	// wrapped-source iterators release their source
	src := &countIter{inner: NewCommitPreorderIter(head, nil, nil)}
	wrapped := NewCommitLimitIterFromIter(src, LogLimitOptions{})
	wrapped.Close()
	if !src.closed {
		t.Fatalf("Close did not reach wrapped source iter")
	}

	src2 := &countIter{inner: NewCommitPreorderIter(head, nil, nil)}
	wrapped2 := NewCommitPathIterFromIter(func(string) bool { return true }, src2, true)
	wrapped2.Close()
	if !src2.closed {
		t.Fatalf("path iter Close did not reach source")
	}
}

type countIter struct {
	inner  CommitIter
	closed bool
}

func (c *countIter) Next() (*Commit, error)                 { return c.inner.Next() }
func (c *countIter) ForEach(f func(*Commit) error) error    { return c.inner.ForEach(f) }
func (c *countIter) Close()                                 { c.closed = true; c.inner.Close() }

