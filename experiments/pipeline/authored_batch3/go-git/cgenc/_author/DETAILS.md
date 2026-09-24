# Details — cgenc

1. The file opens `CGPH` then four bytes `{1, hashVersion, chunkCount, 0}` — hashVersion is
   2 only for a 32-byte hash — and more than 255 chunks is refused. Inferable: partially —
   the signature const is visible, the header bytes are not.
2. The chunk table is `4-byte signature + u64 ABSOLUTE offset` pairs — the first offset
   counts the file header AND the terminator entry — closed by the zero chunk signature
   with the end-of-data offset. Inferable: doc — `doc.go` lays the table out.
3. Chunk order is fixed: fanout, oid lookup, commit data, then optionally extra edges,
   generation data, generation overflow — the optional chunks exist ONLY when needed.
   Inferable: partially.
4. Fanout is 256 cumulative u32 counts keyed on the first byte of each hash over a bytewise-
   sorted hash list — `fanout[i]` is the count of hashes ≤ first-byte i, not == i.
   Inferable: partially.
5. Each commit-data row is tree hash + two u32 parent slots + one u64 packing
   `generation<<34 | unixTime` — generation rides the TOP 30 bits, not a separate field.
   Inferable: doc — `doc.go` and the reader show the row.
6. Parent slots hold the parent's POSITION in the sorted hash list; missing parent = the
   visible `parentNone` constant; a parent not in the index is an error, never a silent
   zero. Inferable: doc — the kept `lookupParentIndex` comment explains why.
7. More than two parents: slot 2 becomes `extraListIndex | 0x80000000`, the extra list
   carries parents 2..N-1 as plain positions, and the LAST entry is OR'd with the
   last-marker bit. Inferable: partially — the marker consts are visible.
8. Generation-v2 values ≥ 2^31 can't fit a slot — they write `rowIndex | 0x80000000` and
   queue the full u64 into the overflow chunk in FILE order. Inferable: no.
9. The trailing checksum covers every byte written from the magic on — the tee wiring in
   `NewEncoder` makes that legible. Inferable: doc.
10. Chunks are counted/sized BEFORE writing — the presence of the extra-edges and overflow
    chunks is decided from a pre-pass over the data, not discovered mid-stream. Inferable:
    no.
11. The overflow list reuses the generation array's front as staging — the returned slice
    is `[:head]` of the input. Inferable: no.
12. An empty index still writes all three mandatory chunks with zero-length data sections —
    not an error. Inferable: no.
