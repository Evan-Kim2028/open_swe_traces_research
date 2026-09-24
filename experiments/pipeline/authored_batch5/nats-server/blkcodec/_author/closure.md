# Closure — blkcodec

File: `server/filestore.go`

Stubbed (8 symbols): `CompressionInfo.MarshalMetadata`,
`CompressionInfo.UnmarshalMetadata`, `StoreCompression.String`,
`StoreCompression.MarshalJSON`, `StoreCompression.UnmarshalJSON`,
`StoreCompression.Compress`, `StoreCompression.Decompress`,
`msgBlock.decode`.

Retained as scaffolding: `msgFromBufNoCopy` (used by the collision
fallback), `msgBlock`/`fileStore` machinery, encryption key path,
`checksumSize`, `s2` import. No imports blanked (all still used).

Test coverage snipped (restored for gold/cheat):
`TestFileStoreCompressionHeaderCollision`, `TestFileStoreDecodeCorruptBlock`,
`TestFileStoreCompressionAfterTruncate`, `TestFileStoreAtomicEraseMsg`
(filestore_test.go).
