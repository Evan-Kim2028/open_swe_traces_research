# Bug report — blkcodec

**Title:** Compressed blocks fail to reopen; uncompressed blocks whose
length mimics the metadata magic are misread as corrupt.

**Symptoms:**
- After restart, an S2-compressed block store fails recovery: the
  block sniff sees any buffer starting with `cmp` as compressed and
  attempts decompression, so uncompressed blocks of exactly
  24,145,251 bytes are reported corrupt instead of falling back to
  plain decode.
- `Compress` compresses the checksum trailer too, so integrity checks
  on read always fail.
- `UnmarshalMetadata` returns a hard error on short buffers, so
  truncated-but-plain blocks become unreadable instead of passing
  through as uncompressed.
- JSON config `"compression": "S2"` (uppercase) is silently accepted
  while `"s2"` is rejected.

**Reproduction:** write a stream message sized to land a 24,145,251-byte
record uncompressed, restart, LoadMsg — gold returns the message, the
defective decode returns a corruption error.

**Expected:** sniff-not-fail metadata, checksum kept outside the s2
stream, exact collision fallback in `decode`, strict lowercase JSON.
