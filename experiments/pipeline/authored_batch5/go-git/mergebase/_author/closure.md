# Closure — mergebase

Package: `plumbing/object`. File: `merge_base.go`.

Removed (9 functions stubbed): `Commit.MergeBase`, `Commit.IsAncestor`,
`ancestorsIndex`, `Independents`, `sortByCommitDateDesc`, `indexOf`,
`remove`, `removeDuplicated`, `isInIndexCommitFilter`.

Kept: `errIsReachable` sentinel (doc comment names its meaning), all doc
comments including the committer-date strategy rationale, and the entire
commit-walker layer (`commit_walker*.go`) this file drives — the
traversal primitives are readable siblings.

Tests deleted: `merge_base_test.go` (1 — the only suite that reaches it;
walker tests stay since walkers are intact).
