# Exported API — indexenc

Package `plumbing/format/index` (module `example.internal/gitkit/v6`) — git index (`.git/index`)
v2/3/4 writer.

`NewEncoder(w io.Writer, h hash.Hash, opts ...Option) *Encoder` (`WithSkipHash` option kept),
`Encoder.Encode(idx *Index) error`. `EncodeVersionSupported` (=4), `ErrInvalidTimestamp` kept.
The `Index`/`Entry` types and the decoder live in sibling files and stay intact.

Callers: worktree/index write paths in the storage backend. In-tree tests removed: 1.
