# Contract — revwalk

`plumbing/revlist`'s `objectWalk` engine behind `Objects`/`ObjectsWithRef`:
a wants/haves painted walk over commits with per-commit tree diffing.
Every commitment below is covered by a hidden test; every hidden test maps
to a commitment.

## Commitments

1. **Newest-first queue.** Commits pop in committer-time descending order;
   a commit reachable from multiple tips is visited once. Covered by
   `TestDetail01`.
2. **Paint boundary.** A commit reached from both want and have sides is a
   boundary; the walk stops once every queued commit is a boundary and
   never collects have-side objects. Covered by `TestDetail02`.
3. **Moving boundary.** A want-discovered commit later reached by
   havePaint is dropped from the result. Covered by `TestDetail03`.
4. **Haves pre-marking.** Tree/blob objects reachable from haves are
   pre-marked seen; the result holds only objects new relative to the
   haves. Covered by `TestDetail04`.
5. **Asymmetric missing objects.** Missing haves are tolerated; missing
   wants are a hard error. Covered by `TestDetail05`.
6. **Missing parents (shape).** A missing parent errors only when its
   child was never painted by haves — missing history behind the haves
   boundary is tolerated. Covered by `TestDetail06`.
7. **Shallow boundary.** Shallow commits stop parent propagation even
   though they carry ParentHashes. Covered by `TestDetail07`.
8. **Non-commit wants.** Tags contribute their hash and unwrap to their
   target; trees and blobs add themselves, trees recursively. Covered by
   `TestDetail08`.
9. **All-parents tree diff.** New trees are diffed against ALL parent
   trees by entry name+hash; a blob unchanged in any single parent is not
   re-sent. Covered by `TestDetail09`.
10. **Gitlinks skipped.** Submodule entries produce no objects — the walk
    neither emits nor errors on a gitlink hash. Covered by `TestDetail10`.
11. **Discovery order (shape).** Result order is discovery order — commits
    in non-increasing committer time, tree objects as encountered; asserted
    shape is completeness plus descending commit order. Covered by
    `TestDetail11`.
12. **Full-traversal fast path.** With no haves the walk collects every
    reachable commit, tree, and blob — no boundary logic. Covered by
    `TestDetail12`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | doc |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | no — shape only |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | partially |
| TestDetail10 | 10 | doc |
| TestDetail11 | 11 | no — shape only |
| TestDetail12 | 12 | doc |
