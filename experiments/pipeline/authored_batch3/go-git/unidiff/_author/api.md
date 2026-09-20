# Exported API — unidiff

Package `plumbing/format/diff` (module `example.internal/gitkit/v6`) — unified-diff
serializer.

`NewUnifiedEncoder(w io.Writer, contextLines int) *UnifiedEncoder`,
`SetColor(ColorConfig)`, `SetSrcPrefix/SetDstPrefix(string)` (defaults `a/`, `b/`),
`Encode(patch Patch) error`. `DefaultContextLines` = 3. `Patch`/`FilePatch`/`Chunk`/
`Operation` types live in `patch.go` (intact).

Callers: `git diff` porcelain, patch output paths. In-tree tests removed: 1.
