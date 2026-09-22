package object

import (
	_ "errors"
	_ "io"
	_ "slices"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/plumbing/storer"
)

type commitPathIter struct {
	pathFilter    func(string) bool
	sourceIter    CommitIter
	currentCommit *Commit
	checkParent   bool
}

// NewCommitPathIterFromIter returns a commit iterator which performs diffTree between
// successive trees returned from the commit iterator from the argument. The purpose of this is
// to find the commits that explain how the files that match the path came to be.
// If checkParent is true then the function double checks if potential parent (next commit in a path)
// is one of the parents in the tree (it's used by `git log --all`).
// pathFilter is a function that takes path of file as argument and returns true if we want it
func NewCommitPathIterFromIter(pathFilter func(string) bool, commitIter CommitIter, checkParent bool) CommitIter {
	return &commitPathIter{pathFilter: pathFilter, sourceIter: commitIter, checkParent: checkParent}
}

// NewCommitFileIterFromIter is kept for compatibility, can be replaced with NewCommitPathIterFromIter
func NewCommitFileIterFromIter(fileName string, commitIter CommitIter, checkParent bool) CommitIter {
	return &commitPathIter{pathFilter: func(path string) bool { return path == fileName }, sourceIter: commitIter, checkParent: checkParent}
}

func (c *commitPathIter) Next() (*Commit, error) {
	return c.sourceIter.Next()
}

func (c *commitPathIter) getNextFileCommit() (*Commit, error) {
	panic("excised: commitPathIter.getNextFileCommit")
}

func (c *commitPathIter) hasFileChange(changes Changes, parent *Commit) bool {
	panic("excised: commitPathIter.hasFileChange")
}

func isParentHash(hash plumbing.Hash, commit *Commit) bool {
	panic("excised: isParentHash")
}

func (c *commitPathIter) ForEach(cb func(*Commit) error) error {
	return forEachCommit(c.Next, cb)
}

func (c *commitPathIter) Close() {
	c.sourceIter.Close()
}
