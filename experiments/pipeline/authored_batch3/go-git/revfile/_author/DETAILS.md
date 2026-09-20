# Details — revfile

1. File layout: `RIDX` magic, big-endian u32 version (only 1 accepted), u32 hash-function id
   (1 = SHA-1, 2 = SHA-256, anything else unsupported), then the entries, the packfile
   checksum, and a trailing checksum over everything before it. Inferable: partially.
2. Decode streams each entry to the output channel as it is read — in file order — and closes
   the channel when finished, INCLUDING on failure (the caller must never close it).
   Inferable: doc for the close; `partially` for the on-error close.
3. The file's hash-function id drives the checksum width AND the algorithm — a SHA-256 rev
   file carries 32-byte checksums; the id field is hashed into the running checksum too.
   Inferable: partially.
4. A declared object count of zero is a dedicated empty-index failure read BEFORE any entry
   is touched — not a clean empty decode. Inferable: no.
5. The stored packfile checksum must equal the caller-supplied one byte-for-byte, and must be
   exactly one hash-length — a short or long blob is malformed, a mismatch is malformed with
   both hashes quoted. Inferable: partially.
6. Bytes after the trailing checksum are malformed — the reader demands end-of-file, not just
   a complete record. Inferable: no.
7. The running checksum is verified against the stored trailer — a bit-flip anywhere in
   header, entries or pack checksum is caught. Inferable: partially.
8. Encode rejects a nil writer AND a typed-nil writer (a non-nil interface holding a nil
   pointer is detected via reflection). Inferable: no.
9. Encode derives the hash-function id from the SIZE of the caller's hash.Hash — 32 bytes
   selects SHA-256, every other size selects SHA-1 — it never inspects the algorithm.
   Inferable: no.
10. The caller's hasher is reset before use — state it carried in is discarded. Inferable: no.
11. The reverse index maps pack-offset order to index position: entries are written sorted by
    pack offset but each VALUE is that object's position in the hash-sorted index.
    Inferable: doc — the builder's comment states it.
12. A short read of the pack checksum (wrong byte count) is malformed even when the bytes
    themselves match the prefix. Inferable: no.
