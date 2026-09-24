# Exported API — cgenc

Package `plumbing/format/commitgraph` (module `example.internal/gitkit/v6`) — commit-graph
(CPGH) file writer.

`NewEncoder(w io.Writer) *Encoder` (SHA-1 tee visible), `Encoder.Encode(idx Index) error`.
`Index` interface (`Hashes`, `GetIndexByHash`, `GetCommitDataByIndex`, `HasGenerationV2`)
and `CommitData{TreeHash, ParentHashes, When, Generation, GenerationV2Data}` intact.
Errors `ErrTooManyChunks`, `ErrParentNotInIndex` kept; `doc.go` describes the chunk format.

Callers: commit-graph write paths for commit traversal acceleration. In-tree tests removed: 1.
