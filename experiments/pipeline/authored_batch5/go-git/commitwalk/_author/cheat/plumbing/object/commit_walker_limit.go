package object

import (
	_ "errors"
	_ "io"
	"time"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/plumbing/storer"
)

type commitLimitIter struct {
	sourceIter   CommitIter
	limitOptions LogLimitOptions
}

// LogLimitOptions defines limits for log traversal.
type LogLimitOptions struct {
	Since    *time.Time
	Until    *time.Time
	TailHash plumbing.Hash
}

// NewCommitLimitIterFromIter creates a new commit iterator with limits applied.
func NewCommitLimitIterFromIter(commitIter CommitIter, limitOptions LogLimitOptions) CommitIter {
	return &commitLimitIter{sourceIter: commitIter, limitOptions: limitOptions}
}

func (c *commitLimitIter) Next() (*Commit, error) {
	return c.sourceIter.Next()
}

func (c *commitLimitIter) ForEach(cb func(*Commit) error) error {
	return forEachCommit(c.Next, cb)
}

func (c *commitLimitIter) Close() {
	c.sourceIter.Close()
}
