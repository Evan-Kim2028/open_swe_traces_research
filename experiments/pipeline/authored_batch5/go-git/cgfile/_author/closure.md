# Closure — cgfile

Package: `plumbing/format/commitgraph`. Files: `file.go`, `chain.go`.

Removed (19 functions stubbed): `readerSize`, `OpenFileIndex`,
`OpenFileIndexWithParent`, `fileIndex.Close`, `.verifyFileHeader`,
`.verifyFileSize`, `.readChunkHeaders`, `.verifyChunkSizes`, `.readFanout`,
`.GetIndexByHash`, `.GetCommitDataByIndex`, `.GetHashByIndex`,
`.getHashesFromIndexes`, `.Hashes`, `.HasGenerationV2`,
`.MaximumNumberOfHashes`; `OpenChainFile`, `OpenChainOrFileIndex`,
`OpenChainIndex`.

Kept: all five error vars, `commitFileSignature` and the parent-sentinel
consts (`parentNone`/`parentOctopusUsed`/`parentOctopusMask`/`parentLast`),
the `sz*`/layout consts, `fileIndex`/`sizer`/`ReaderAtCloser`/
`chunkAssignment` types, `chunk.go` (`ChunkType` enum + `chunkSignatures`),
`commitgraph.go` (`CommitData`, `Index`), `memory.go` — the in-memory Index
stays as a readable sibling. The doc comments on `verifyFileSize`,
`readChunkHeaders`, `verifyChunkSizes` carry the upstream-cited guards.

Tests deleted: `bench_test.go`, `chain_test.go`, `commitgraph_fuzz_test.go`,
`commitgraph_test.go` (4 — `encoder_test.go` stays; it exercises the kept
encoder, batch3 `cgenc` unit).
