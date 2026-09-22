# Contract — commitwalk

The commit-walker family in `plumbing/object`: preorder, postorder,
first-parent postorder, all-refs, BFS, filtered BFS, committer-time,
path-filtered and limit iterators. Every commitment below is covered by a
hidden test; every hidden test maps to a commitment.

## Commitments

1. **Termination.** Iterators end on `io.EOF` from `Next`, halt early and
   cleanly on `storer.ErrStop` from the callback, and propagate other
   callback errors. Covered by `TestDetail01`.
2. **Preorder.** A commit is yielded before its (unvisited) parents and no
   commit is ever revisited; the caller-supplied external seen map also
   gates descent. Covered by `TestDetail02`.
3. **Postorder / first-parent.** The postorder walk covers the history once
   each, deterministically, with the merged-in commit walked before the
   merge base; the first-parent variant follows only the first parent edge
   like `git log --first-parent`. Covered by `TestDetail03`.
4. **Ignore list.** Ignored commits are never emitted and never descended
   through. Covered by `TestDetail04`.
5. **BFS dedup.** Breadth-first order with parents marked seen at enqueue
   time — no commit enters the queue or is emitted twice. Covered by
   `TestDetail05`.
6. **Filtered BFS.** `isValid` gates emission, `isLimit` gates descent
   (the commit is yielded but not descended); a traversal failure is
   latched for `Error()`. Covered by `TestDetail06`.
7. **Committer-time order.** A heap on committer time pops the newest
   commit first; commits with equal times are all emitted (tie order is
   unspecified). Covered by `TestDetail07`.
8. **Path filtering.** A commit is emitted only when the diff to the next
   commit in the walk touches the filtered path; with `checkParent` the
   next commit is used as the diff base only when it is a real parent, and
   a merge identical to the next parent on the path is suppressed. Covered
   by `TestDetail08`.
9. **Limits.** Commits outside the since/until window are dropped; the walk
   stops once the tail hash is seen and the tail itself is not emitted.
   Covered by `TestDetail09`.
10. **All-refs seeding.** `NewCommitAllIter` seeds the walk from every
    reference's commit; a reference resolution failure propagates before
    any commit is yielded. Covered by `TestDetail10`.
11. **Close (shape).** `Close` is a safe no-op for plain iterators and
    releases a wrapped source iterator where one is held. Covered by
    `TestDetail11`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | partially |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | partially |
| TestDetail10 | 10 | partially |
| TestDetail11 | 11 | no — shape only |
