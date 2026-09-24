# Details — unioniter

1. `UnionIter` merges a dirty (txn buffer) iterator and a snapshot iterator;
   both children must already yield keys in the requested direction.
   Inferable: doc.
2. Empty-value dirty entries are TOMBSTONES: `updateCur` skips them — an
   equal-key tombstone consumes BOTH children; a tombstone ahead of the
   snapshot key logs "delete a record not exists?" and advances dirty only.
   Inferable: no — tombstone semantics are undocumented.
3. On equal keys the dirty record wins and the snapshot child is advanced
   past the duplicate (`snapshotNext` without emitting it). Inferable: no.
4. `reverse` negates the key comparison; everything else is unchanged.
   Inferable: partially.
5. `updateCur` loops until a non-tombstone winner is found; `isValid` is
   false only when both children are exhausted. Inferable: partially.
6. `Next` advances whichever child produced the current record then
   re-runs `updateCur`; `Value`/`Key` delegate to the current child;
   `Valid` returns `isValid`; `Close` nils both children after closing.
   Inferable: yes.
