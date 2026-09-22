# Details — revwalk

1. Commits are popped newest-first — the queue stays sorted by committer
   time descending, so a want reachable from both a new and an old tip is
   visited once. Inferable: partially — the sort key is visible in the
   kept `insertSorted` signature and flag comments.
2. Paint propagates: a commit reached from both the want side and the
   have side is a boundary, and the walk stops when every queued commit
   is a boundary — it never walks all history when haves exist.
   Inferable: doc — the flag constants and walk doc comment say exactly
   this.
3. A commit painted want-only is tentatively collected, but if havePaint
   reaches it later it is dropped from the result — the boundary can move
   after discovery. Inferable: partially.
4. Haves seeds pre-mark all reachable tree/blob objects as seen — the
   result contains only objects new relative to the haves, not a full
   traversal. Inferable: doc — seedHaves' comment states it.
5. Missing objects on the haves side are tolerated (ErrObjectNotFound
   skipped); a missing object on the wants side is a hard error.
   Inferable: partially — asymmetric handling is a git semantic.
6. A missing parent is only an error if its child was never painted by
   haves — missing history behind the haves boundary is fine. Inferable:
   no — deferred validation is a subtle correctness detail.
7. Shallow commits stop parent propagation — they are leaf boundaries
   even though they have ParentHashes. Inferable: partially.
8. Non-commit wants seed directly: tags contribute their hash then
   unwrap to the target; trees and blobs add themselves (trees
   recursively). Inferable: partially.
9. Tree collection diffs new trees against all parent trees by entry
   name+hash — a blob unchanged in any single parent is not re-sent.
   Inferable: partially.
10. Submodule entries are never collected — gitlinks produce no objects.
    Inferable: doc.
11. Result order is discovery order — commits sorted by time, tree
    objects in tree-entry order, no post-sort. Inferable: no.
12. When no haves exist the walk takes the full-traversal path — no
    boundary logic runs. Inferable: doc — walk's comment calls it the
    fast path.
