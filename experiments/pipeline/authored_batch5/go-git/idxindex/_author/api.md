# Exported API — idxindex

Package `plumbing/format/idxfile` — the Index IMPLEMENTATIONS:
`MemoryIndex` (bucketed in-memory tables), `LazyIndex` (fd-pooled
ReadAt lookups over .idx + .rev), the pack-scan `Writer` observer, and
the entry iterators (`idxfileEntryIter`, `idxfilePrefixIter`,
`idxfileEntryOffsetIter`, `scannerEntryIter`, `revEntryIter`,
`lazyPrefixIter`).

`NewMemoryIndex`, `Contains/MayContain/FindOffset/FindCRC32/FindHash/
Count/Entries/EntriesWithPrefix/EntriesByOffset/Close` on both index
types; `Writer.Index/Add/Finished/OnHeader/OnInflatedObjectHeader/
OnInflatedObjectContent/OnFooter`; `NewLazyIndex[WithPool]`,
`LazyIndex.init`, offset/crc32/hashAtPos/findHashViaRev/entryAt helpers.

Wire codec (decoder.go/encoder.go) stays intact — `idxdecode` unit.
In-tree tests removed: all 8 idxfile test files (every suite reaches the
index types).
