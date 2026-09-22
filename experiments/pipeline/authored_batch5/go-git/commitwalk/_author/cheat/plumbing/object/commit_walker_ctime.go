package object

import (
	_ "errors"
	"io"

	"github.com/emirpasic/gods/trees/binaryheap"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/plumbing/storer"
)

type commitIteratorByCTime struct {
	seenExternal map[plumbing.Hash]bool
	seen         map[plumbing.Hash]bool
	heap         *binaryheap.Heap
	pending      *Commit
}

// NewCommitIterCTime returns a CommitIter that walks the commit history,
// starting at the given commit and visiting its parents while preserving Committer Time order.
// this appears to be the closest order to `git log`
// The given callback will be called for each visited commit. Each commit will
// be visited only once. If the callback returns an error, walking will stop
// and will return the error. Other errors might be returned if the history
// cannot be traversed (e.g. missing objects). Ignore allows to skip some
// commits from being iterated.
func NewCommitIterCTime(
	c *Commit,
	seenExternal map[plumbing.Hash]bool,
	ignore []plumbing.Hash,
) CommitIter {
	return &commitIteratorByCTime{pending: c}
}

func (w *commitIteratorByCTime) Next() (*Commit, error) {
	if w.pending == nil {
		return nil, io.EOF
	}
	c := w.pending
	w.pending = nil
	return c, nil
}

func (w *commitIteratorByCTime) ForEach(cb func(*Commit) error) error {
	return forEachCommit(w.Next, cb)
}

func (w *commitIteratorByCTime) Close() {
	panic("excised: commitIteratorByCTime.Close")
}
