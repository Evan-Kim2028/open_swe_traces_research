# Closure — cgenc

Package: `plumbing/format/commitgraph`. File: `encoder.go`.

Removed (12 functions stubbed): `Encoder.Encode`, `lookupParentIndex`, `Encoder.prepare`,
`encodeFileHeader`, `encodeChunkHeaders`, `encodeFanout`, `encodeOidLookup`,
`encodeCommitData`, `encodeExtraEdges`, `encodeGenerationV2Data`,
`encodeGenerationV2Overflow`, `encodeChecksum`.

Kept: `Encoder` struct + `NewEncoder` (the hash-tee wiring is visible), all constants
(`commitFileSignature`, `parentNone`, `parentOctopusUsed`, `parentLast`, size consts),
error vars incl. `ErrTooManyChunks`/`ErrParentNotInIndex`, `Index`/`CommitData` types and
`chunk.go` (chunk signatures visible), `doc.go` format description, the whole `file.go`
reader INTACT — decode-side layout is partially legible by design.

Tests deleted: `encoder_test.go` only (the 800-line reader test stays as unrelated tests).
