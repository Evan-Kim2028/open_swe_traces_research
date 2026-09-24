package object

import (
	"errors"
	"slices"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	"example.internal/gitkit/v6/storage/memory"
)

func mbBlob(t *testing.T, st *memory.Storage, body string) plumbing.Hash {
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

func mbTree(t *testing.T, st *memory.Storage) plumbing.Hash {
	t.Helper()
	o := st.NewEncodedObject()
	o.SetType(plumbing.TreeObject)
	ents := []TreeEntry{
		{Name: "f", Mode: filemode.Regular, Hash: mbBlob(t, st, "x")},
	}
	if err := (&Tree{Entries: ents}).Encode(o); err != nil {
		t.Fatalf("encode tree: %v", err)
	}
	h, err := st.SetEncodedObject(o)
	if err != nil {
		t.Fatalf("store tree: %v", err)
	}
	return h
}

var mbT = func(n int) time.Time { return time.Date(2020, 1, 1, 0, 0, n, 0, time.UTC) }

func mbCommit(t *testing.T, st *memory.Storage, tree plumbing.Hash, when time.Time, parents ...plumbing.Hash) *Commit {
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
	c, err := GetCommit(st, h)
	if err != nil {
		t.Fatalf("GetCommit: %v", err)
	}
	return c
}

func mbHashes(cs []*Commit) []plumbing.Hash {
	var out []plumbing.Hash
	for _, c := range cs {
		out = append(out, c.Hash)
	}
	return out
}

// mbFork: A <- B <- C (main) and A <- D <- E (feature). Returns all.
func mbFork(t *testing.T) (*memory.Storage, *Commit, *Commit, *Commit, *Commit, *Commit) {
	st := memory.NewStorage()
	tr := mbTree(t, st)
	a := mbCommit(t, st, tr, mbT(1))
	b := mbCommit(t, st, tr, mbT(2), a.Hash)
	c := mbCommit(t, st, tr, mbT(5), b.Hash)
	d := mbCommit(t, st, tr, mbT(3), a.Hash)
	e := mbCommit(t, st, tr, mbT(4), d.Hash)
	return st, a, b, c, d, e
}

// 1. Both commits are sorted committer-date DESCENDING first — the newer
//    history is indexed, the older one walked — and this ordering is an
//    optimisation, not semantics.
func TestDetail01(t *testing.T) {
	_, a, _, c, _, e := mbFork(t)

	sorted := sortByCommitDateDesc(a, c, e)
	if sorted[0].Hash != c.Hash || sorted[2].Hash != a.Hash {
		t.Fatalf("not sorted desc: %v", mbHashes(sorted))
	}

	// semantics identical in either arg order
	mb1, err := c.MergeBase(e)
	if err != nil {
		t.Fatalf("MergeBase: %v", err)
	}
	mb2, err := e.MergeBase(c)
	if err != nil {
		t.Fatalf("MergeBase reversed: %v", err)
	}
	if !slices.Equal(mbHashes(mb1), mbHashes(mb2)) {
		t.Fatalf("arg order changed result: %v vs %v", mb1, mb2)
	}
}

// 2. If the older commit is reachable from the newer's history, the answer
//    is exactly the older commit — a single-element result, no further
//    filtering.
func TestDetail02(t *testing.T) {
	_, a, _, c, _, _ := mbFork(t)

	mbs, err := c.MergeBase(a)
	if err != nil {
		t.Fatalf("MergeBase: %v", err)
	}
	if len(mbs) != 1 || mbs[0].Hash != a.Hash {
		t.Fatalf("reachable ancestor: got %v, want [%s]", mbHashes(mbs), a.Hash)
	}

	// and in the other arg order
	mbs2, err := a.MergeBase(c)
	if err != nil {
		t.Fatalf("MergeBase: %v", err)
	}
	if len(mbs2) != 1 || mbs2[0].Hash != a.Hash {
		t.Fatalf("reversed reachable ancestor: %v", mbHashes(mbs2))
	}
}

// 3. Reachability is detected by walking the newer commit's whole ancestor
//    set and erroring the moment the older one appears — the walk aborts,
//    it does not finish collecting.
func TestDetail03(t *testing.T) {
	_, a, _, c, _, _ := mbFork(t)

	// ancestorsIndex reports reachability via the sentinel
	_, err := ancestorsIndex(a, c)
	if !errors.Is(err, errIsReachable) {
		t.Fatalf("ancestorsIndex: got %v, want errIsReachable", err)
	}

	// and returns the full ancestor set when excluded is not reachable
	_, _, _, _, d, e := mbFork(t)
	_ = d
	idx, err := ancestorsIndex(e, c)
	if err != nil {
		t.Fatalf("ancestorsIndex unrelated: %v", err)
	}
	if _, ok := idx[a.Hash]; !ok {
		t.Fatalf("ancestor index missing shared base")
	}
}

// 4. Merge-base candidates are the older commit's ancestors that also sit
//    in the newer's ancestor index — found via a filtered walk where the
//    index gates BOTH the visit and the descent.
func TestDetail04(t *testing.T) {
	_, a, b, c, _, e := mbFork(t)

	mbs, err := c.MergeBase(e)
	if err != nil {
		t.Fatalf("MergeBase: %v", err)
	}
	if len(mbs) != 1 || mbs[0].Hash != a.Hash {
		t.Fatalf("fork merge base: got %v, want [%s]", mbHashes(mbs), a.Hash)
	}
	// B is on c's side only — never a merge base of c and e
	if slices.Contains(mbHashes(mbs), b.Hash) {
		t.Fatalf("non-common ancestor returned")
	}
}

// 5. Independents drops any candidate reachable from another candidate —
//    each round walks one candidate's history and evicts matches, stopping
//    early when a single candidate remains.
func TestDetail05(t *testing.T) {
	_, a, _, c, d, e := mbFork(t)

	// a chain: only the tip is independent
	out, err := Independents([]*Commit{a, d, e})
	if err != nil {
		t.Fatalf("Independents: %v", err)
	}
	if len(out) != 1 || out[0].Hash != e.Hash {
		t.Fatalf("chain independents: %v", mbHashes(out))
	}

	// divergent tips are both independent
	out2, err := Independents([]*Commit{c, e})
	if err != nil {
		t.Fatalf("Independents: %v", err)
	}
	if len(out2) != 2 {
		t.Fatalf("divergent independents: %v", mbHashes(out2))
	}
}

// 6. Candidates are processed newest-first and duplicates are removed by
//    hash BEFORE any walking — identical input hashes can never evict each
//    other.
func TestDetail06(t *testing.T) {
	_, a, _, c, _, _ := mbFork(t)

	// duplicate hashes dedupe, not evict
	out, err := Independents([]*Commit{c, c})
	if err != nil {
		t.Fatalf("Independents: %v", err)
	}
	if len(out) != 1 || out[0].Hash != c.Hash {
		t.Fatalf("duplicates: %v", mbHashes(out))
	}

	// dedup + eviction: [a, c, c] -> [c]
	out2, err := Independents([]*Commit{a, c, c})
	if err != nil {
		t.Fatalf("Independents: %v", err)
	}
	if len(out2) != 1 || out2[0].Hash != c.Hash {
		t.Fatalf("dedup+evict: %v", mbHashes(out2))
	}
}

// 7. The per-round walk is bounded by a seen-set limiter — ancestors of an
//    already-eliminated lineage are never re-traversed.
//    Inferable: no — assert SHAPE: Independents on a shared-history DAG
//    terminates with the correct set.
func TestDetail07(t *testing.T) {
	st := memory.NewStorage()
	tr := mbTree(t, st)
	base := mbCommit(t, st, tr, mbT(1))
	mid := mbCommit(t, st, tr, mbT(2), base.Hash)
	tip1 := mbCommit(t, st, tr, mbT(3), mid.Hash)
	tip2 := mbCommit(t, st, tr, mbT(4), mid.Hash)
	tip3 := mbCommit(t, st, tr, mbT(5), mid.Hash)

	out, err := Independents([]*Commit{tip1, tip2, tip3, mid, base})
	if err != nil {
		t.Fatalf("Independents: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("shared-history independents: %v", mbHashes(out))
	}
	for _, h := range mbHashes(out) {
		if h == mid.Hash || h == base.Hash {
			t.Fatalf("reachable ancestor survived: %v", mbHashes(out))
		}
	}
}

// 8. IsAncestor is a preorder walk that stops at the first hash match —
//    equality IS ancestry (a commit is its own ancestor).
func TestDetail08(t *testing.T) {
	_, a, _, c, _, e := mbFork(t)

	ok, err := a.IsAncestor(c)
	if err != nil || !ok {
		t.Fatalf("IsAncestor(a,c)=%v,%v", ok, err)
	}
	ok, err = c.IsAncestor(a)
	if err != nil || ok {
		t.Fatalf("IsAncestor(c,a)=%v,%v", ok, err)
	}
	ok, err = a.IsAncestor(a)
	if err != nil || !ok {
		t.Fatalf("IsAncestor(a,a)=%v,%v — equality is ancestry", ok, err)
	}
	ok, err = e.IsAncestor(c)
	if err != nil || ok {
		t.Fatalf("IsAncestor(e,c)=%v,%v — divergent", ok, err)
	}
}

// 9. Helper removal/dedup preserve input order and operate by hash
//    equality — not pointer identity, not date.
//    Inferable: no — assert SHAPE on observable order preservation.
func TestDetail09(t *testing.T) {
	_, a, b, c, _, _ := mbFork(t)

	// order preserved
	got := remove([]*Commit{a, b, c}, b)
	if !slices.Equal(mbHashes(got), []plumbing.Hash{a.Hash, c.Hash}) {
		t.Fatalf("remove broke order: %v", mbHashes(got))
	}
	got2 := removeDuplicated([]*Commit{a, c, a, b, c})
	if !slices.Equal(mbHashes(got2), []plumbing.Hash{a.Hash, c.Hash, b.Hash}) {
		t.Fatalf("dedup broke order: %v", mbHashes(got2))
	}

	// hash equality, not pointer identity: a fresh Commit with a's hash
	// still matches
	twin := &Commit{Hash: a.Hash}
	if indexOf([]*Commit{b, c}, twin) != -1 {
		t.Fatalf("indexOf matched unrelated hash")
	}
	if indexOf([]*Commit{b, c, twin}, a) != 2 {
		t.Fatalf("indexOf did not match by hash")
	}
	if !slices.Equal(mbHashes(remove([]*Commit{a, b}, twin)), []plumbing.Hash{b.Hash}) {
		t.Fatalf("remove did not match by hash")
	}
}

// 10. An untraversable history (missing objects) propagates the walker
//     error rather than returning a partial answer.
func TestDetail10(t *testing.T) {
	st := memory.NewStorage()
	tr := mbTree(t, st)
	missing := plumbing.NewHash("3333333333333333333333333333333333333333")
	normal := mbCommit(t, st, tr, mbT(2))
	// the newer commit's history is indexed first — its missing parent must
	// surface as an error
	dangling := mbCommit(t, st, tr, mbT(9), missing)

	if _, err := normal.MergeBase(dangling); err == nil {
		t.Fatalf("untraversable history returned a partial answer")
	}
	if _, err := normal.IsAncestor(dangling); err == nil {
		t.Fatalf("IsAncestor swallowed traversal error")
	}
}
