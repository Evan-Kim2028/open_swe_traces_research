# Closure — idxindex

Package: `plumbing/format/idxfile`. Files: `idxfile.go`, `lazy_index.go`,
`writer.go`.

Removed (64 functions stubbed): `MemoryIndex` methods (`Close`,
`findHashIndex`, `MayContain`, `Contains`, `FindOffset`, `getOffset`,
`FindCRC32`, `getCRC32`, `FindHash`, `genOffsetHash`, `Count`, `Entries`,
`EntriesWithPrefix`, `EntriesByOffset`, `idSize`), `NewMemoryIndex`;
`idxfileEntryIter`/`idxfilePrefixIter`/`idxfileEntryOffsetIter`/`entriesByOffset`
methods; `NewLazyIndex[WithPool]`, `LazyIndex.init` and all its lookup,
iterator and helper methods plus `scannerEntryIter`/`revEntryIter`/
`lazyPrefixIter`; `Writer` observer methods + `createIndex`/`addOffset64`,
`objects` sort impl.

Kept: `Index`/`EntryIter` interfaces, `Entry`, all struct field layouts,
`VersionSupported`/`isO64Mask`/section-size consts, `noMapping`, the doc
comments on fanout mapping, prefix-iterator lifetime, and LazyIndex fd
sharing; `decoder.go`/`encoder.go` wire codec intact (idxdecode owns it);
`internal/sharedfile` + `x/fdpool` packages intact.

Tests deleted: all 8 idxfile `*_test.go` files — every suite reaches the
index types.
