# Contract — consumerstate

Binary encodings for JetStream consumer state and the replicated stream
snapshot frame, plus the delete-block helpers shared by both. Every
commitment below is covered by a hidden test; every hidden test maps to a
commitment.

## Commitments

1. **Consumer record layout.** The record is a magic byte, a version
   byte (the encoder emits version 2), then uvarints for the ack-floor
   pair, the delivered pair, and the pending count. Covered by
   `TestDetail01`.
2. **Pending entries.** A non-empty pending map is preceded by a
   mints timestamp anchor; each entry stores a stream-sequence delta
   against the ack floor, the consumer sequence, and a
   second-resolution signed timestamp offset — so a pending timestamp
   up to about a second in the future still decodes. Covered by
   `TestDetail02`.
3. **Redelivered entries.** The redelivered count is written even when
   pending is empty; entries are (stream-seq delta, count) uvarint
   pairs. Covered by `TestDetail03`.
4. **Exact-size output (shape).** The encoded buffer is sliced to the
   exact encoded size with no trailing slack. Covered by
   `TestDetail04`.
5. **Header gate.** checkConsumerHeader requires the magic byte and
   version 1 or 2; other inputs return an error (corrupt-state for a
   bad magic or short buffer). Covered by `TestDetail05`.
6. **Sequential decode.** Any failed varint mid-record yields a
   corrupt-state error rather than a panic or partial state. Covered by
   `TestDetail06`.
7. **Version-1 compat (shape).** A record carrying version 1 still
   decodes; its delivered pair is adjusted upward relative to the
   encoded values when the floor is above one. Covered by
   `TestDetail07`.
8. **Corruption guards.** Stream sequences with the top bit set, and a
   pending entry resolving to stream sequence zero, fail as corrupt;
   zero-sequence or zero-count redelivered entries are skipped.
   Covered by `TestDetail08`.
9. **Stream snapshot frame.** The frame is a magic byte, a version
   byte, then uvarints for Msgs, Bytes, FirstSeq, LastSeq, Failed;
   IsEncodedStreamState checks only magic, version, and minimum length.
   Covered by `TestDetail09`.
10. **Sources section.** The with-sources version writes a source count
    then per-source name/seq/ident before any deleted blocks. Covered
    by `TestDetail10`.
11. **Delete blocks (shape).** A deleted-block count is followed by
    per-block magic bytes: a run-length record decodes a (first, num)
    pair; an unknown magic errors. Covered by `TestDetail11`.
12. **DeleteBlock semantics.** DeleteRange.State yields (first,
    first+num-1, num) and Range iterates the contiguous range;
    DeleteSlice.State yields (first, last, len) with all zeros when
    empty and Range iterates slice order; NumDeleted sums blocks.
    Covered by `TestDetail12`.
13. **Varint sizing.** uvarintLen equals the length binary.PutUvarint
    writes; runLengthEncodeLen and appendRunLength agree on the emitted
    magic-plus-two-uvarints record. Covered by `TestDetail13`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | no — shape only (exact length, no slack) |
| TestDetail05 | 5 | yes |
| TestDetail06 | 6 | partially |
| TestDetail07 | 7 | no — shape only (decodes, delivered adjusted up) |
| TestDetail08 | 8 | doc |
| TestDetail09 | 9 | yes |
| TestDetail10 | 10 | partially |
| TestDetail11 | 11 | partially — unknown magic asserted as error shape |
| TestDetail12 | 12 | yes |
| TestDetail13 | 13 | doc |
