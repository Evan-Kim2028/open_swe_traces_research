# Exported API — revwalk

Package `plumbing/revlist` — the internal `objectWalk` engine behind
`Objects`/`ObjectsWithRef`: `newObjectWalk`, `shallowSet`,
`seedWants`, `seedHaves`, `walk`, `propagate`, `allStale`, `walkFull`,
`processCommitTrees`, `insertSorted`, `collectChangedTreeObjects`,
`collectAllTreeObjects`, `markTreeSeen`.

Kept visible: `objectWalk`/`missingParent` types, `wantPaint`/`havePaint`
flag constants, all doc comments; `revlist.go` entry points intact.

Callers: `revlist.Objects`, transport fetch/push object enumeration.
In-tree tests removed: 1 (`revlist_test.go`).
