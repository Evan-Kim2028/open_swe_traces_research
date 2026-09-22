package revlist

import (
	"slices"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	"example.internal/gitkit/v6/plumbing/object"
	"example.internal/gitkit/v6/storage/memory"
)

func rwObj(t *testing.T, st *memory.Storage, typ plumbing.ObjectType, enc func(o plumbing.EncodedObject) error) plumbing.Hash {
	t.Helper()
	o := st.NewEncodedObject()
	o.SetType(typ)
	if err := enc(o); err != nil {
		t.Fatalf("encode %s: %v", typ, err)
	}
	h, err := st.SetEncodedObject(o)
	if err != nil {
		t.Fatalf("store %s: %v", typ, err)
	}
	return h
}

func rwBlob(t *testing.T, st *memory.Storage, body string) plumbing.Hash {
	return rwObj(t, st, plumbing.BlobObject, func(o plumbing.EncodedObject) error {
		w, err := o.Writer()
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(body)); err != nil {
			return err
		}
		return w.Close()
	})
}

func rwTree(t *testing.T, st *memory.Storage, ents ...object.TreeEntry) plumbing.Hash {
	key := func(e object.TreeEntry) string {
		if e.Mode == filemode.Dir {
			return e.Name + "/"
		}
		return e.Name
	}
	slices.SortFunc(ents, func(a, b object.TreeEntry) int {
		if key(a) < key(b) {
			return -1
		}
		return 1
	})
	return rwObj(t, st, plumbing.TreeObject, func(o plumbing.EncodedObject) error {
		return (&object.Tree{Entries: ents}).Encode(o)
	})
}

func rwTreeOf(t *testing.T, st *memory.Storage, files map[string]string) plumbing.Hash {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	slices.Sort(names)
	ents := make([]object.TreeEntry, 0, len(names))
	for _, n := range names {
		ents = append(ents, object.TreeEntry{
			Name: n, Mode: filemode.Regular, Hash: rwBlob(t, st, files[n]),
		})
	}
	return rwTree(t, st, ents...)
}

func rwCommit(t *testing.T, st *memory.Storage, tree plumbing.Hash, when time.Time, parents ...plumbing.Hash) plumbing.Hash {
	sig := object.Signature{Name: "a", Email: "a@b", When: when}
	return rwObj(t, st, plumbing.CommitObject, func(o plumbing.EncodedObject) error {
		return (&object.Commit{
			Author:       sig,
			Committer:    sig,
			Message:      "m\n",
			TreeHash:     tree,
			ParentHashes: parents,
		}).Encode(o)
	})
}

func rwTag(t *testing.T, st *memory.Storage, name string, target plumbing.Hash, when time.Time) plumbing.Hash {
	return rwObj(t, st, plumbing.TagObject, func(o plumbing.EncodedObject) error {
		return (&object.Tag{
			Name:       name,
			Tagger:     object.Signature{Name: "a", Email: "a@b", When: when},
			Message:    "t\n",
			TargetType: plumbing.CommitObject,
			Target:     target,
		}).Encode(o)
	})
}

var rwT = func(n int) time.Time { return time.Date(2020, 1, 1, 0, 0, n, 0, time.UTC) }

func rwWhen(t *testing.T, st *memory.Storage, h plumbing.Hash) time.Time {
	t.Helper()
	c, err := object.GetCommit(st, h)
	if err != nil {
		t.Fatalf("GetCommit %s: %v", h, err)
	}
	return c.Committer.When
}

// 1. Commits are popped newest-first — the queue stays sorted by committer
//    time descending, so a want reachable from both a new and an old tip is
//    visited once.
func TestDetail01(t *testing.T) {
	st := memory.NewStorage()
	tr := rwTreeOf(t, st, map[string]string{"f": "x"})
	a := rwCommit(t, st, tr, rwT(1))
	b := rwCommit(t, st, tr, rwT(2), a)
	c := rwCommit(t, st, tr, rwT(3), a)
	d := rwCommit(t, st, tr, rwT(4), c, b)

	got, err := Objects(st, []plumbing.Hash{d}, nil)
	if err != nil {
		t.Fatalf("Objects: %v", err)
	}
	seen := map[plumbing.Hash]int{}
	var commitSeq []plumbing.Hash
	for _, h := range got {
		seen[h]++
		if seen[h] > 1 {
			t.Fatalf("duplicate object %s in result", h)
		}
		if _, err := object.GetCommit(st, h); err == nil {
			commitSeq = append(commitSeq, h)
		}
	}
	// commits appear newest-committer-time first
	for i := 1; i < len(commitSeq); i++ {
		if rwWhen(t, st, commitSeq[i]).After(rwWhen(t, st, commitSeq[i-1])) {
			t.Fatalf("commits not newest-first: %v", commitSeq)
		}
	}
	if !slices.Contains(got, d) {
		t.Fatalf("want missing from result")
	}
}

// 2. Paint propagates: a commit reached from both the want side and the
//    have side is a boundary, and the walk stops when every queued commit
//    is a boundary — it never walks all history when haves exist.
func TestDetail02(t *testing.T) {
	st := memory.NewStorage()
	trA := rwTreeOf(t, st, map[string]string{"f": "base"})
	trC1 := rwTreeOf(t, st, map[string]string{"f": "c1"})
	trC2 := rwTreeOf(t, st, map[string]string{"f": "c2"})

	a := rwCommit(t, st, trA, rwT(1))
	c1 := rwCommit(t, st, trC1, rwT(2), a) // have-side tip
	c2 := rwCommit(t, st, trC2, rwT(3), a) // want-side tip

	got, err := Objects(st, []plumbing.Hash{c2}, []plumbing.Hash{c1})
	if err != nil {
		t.Fatalf("Objects: %v", err)
	}
	if !slices.Contains(got, c2) {
		t.Fatalf("want tip missing")
	}
	for _, h := range []plumbing.Hash{c1, a} {
		if slices.Contains(got, h) {
			t.Fatalf("have-side commit %s collected — boundary ignored", h)
		}
	}
	// have-side tree objects are not re-sent
	if slices.Contains(got, trC1) || slices.Contains(got, trA) {
		t.Fatalf("have-side objects collected")
	}
}

// 3. A commit painted want-only is tentatively collected, but if havePaint
//    reaches it later it is dropped from the result — the boundary can move
//    after discovery.
func TestDetail03(t *testing.T) {
	st := memory.NewStorage()
	tr := rwTreeOf(t, st, map[string]string{"f": "x"})
	a := rwCommit(t, st, tr, rwT(1))
	b := rwCommit(t, st, tr, rwT(2), a)
	c := rwCommit(t, st, tr, rwT(3), b)
	d := rwCommit(t, st, tr, rwT(4), c)

	got, err := Objects(st, []plumbing.Hash{d}, []plumbing.Hash{b})
	if err != nil {
		t.Fatalf("Objects: %v", err)
	}
	if slices.Contains(got, b) || slices.Contains(got, a) {
		t.Fatalf("have-painted commits in result: %v", got)
	}
	if !slices.Contains(got, d) || !slices.Contains(got, c) {
		t.Fatalf("want commits missing: %v", got)
	}
}

// 4. Haves seeds pre-mark all reachable tree/blob objects as seen — the
//    result contains only objects new relative to the haves, not a full
//    traversal.
func TestDetail04(t *testing.T) {
	st := memory.NewStorage()
	shared := rwBlob(t, st, "shared blob")
	trHave := rwTree(t, st, object.TreeEntry{Name: "shared", Mode: filemode.Regular, Hash: shared})
	trWant := rwTree(t, st,
		object.TreeEntry{Name: "new", Mode: filemode.Regular, Hash: rwBlob(t, st, "new blob")},
		object.TreeEntry{Name: "shared", Mode: filemode.Regular, Hash: shared},
	)
	ch := rwCommit(t, st, trHave, rwT(1))
	cw := rwCommit(t, st, trWant, rwT(2))

	got, err := Objects(st, []plumbing.Hash{cw}, []plumbing.Hash{ch})
	if err != nil {
		t.Fatalf("Objects: %v", err)
	}
	if slices.Contains(got, shared) {
		t.Fatalf("have-reachable blob re-sent")
	}
	if !slices.Contains(got, cw) || !slices.Contains(got, trWant) {
		t.Fatalf("want objects missing: %v", got)
	}
}

// 5. Missing objects on the haves side are tolerated (ErrObjectNotFound
//    skipped); a missing object on the wants side is a hard error.
func TestDetail05(t *testing.T) {
	st := memory.NewStorage()
	tr := rwTreeOf(t, st, map[string]string{"f": "x"})
	c := rwCommit(t, st, tr, rwT(1))
	missing := plumbing.NewHash("0123456789012345678901234567890123456789")

	if _, err := Objects(st, []plumbing.Hash{c}, []plumbing.Hash{missing}); err != nil {
		t.Fatalf("missing have must be tolerated: %v", err)
	}
	if _, err := Objects(st, []plumbing.Hash{missing}, nil); err == nil {
		t.Fatalf("missing want did not error")
	}
}

// 6. A missing parent is only an error if its child was never painted by
//    haves — missing history behind the haves boundary is fine.
//    Inferable: no — assert SHAPE: want-side missing parent errors,
//    have-side missing parent does not.
func TestDetail06(t *testing.T) {
	st := memory.NewStorage()
	tr := rwTreeOf(t, st, map[string]string{"f": "x"})
	missing := plumbing.NewHash("1111111111111111111111111111111111111111")
	dangling := rwCommit(t, st, tr, rwT(1), missing) // parent not in store
	ok := rwCommit(t, st, tr, rwT(2))

	if _, err := Objects(st, []plumbing.Hash{dangling}, nil); err == nil {
		t.Fatalf("want-side missing parent tolerated")
	}
	if _, err := Objects(st, []plumbing.Hash{ok}, []plumbing.Hash{dangling}); err != nil {
		t.Fatalf("have-side missing parent errored: %v", err)
	}
}

// 7. Shallow commits stop parent propagation — they are leaf boundaries
//    even though they have ParentHashes.
func TestDetail07(t *testing.T) {
	st := memory.NewStorage()
	tr := rwTreeOf(t, st, map[string]string{"f": "x"})
	p := rwCommit(t, st, tr, rwT(1))          // behind the shallow boundary
	s := rwCommit(t, st, tr, rwT(2), p)       // shallow commit
	w := rwCommit(t, st, tr, rwT(3), s)       // want tip
	if err := st.SetShallow([]plumbing.Hash{s}); err != nil {
		t.Fatalf("SetShallow: %v", err)
	}

	got, err := Objects(st, []plumbing.Hash{w}, nil)
	if err != nil {
		t.Fatalf("Objects: %v", err)
	}
	if slices.Contains(got, p) {
		t.Fatalf("shallow boundary crossed: %v", got)
	}
	if !slices.Contains(got, w) || !slices.Contains(got, s) {
		t.Fatalf("shallow-side commits missing: %v", got)
	}
}

// 8. Non-commit wants seed directly: tags contribute their hash then
//    unwrap to the target; trees and blobs add themselves (trees
//    recursively).
func TestDetail08(t *testing.T) {
	st := memory.NewStorage()
	tr := rwTreeOf(t, st, map[string]string{"f": "x"})
	c := rwCommit(t, st, tr, rwT(1))
	tag := rwTag(t, st, "v1", c, rwT(2))
	blob := rwBlob(t, st, "loose blob")

	got, err := Objects(st, []plumbing.Hash{tag}, nil)
	if err != nil {
		t.Fatalf("tag want: %v", err)
	}
	if !slices.Contains(got, tag) || !slices.Contains(got, c) || !slices.Contains(got, tr) {
		t.Fatalf("tag unwrap incomplete: %v", got)
	}

	got2, err := Objects(st, []plumbing.Hash{blob}, nil)
	if err != nil {
		t.Fatalf("blob want: %v", err)
	}
	if !slices.Contains(got2, blob) {
		t.Fatalf("blob want not seeded")
	}

	got3, err := Objects(st, []plumbing.Hash{tr}, nil)
	if err != nil {
		t.Fatalf("tree want: %v", err)
	}
	if !slices.Contains(got3, tr) {
		t.Fatalf("tree want not seeded")
	}
	// tree children collected recursively
	f, err := object.GetTree(st, tr)
	if err != nil {
		t.Fatalf("GetTree: %v", err)
	}
	if !slices.Contains(got3, f.Entries[0].Hash) {
		t.Fatalf("tree children not collected: %v", got3)
	}
}

// 9. Tree collection diffs new trees against all parent trees by entry
//    name+hash — a blob unchanged in any single parent is not re-sent.
func TestDetail09(t *testing.T) {
	st := memory.NewStorage()
	base := rwTreeOf(t, st, map[string]string{"base": "1"})
	v0 := rwBlob(t, st, "x v0")
	v1 := rwBlob(t, st, "x v1")
	z := rwBlob(t, st, "z new")
	trP1 := rwTree(t, st, object.TreeEntry{Name: "x", Mode: filemode.Regular, Hash: v1})
	trP2 := rwTree(t, st, object.TreeEntry{Name: "x", Mode: filemode.Regular, Hash: v0})
	trM := rwTree(t, st,
		object.TreeEntry{Name: "x", Mode: filemode.Regular, Hash: v1},
		object.TreeEntry{Name: "z", Mode: filemode.Regular, Hash: z},
	)
	b := rwCommit(t, st, base, rwT(1))
	p1 := rwCommit(t, st, trP1, rwT(2), b)
	p2 := rwCommit(t, st, trP2, rwT(3), b)
	m := rwCommit(t, st, trM, rwT(4), p1, p2)

	got, err := Objects(st, []plumbing.Hash{m}, []plumbing.Hash{p1, b})
	if err != nil {
		t.Fatalf("Objects: %v", err)
	}
	// M's x matches parent p1's entry exactly — have-seen and not re-sent
	if slices.Contains(got, v1) {
		t.Fatalf("blob identical to a parent entry re-sent")
	}
	if !slices.Contains(got, z) || !slices.Contains(got, trM) {
		t.Fatalf("new objects missing: %v", got)
	}
	if !slices.Contains(got, m) || !slices.Contains(got, p2) {
		t.Fatalf("want commits missing: %v", got)
	}
}

// 10. Submodule entries are never collected — gitlinks produce no objects.
func TestDetail10(t *testing.T) {
	st := memory.NewStorage()
	gitlink := plumbing.NewHash("2222222222222222222222222222222222222222")
	tr := rwTree(t, st,
		object.TreeEntry{Name: "f.txt", Mode: filemode.Regular, Hash: rwBlob(t, st, "x")},
		object.TreeEntry{Name: "sub", Mode: filemode.Submodule, Hash: gitlink},
	)
	c := rwCommit(t, st, tr, rwT(1))

	got, err := Objects(st, []plumbing.Hash{c}, nil)
	if err != nil {
		t.Fatalf("gitlink broke the walk: %v", err)
	}
	if slices.Contains(got, gitlink) {
		t.Fatalf("gitlink collected as object")
	}
}

// 11. Result order is discovery order — commits sorted by time, tree
//     objects in tree-entry order, no post-sort. Inferable: no — assert
//     SHAPE: result is complete and commits are in non-increasing
//     committer-time order.
func TestDetail11(t *testing.T) {
	st := memory.NewStorage()
	tr1 := rwTreeOf(t, st, map[string]string{"a": "1"})
	tr2 := rwTreeOf(t, st, map[string]string{"a": "1", "b": "2"})
	c1 := rwCommit(t, st, tr1, rwT(1))
	c2 := rwCommit(t, st, tr2, rwT(2), c1)
	c3 := rwCommit(t, st, tr2, rwT(3), c2)

	got, err := Objects(st, []plumbing.Hash{c3}, nil)
	if err != nil {
		t.Fatalf("Objects: %v", err)
	}
	var commits []plumbing.Hash
	for _, h := range got {
		if _, err := object.GetCommit(st, h); err == nil {
			commits = append(commits, h)
		}
	}
	if len(commits) != 3 {
		t.Fatalf("commits collected: %v", commits)
	}
	for i := 1; i < len(commits); i++ {
		if rwWhen(t, st, commits[i]).After(rwWhen(t, st, commits[i-1])) {
			t.Fatalf("commit order not time-descending: %v", commits)
		}
	}
}

// 12. When no haves exist the walk takes the full-traversal path — no
//     boundary logic runs.
func TestDetail12(t *testing.T) {
	st := memory.NewStorage()
	tr1 := rwTreeOf(t, st, map[string]string{"a": "1"})
	tr2 := rwTreeOf(t, st, map[string]string{"a": "1", "b": "2"})
	c1 := rwCommit(t, st, tr1, rwT(1))
	c2 := rwCommit(t, st, tr2, rwT(2), c1)

	got, err := Objects(st, []plumbing.Hash{c2}, nil)
	if err != nil {
		t.Fatalf("Objects: %v", err)
	}
	// full traversal: every commit, tree, and blob reachable is collected
	for _, h := range []plumbing.Hash{c1, c2, tr1, tr2} {
		if !slices.Contains(got, h) {
			t.Fatalf("full walk missed %s", h)
		}
	}
	tt, _ := object.GetTree(st, tr2)
	for _, e := range tt.Entries {
		if !slices.Contains(got, e.Hash) {
			t.Fatalf("full walk missed blob %s", e.Hash)
		}
	}
}
