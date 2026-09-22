# Exported API — deltadiff

Package `plumbing/format/packfile` — pack DELTA GENERATION: the block-hash
index (`deltaIndex`, JGit-derived), the rolling scanner, and the
insert/copy opcode encoder.

`GetDelta(base, target EncodedObject) (EncodedObject, error)`,
`DiffDelta(src, tgt []byte) []byte`; internal: `diffDelta`,
`encodeInsertOperation`, `encodeCopyOperation`, `deltaIndex`
(`init`/`findMatch`), `matchLength`, `countEntries`, `copyEntries`,
`deltaIndexScanner` (`newDeltaIndexScanner`/`scan`), `tableSize`,
`leadingZeros`, `hashBlock`, `T`, `len8tab`; consts `s=16`, `blksz=16`,
`maxCopySize=64K`, `maxChainLength=64`.

Used by `DeltaSelector` (pack compression). In-tree tests removed: 8.
