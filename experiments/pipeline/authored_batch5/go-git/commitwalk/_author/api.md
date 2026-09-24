# Exported API — commitwalk

Package `plumbing/object` — the commit-walker family: preorder, postorder,
postorder-first-parent, all-commits-from-refs, BFS, filtered BFS,
committer-time ordered, path-filtered and limit iterators.

`NewCommitPreorderIter`, `NewCommitPostorderIter`,
`NewCommitPostorderIterFirstParent`, `NewCommitAllIter`,
`NewCommitIterBSF`, `NewFilterCommitIter`, `NewCommitIterCTime`,
`NewCommitPathIterFromIter`, `NewCommitFileIterFromIter`,
`NewCommitLimitIterFromIter`; `LogLimitOptions`; `CommitFilter`;
`forEachCommit` driver; per-iterator `Next`/`ForEach`/`Close`/`Error`.

Callers: `Log`, `MergeBase`, rev-list plumbing. In-tree tests removed: 2.
