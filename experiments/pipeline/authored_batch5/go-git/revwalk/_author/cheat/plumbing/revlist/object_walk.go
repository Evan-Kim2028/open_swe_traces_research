package revlist

import (
	_ "errors"
	"fmt"
	"sort"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	"example.internal/gitkit/v6/plumbing/object"
	"example.internal/gitkit/v6/plumbing/storer"
)

// objectWalk holds the state for a single Objects computation.
type objectWalk struct {
	s          storer.EncodedObjectStorer
	shallows   map[plumbing.Hash]struct{}
	wantsQueue []*object.Commit
	havesQueue []*object.Commit
	wantsSeen  map[plumbing.Hash]struct{}
	havesSeen  map[plumbing.Hash]struct{}
	seen       map[plumbing.Hash]struct{}
	result     []plumbing.Hash
}

func newObjectWalk(s storer.EncodedObjectStorer) (*objectWalk, error) {
	return &objectWalk{
		s:         s,
		shallows:  map[plumbing.Hash]struct{}{},
		wantsSeen: make(map[plumbing.Hash]struct{}),
		havesSeen: make(map[plumbing.Hash]struct{}),
		seen:      make(map[plumbing.Hash]struct{}),
	}, nil
}

func shallowSet(s storer.EncodedObjectStorer) (map[plumbing.Hash]struct{}, error) {
	panic("excised: shallowSet")
}

// seedWants resolves each want hash and enqueues commits for walking.
// Non-commit objects (blobs, trees, tags) are added directly to the result.
func (w *objectWalk) seedWants(wants []plumbing.Hash) error {
	for i := 0; i < len(wants); i++ {
		h := wants[i]
		if _, ok := w.wantsSeen[h]; ok {
			continue
		}
		if _, ok := w.seen[h]; ok {
			continue
		}

		o, err := w.s.EncodedObject(plumbing.AnyObject, h)
		if err != nil {
			return fmt.Errorf("getting wanted object %s: %w", h, err)
		}

		switch o.Type() {
		case plumbing.CommitObject:
			c, err := object.DecodeCommit(w.s, o)
			if err != nil {
				return fmt.Errorf("decoding commit %s: %w", h, err)
			}
			w.wantsSeen[h] = struct{}{}
			insertSorted(&w.wantsQueue, c)
		case plumbing.TagObject:
			tag, err := object.DecodeTag(w.s, o)
			if err != nil {
				return fmt.Errorf("decoding tag %s: %w", h, err)
			}
			w.seen[tag.Hash] = struct{}{}
			w.result = append(w.result, tag.Hash)
			wants = append(wants, tag.Target)
		case plumbing.TreeObject:
			t, err := object.GetTree(w.s, h)
			if err != nil {
				return fmt.Errorf("getting tree %s: %w", h, err)
			}
			if err := collectAllTreeObjects(w.s, t, w.seen, &w.result); err != nil {
				return err
			}
		case plumbing.BlobObject:
			w.seen[h] = struct{}{}
			w.result = append(w.result, h)
		default:
			return fmt.Errorf("unsupported object type %s for %s", o.Type(), h)
		}
	}
	return nil
}

// seedHaves enqueues each have commit and pre-populates seen with all
// tree/blob objects reachable from the haves tips. Non-commit objects
// (tags, trees, blobs) are marked as seen so the diff walk skips them.
// Missing objects (ErrObjectNotFound) are tolerated since the remote
// may advertise refs we don't have locally.
func (w *objectWalk) seedHaves(haves []plumbing.Hash) error {
	for _, h := range haves {
		w.havesSeen[h] = struct{}{}
	}
	return nil
}

// Paint flags for the commit walk. A commit reachable from both
// sides is a boundary; once all queue entries have both flags
// the walk can stop.
const (
	wantPaint uint8 = 1 << iota
	havePaint
)

// walk identifies new commits and collects their tree objects.
func (w *objectWalk) walk() error {
	return w.walkFull()
}

// missingParent records a parent commit that could not be loaded during
// the painted walk, along with the child from which it was reached.
// Validation is deferred until the walk completes so that havePaint
// propagation from later iterations can mark the child (and its missing
// parent) as behind the haves boundary.
type missingParent struct {
	hash  plumbing.Hash
	child plumbing.Hash
}

// propagate adds the given flags to each parent commit. Parents that
// already have all the flags are skipped. Missing parents are not
// treated as fatal here; they are recorded and validated after the
// walk, where we can tell whether the child was eventually painted by
// haves (in which case the missing parent is behind the haves boundary
// and tolerable, matching Git's behavior).
func (w *objectWalk) propagate(queue *[]*object.Commit, flags map[plumbing.Hash]uint8, missing *[]missingParent, lc *object.Commit, f uint8) error {
	panic("excised: objectWalk.propagate")
}

// allStale returns true when every commit in the queue has both paint
// flags, meaning all remaining commits are boundaries and no new
// commits can be discovered.
func allStale(queue []*object.Commit, flags map[plumbing.Hash]uint8) bool {
	panic("excised: allStale")
}

// walkFull is the fast path when there are no haves. It walks all
// commits and collects every reachable tree/blob via a simple seen-set
// traversal — no per-commit tree diffs needed.
func (w *objectWalk) walkFull() error {
	for len(w.wantsQueue) > 0 {
		lc := w.wantsQueue[0]
		w.wantsQueue = w.wantsQueue[1:]

		if _, ok := w.seen[lc.Hash]; ok {
			continue
		}
		w.seen[lc.Hash] = struct{}{}
		w.result = append(w.result, lc.Hash)

		tree, err := lc.Tree()
		if err != nil {
			return fmt.Errorf("getting tree for %s: %w", lc.Hash, err)
		}

		if err := collectAllTreeObjects(w.s, tree, w.seen, &w.result); err != nil {
			return fmt.Errorf("collecting tree objects for %s: %w", lc.Hash, err)
		}

		for _, ph := range lc.ParentHashes {
			if _, ok := w.wantsSeen[ph]; ok {
				continue
			}
			w.wantsSeen[ph] = struct{}{}
			pc, err := object.GetCommit(w.s, ph)
			if err != nil {
				return fmt.Errorf("getting parent commit %s: %w", ph, err)
			}
			insertSorted(&w.wantsQueue, pc)
		}
	}
	return nil
}

// processCommitTrees collects new tree/blob objects for a commit by
// diffing its tree against its parents' trees.
func (w *objectWalk) processCommitTrees(lc *object.Commit) error {
	panic("excised: objectWalk.processCommitTrees")
}

// insertSorted inserts a commit into a slice sorted by committer time
// descending (newest first).
func insertSorted(q *[]*object.Commit, c *object.Commit) {
	i := sort.Search(len(*q), func(i int) bool {
		return (*q)[i].Committer.When.Before(c.Committer.When)
	})
	*q = append(*q, nil)
	copy((*q)[i+1:], (*q)[i:])
	(*q)[i] = c
}

// collectChangedTreeObjects walks newTree, comparing entry hashes against
// all oldTrees. An entry is considered unchanged if any old tree contains the
// same name with the same hash. Only new or modified tree and blob hashes are
// added to result.
func collectChangedTreeObjects(
	s storer.EncodedObjectStorer,
	newTree *object.Tree,
	oldTrees []*object.Tree,
	seen map[plumbing.Hash]struct{},
	result *[]plumbing.Hash,
) error {
	panic("excised: collectChangedTreeObjects")
}

// collectAllTreeObjects recursively walks a tree, adding all unseen
// tree and blob hashes to result. This is faster than collectChangedTreeObjects
// when we need all objects (no haves to diff against).
func collectAllTreeObjects(
	s storer.EncodedObjectStorer,
	t *object.Tree,
	seen map[plumbing.Hash]struct{},
	result *[]plumbing.Hash,
) error {
	if _, ok := seen[t.Hash]; ok {
		return nil
	}
	seen[t.Hash] = struct{}{}
	*result = append(*result, t.Hash)

	for _, e := range t.Entries {
		if e.Mode == filemode.Submodule {
			continue
		}
		if _, ok := seen[e.Hash]; ok {
			continue
		}
		if e.Mode == filemode.Dir {
			sub, err := object.GetTree(s, e.Hash)
			if err != nil {
				return fmt.Errorf("getting subtree %s: %w", e.Hash, err)
			}
			if err := collectAllTreeObjects(s, sub, seen, result); err != nil {
				return err
			}
		} else {
			seen[e.Hash] = struct{}{}
			*result = append(*result, e.Hash)
		}
	}
	return nil
}

// markTreeSeen recursively adds all tree and blob hashes in t to seen.
// Objects added to seen are not added to result — this is used to mark
// haves-reachable objects so they are skipped during diff walks.
func markTreeSeen(s storer.EncodedObjectStorer, t *object.Tree, seen map[plumbing.Hash]struct{}) {
	panic("excised: markTreeSeen")
}
