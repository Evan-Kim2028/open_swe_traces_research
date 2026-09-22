# Exported API — cgfile

Package `plumbing/format/commitgraph` (module `example.internal/gitkit/v6`) —
reader for the on-disk commit-graph file (`CGPH`), its table of contents,
fanout, commit-data and edge/generation chunks, plus chained-graph
coalescing.

`OpenFileIndex(ReaderAtCloser)`, `OpenFileIndexWithParent(r, parent)`,
`OpenChainFile(io.Reader) []string`, `OpenChainIndex(billy.Filesystem)`,
`OpenChainOrFileIndex(fs)`; `Index` iface via `fileIndex` impl
(`GetIndexByHash/GetCommitDataByIndex/GetHashByIndex/Hashes/HasGenerationV2/
MaximumNumberOfHashes/Close`); error vars `ErrUnsupportedVersion`,
`ErrUnsupportedHash`, `ErrMalformedCommitGraphFile`, `ErrTooManyChunks`,
`ErrParentNotInIndex`; `ChunkType` + signatures kept.

Callers: commit-graph backed commit walks (`object/commitgraph`). In-tree
tests removed: 4.
