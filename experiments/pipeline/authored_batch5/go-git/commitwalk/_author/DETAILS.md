# Details — commitwalk

1. Every iterator terminates on io.EOF from Next and halts early — cleanly —
   on storer.ErrStop from the callback; other callback errors propagate.
   Inferable: doc — the ForEach doc comments state the stop semantics.
2. Preorder yields a commit before its parents and never revisits — the
   seen set plus the caller-supplied external seen map both gate descent.
   Inferable: doc — "each commit will be visited only once" is kept.
3. Postorder yields parents before the commit itself; the first-parent
   variant follows only the first parent edge. Inferable: partially — the
   naming and doc shape are visible, the traversal stack mechanics are the
   detail.
4. The ignore list seeds the seen set — ignored commits are never emitted
   and never descended through. Inferable: partially.
5. BFS keeps a queue and marks parents seen at ENQUEUE time — duplicates
   never enter the queue twice. Inferable: partially.
6. The filtered BFS consults isValid before emitting and isLimit before
   descending — a limit hit yields the commit but stops the descent, and a
   storer failure is latched for Error() while the walk ends. Inferable:
   partially — the filter types are kept, the latching is the detail.
7. The ctime iterator keeps a min-heap on committer time so the newest
   commit pops first — equal timestamps fall back to insertion order.
   Inferable: partially — the heap field is visible, ordering detail is
   not.
8. The path iterator emits a commit only when the diff to its parent —
   or, with checkParent off, to each parent — touches the filtered path;
   merge commits need the flag set to be compared against every parent.
   Inferable: partially — the doc comment describes path filtering, the
   checkParent split is the detail.
9. The limit iterator drops commits outside the since/until window and
   stops entirely once the tail hash is seen — the tail itself is not
   emitted. Inferable: partially — LogLimitOptions fields are visible.
10. NewCommitAllIter seeds the walk from every reference's commit — the
    reference walk failure propagates before any commit is yielded.
    Inferable: partially.
11. Iterators hold only their own state — Close releases wrapped source
    iterators where present and is otherwise a no-op. Inferable: no.
