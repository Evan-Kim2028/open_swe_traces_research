# Closure — revwalk

Package: `plumbing/revlist`. File: `object_walk.go`.

Removed (13 functions stubbed): `newObjectWalk`, `shallowSet`,
`objectWalk.seedWants`, `objectWalk.seedHaves`, `objectWalk.walk`,
`objectWalk.propagate`, `allStale`, `objectWalk.walkFull`,
`objectWalk.processCommitTrees`, `insertSorted`,
`collectChangedTreeObjects`, `collectAllTreeObjects`, `markTreeSeen`.

Kept: `objectWalk`/`missingParent` types, `wantPaint`/`havePaint`
constants and their semantics comments, `revlist.go` `Objects`/
`ObjectsWithRef` entry points. Disjoint from `mergebase` (ancestor
queries in plumbing/object) — this is the pack-enumeration walk.

Tests deleted: `plumbing/revlist/revlist_test.go` (1 — the package's
only test file; root `object_walker_test.go` exercises a different
walker type).
