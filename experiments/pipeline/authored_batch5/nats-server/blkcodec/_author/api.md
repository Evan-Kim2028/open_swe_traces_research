# API — blkcodec

Module: `example.internal/msgkit/v2`, package `server`, file
`server/filestore.go`.

Excised symbols:

- `func (c *CompressionInfo) MarshalMetadata() []byte` — writes the
  per-block `"cmp"` compression metadata header.
- `func (c *CompressionInfo) UnmarshalMetadata(b []byte) (int, error)`
  — sniffs/parses it, returning bytes consumed (0 = not compressed).
- `func (alg StoreCompression) String() string`,
  `MarshalJSON() ([]byte, error)`,
  `(*StoreCompression) UnmarshalJSON(b []byte) error` — the enum's
  display/JSON codec ("none"/"s2").
- `func (alg StoreCompression) Compress(buf []byte) ([]byte, error)`,
  `Decompress(buf []byte) ([]byte, error)` — s2 block codec that
  preserves the trailing checksum uncompressed.
- `func (mb *msgBlock) decode(buf []byte) ([]byte, StoreCompression,
  error)` — per-block dispatch: sniff metadata, decompress, with a
  fallback for uncompressed records whose length collides with the
  `"cmp"` magic.

Callers: block load path (`decompressIfNeeded`, `loadBlock`), block
write path (`Compress` before flush), stream-config JSON decode, and
test-side block inspection.

Retained dependencies: `msgFromBufNoCopy` (collision fallback),
`checksumSize`, `s2`, `fileStoreMsgSize`.
