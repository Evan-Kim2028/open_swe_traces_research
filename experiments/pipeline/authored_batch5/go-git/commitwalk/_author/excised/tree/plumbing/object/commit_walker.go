package object

import (
	"container/list"
	_ "errors"
	"io"
	_ "slices"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/plumbing/storer"
	"example.internal/gitkit/v6/storage"
)

type commitPreIterator struct {
	seenExternal map[plumbing.Hash]bool
	seen         map[plumbing.Hash]bool
	stack        []CommitIter
	start        *Commit
}

func forEachCommit(next func() (*Commit, error), cb func(*Commit) error) error {
	panic("excised: forEachCommit")
}

// NewCommitPreorderIter returns a CommitIter that walks the commit history,
// starting at the given commit and visiting its parents in pre-order.
// The given callback will be called for each visited commit. Each commit will
// be visited only once. If the callback returns an error, walking will stop
// and will return the error. Other errors might be returned if the history
// cannot be traversed (e.g. missing objects). Ignore allows to skip some
// commits from being iterated.
func NewCommitPreorderIter(
	c *Commit,
	seenExternal map[plumbing.Hash]bool,
	ignore []plumbing.Hash,
) CommitIter {
	panic("excised: NewCommitPreorderIter")
}

func (w *commitPreIterator) Next() (*Commit, error) {
	panic("excised: commitPreIterator.Next")
}

func filteredParentIter(c *Commit, seen map[plumbing.Hash]bool) CommitIter {
	panic("excised: filteredParentIter")
}

func (w *commitPreIterator) ForEach(cb func(*Commit) error) error {
	panic("excised: commitPreIterator.ForEach")
}

func (w *commitPreIterator) Close() {
	panic("excised: commitPreIterator.Close")
}

type commitPostIterator struct {
	stack []*Commit
	seen  map[plumbing.Hash]bool
}

// NewCommitPostorderIter returns a CommitIter that walks the commit
// history like WalkCommitHistory but in post-order. This means that after
// walking a merge commit, the merged commit will be walked before the base
// it was merged on. This can be useful if you wish to see the history in
// chronological order. Ignore allows to skip some commits from being iterated.
func NewCommitPostorderIter(c *Commit, ignore []plumbing.Hash) CommitIter {
	panic("excised: NewCommitPostorderIter")
}

func (w *commitPostIterator) Next() (*Commit, error) {
	panic("excised: commitPostIterator.Next")
}

func (w *commitPostIterator) ForEach(cb func(*Commit) error) error {
	panic("excised: commitPostIterator.ForEach")
}

func (w *commitPostIterator) Close() {
	panic("excised: commitPostIterator.Close")
}

type commitPostIteratorFirstParent struct {
	stack []*Commit
	seen  map[plumbing.Hash]bool
}

// NewCommitPostorderIterFirstParent returns a CommitIter that walks the commit
// history like WalkCommitHistory but in post-order.
//
// This option acts like the git log --first-parent flag, skipping intermediate
// commits that were brought in via a merge commit.
// Ignore allows to skip some commits from being iterated.
func NewCommitPostorderIterFirstParent(c *Commit, ignore []plumbing.Hash) CommitIter {
	panic("excised: NewCommitPostorderIterFirstParent")
}

func (w *commitPostIteratorFirstParent) Next() (*Commit, error) {
	panic("excised: commitPostIteratorFirstParent.Next")
}

func (w *commitPostIteratorFirstParent) ForEach(cb func(*Commit) error) error {
	panic("excised: commitPostIteratorFirstParent.ForEach")
}

func (w *commitPostIteratorFirstParent) Close() {
	panic("excised: commitPostIteratorFirstParent.Close")
}

// commitAllIterator stands for commit iterator for all refs.
type commitAllIterator struct {
	// currCommit points to the current commit.
	currCommit *list.Element
}

// NewCommitAllIter returns a new commit iterator for all refs.
// repoStorer is a repo Storer used to get commits and references.
// commitIterFunc is a commit iterator function, used to iterate through ref commits in chosen order
func NewCommitAllIter(repoStorer storage.Storer, commitIterFunc func(*Commit) CommitIter) (CommitIter, error) {
	panic("excised: NewCommitAllIter")
}

func addReference(
	repoStorer storage.Storer,
	commitIterFunc func(*Commit) CommitIter,
	ref *plumbing.Reference,
	commitsPath *list.List,
	commitsLookup map[plumbing.Hash]*list.Element,
) error {
	panic("excised: addReference")
}

func (it *commitAllIterator) Next() (*Commit, error) {
	if it.currCommit == nil {
		return nil, io.EOF
	}

	c := it.currCommit.Value.(*Commit)
	it.currCommit = it.currCommit.Next()

	return c, nil
}

func (it *commitAllIterator) ForEach(cb func(*Commit) error) error {
	return forEachCommit(it.Next, cb)
}

func (it *commitAllIterator) Close() {
	it.currCommit = nil
}
