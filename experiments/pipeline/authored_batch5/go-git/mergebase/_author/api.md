# Exported API — mergebase

Package `plumbing/object` — `Commit.MergeBase` (best common ancestors,
`git merge-base` semantics), `Commit.IsAncestor`, `Independents`
(`--independent` semantics), plus the commit-date sort, index/remove/
dedupe helpers and the `isInIndexCommitFilter`/`ancestorsIndex` internals.

Kept visible: `errIsReachable` sentinel, all doc comments (the
committer-date strategy rationale is spelled out), the commit-walker
iterators in `commit_walker*.go` that this code drives.

In-tree tests removed: 1 (`merge_base_test.go`).
