# Closure — unioniter

Package: `internal/unionstore`. File:
`internal/unionstore/union_iter.go` (208 lines).

Removed (all bodies stubbed): `NewUnionIter`, `UnionIter.dirtyNext`,
`.snapshotNext`, `.updateCur`, `.Next`, `.Value`, `.Key`, `.Valid`,
`.Close`.

Kept: `UnionIter` struct and field layout, `Iterator` interface (in
`union_store.go`), the tombstone/warn log call site context.

Tests edited: none — `union_store_test.go` reaches the closure through
`us.Iter`/`IterReverse`; the panic stubs are the failing signal.
