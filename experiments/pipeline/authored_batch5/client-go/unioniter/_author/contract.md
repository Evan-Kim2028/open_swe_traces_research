# Contract (L2) — unioniter

`UnionIter` merges a dirty (txn-buffer) iterator with a snapshot
iterator; both children must already yield keys in the requested
direction. Empty-value dirty entries are tombstones: they are skipped —
an equal-key tombstone consumes both children, and a tombstone ahead of
the snapshot key advances only the dirty child. On equal keys the dirty
record wins and the snapshot child is advanced past the duplicate. The
reverse iterator negates the key comparison; everything else is
unchanged. `updateCur` loops until a non-tombstone winner is found and
the iterator is invalid only when both children are exhausted. `Next`
advances whichever child produced the current record and re-runs
`updateCur`; `Value`/`Key` delegate to the current child, `Valid` returns
the validity flag, and `Close` nils both children after closing.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | `UnionIter` merges dirty and snapshot iterators in the requested direction |
| `TestDetail02` | empty-value dirty entries are tombstones: equal-key consumes both children, ahead-of-snapshot advances dirty only |
| `TestDetail03` | on equal keys the dirty record wins and the snapshot duplicate is skipped |
| `TestDetail04` | `reverse` negates the comparison; behavior otherwise unchanged |
| `TestDetail05` | `updateCur` loops past tombstones; invalid only when both children are exhausted |
| `TestDetail06` | `Next` advances the producing child; `Value`/`Key`/`Valid`/`Close` delegate and clean up |
