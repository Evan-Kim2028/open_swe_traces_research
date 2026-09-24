# Exported API — unioniter

Package `internal/unionstore` (module `example.internal/kvstore/v2`).

- `NewUnionIter(dirtyIt, snapshotIt Iterator, reverse bool) (*UnionIter, error)`
- `(*UnionIter) Next() error`, `Key() []byte`, `Value() []byte`,
  `Valid() bool`, `Close()` — implements the package `Iterator` interface.

Package-internal: `dirtyNext`, `snapshotNext`, `updateCur`.

Callers: `union_store.go` `Iter`/`IterReverse` wrap the buffer+snapshot
pair into a `UnionIter`. In-tree `union_store_test.go` exercises it through
`us.Iter`/`IterReverse` — kept, it is the intended failure signal.
