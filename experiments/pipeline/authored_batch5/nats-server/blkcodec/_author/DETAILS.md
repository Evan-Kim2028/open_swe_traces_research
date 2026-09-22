# Details — blkcodec

1. Metadata layout: `"cmp"` + 1 algorithm byte + uvarint OriginalSize,
  emitted into a 14-byte scratch and sliced to `4+n`. Inferable:
  partially — the magic and uvarint are only visible in code/tests.
2. `UnmarshalMetadata` is a SNIFFER: `<5` bytes, wrong magic, or
  algorithm byte != S2 all return `(0, nil)` — not an error — because
  an uncompressed record of length 7,368,035 starts with the bytes
  `"cmp"`. Only a bad uvarint after a valid header errors
  ("metadata incomplete"). Inferable: partially — the no-error
  contract is load-bearing.
3. `Compress`/`Decompress` treat the LAST `checksumSize` bytes as a
  plaintext trailer: s2 covers `buf[:len-checksumSize]` only; the
  checksum is re-appended verbatim. `len(buf) < checksumSize` → error.
  `NoCompression` is identity. Inferable: partially — the
  uncompressed-checksum design is an internal format invariant.
4. `msgBlock.decode` order: UnmarshalMetadata → n==0 means return buf
  as NoCompression; else Decompress. On decompress failure with
  S2Compression AND `len(buf) >= 24145251`, it tries
  `msgFromBufNoCopy` — a record of that exact length collides with a
  valid `"cmp"+alg=1` header — success means the block was really
  uncompressed. Otherwise the ORIGINAL decompress error is returned.
  Inferable: partially — the magic-length fallback is a quirk pinned
  by TestFileStoreCompressionHeaderCollision.
5. Enum strings: `String()` → "None"/"S2"/"Unknown StoreCompression";
  JSON uses lowercase `"none"`/`"s2"`, anything else errors on both
  directions. Inferable: yes for JSON (wire-visible), partially for
  String casing.
6. Compress/Decompress propagate writer/reader errors wrapped (`%w`)
  and report short writes/copies explicitly. Inferable: partially.
