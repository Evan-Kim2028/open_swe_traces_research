package object

import (
	_ "errors"
	"io"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/storer"
)

type bfsCommitIterator struct {
	seenExternal map[plumbing.Hash]bool
	seen         map[plumbing.Hash]bool
	queue        []*Commit
}

// NewCommitIterBSF returns a CommitIter that walks the commit history,
// starting at the given commit and visiting its parents in pre-order.
// The given callback will be called for each visited commit. Each commit will
// be visited only once. If the callback returns an error, walking will stop
// and will return the error. Other errors might be returned if the history
// cannot be traversed (e.g. missing objects). Ignore allows to skip some
// commits from being iterated.
func NewCommitIterBSF(
	c *Commit,
	seenExternal map[plumbing.Hash]bool,
	ignore []plumbing.Hash,
) CommitIter {
	return &bfsCommitIterator{queue: []*Commit{c}}
}

func (w *bfsCommitIterator) appendHash(store storer.EncodedObjectStorer, h plumbing.Hash) error {
	panic("excised: bfsCommitIterator.appendHash")
}

func (w *bfsCommitIterator) Next() (*Commit, error) {
	if len(w.queue) == 0 {
		return nil, io.EOF
	}
	c := w.queue[0]
	w.queue = w.queue[1:]
	return c, nil
}

func (w *bfsCommitIterator) ForEach(cb func(*Commit) error) error {
	return forEachCommit(w.Next, cb)
}

func (w *bfsCommitIterator) Close() {
	panic("excised: bfsCommitIterator.Close")
}
