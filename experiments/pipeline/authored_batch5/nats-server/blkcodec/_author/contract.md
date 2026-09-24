# Contract — blkcodec

`CompressionInfo`/`StoreCompression`/`msgBlock.decode` implement the
per-block compression metadata header and the s2 block codec that keeps
the trailing checksum in the clear. Every commitment below is covered by
a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Metadata layout.** Marshaled metadata is the three-byte magic
   `cmp`, one algorithm byte, then a uvarint of the original size —
   nothing else. UnmarshalMetadata parses it back losslessly. Covered
   by `TestDetail01`.
2. **Sniffer contract (shape).** UnmarshalMetadata never errors on
   input that is too short, has the wrong magic, or carries an
   algorithm byte other than S2 — all of those return zero bytes
   consumed and nil error. Only a malformed uvarint after a valid
   header returns an error. Covered by `TestDetail02`.
3. **Checksum trailer.** Compress and Decompress treat the last
   checksumSize bytes as a plaintext trailer that is preserved
   verbatim; a buffer shorter than the trailer errors; the no-op
   algorithm is identity in both directions. Covered by `TestDetail03`.
4. **Block decode order.** decode returns a buffer with no valid
   metadata as uncompressed; a metadata-bearing block is decompressed;
   a corrupt compressed block errors; and an uncompressed record whose
   length collides with the magic-plus-S2 header is still read back
   correctly through the store. Covered by `TestDetail04`.
5. **Enum codec.** String renders None/S2/Unknown StoreCompression;
   JSON marshals to lowercase none/s2; any other algorithm value or
   string errors in both directions. Covered by `TestDetail05`.
6. **Error surfacing (shape).** A corrupt compressed stream makes
   Decompress return a non-nil error; well-formed input never errors.
   Covered by `TestDetail06`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | yes / partially |
| TestDetail06 | 6 | partially — shape only |
