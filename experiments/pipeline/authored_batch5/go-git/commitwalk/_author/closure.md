# Closure — commitwalk

Package: `plumbing/object`. Files: `commit_walker.go`,
`commit_walker_bfs.go`, `commit_walker_bfs_filtered.go`,
`commit_walker_ctime.go`, `commit_walker_path.go`,
`commit_walker_limit.go`.

Removed (42 functions stubbed): `forEachCommit`, `NewCommitPreorderIter`,
`commitPreIterator.Next/.ForEach/.Close`, `filteredParentIter`,
`NewCommitPostorderIter`, `commitPostIterator.Next/.ForEach/.Close`,
`NewCommitPostorderIterFirstParent`,
`commitPostIteratorFirstParent.Next/.ForEach/.Close`, `NewCommitAllIter`,
`addReference`; `NewCommitIterBSF`, `bfsCommitIterator.appendHash/.Next/
.ForEach/.Close`; `NewFilterCommitIter`, `filterCommitIter.Next/.ForEach/
.Error/.Close/.close/.popNewFromQueue/.addToQueue`; `NewCommitIterCTime`,
`commitIteratorByCTime.Next/.ForEach/.Close`; `NewCommitPathIterFromIter`,
`NewCommitFileIterFromIter`, `commitPathIter.Next/.getNextFileCommit/
.hasFileChange/.ForEach/.Close`, `isParentHash`; `NewCommitLimitIterFromIter`,
`commitLimitIter.Next/.ForEach/.Close`.

Kept: all iterator struct types and their field layouts, `CommitFilter`,
`LogLimitOptions`, `binaryheap` usage in signatures, all doc comments;
`merge_base.go` (excised separately in `mergebase`) and `difftree.go`
stay implemented.

Tests deleted: `commit_walker_test.go`,
`commit_walker_bfs_filtered_test.go` (2 — the only suites exercising the
iterators directly).
