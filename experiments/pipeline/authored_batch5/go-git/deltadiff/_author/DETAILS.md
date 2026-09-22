# Details — deltadiff

1. A delta stream starts with the SOURCE size then the TARGET size, each
   little-endian LEB128 — before any instructions. Inferable: partially —
   the EncodeLEB128 helper is visible, the ordering is the detail.
2. Insert instructions carry a literal length byte (high bit clear) and are
   chunked at 127 bytes — a longer literal run is split, never merged into
   one opcode. Inferable: partially — 127 appears in no kept doc.
3. Copy instructions set the high bit and pack offset into up to four flag-
   selected bytes plus length into up to three — a zero offset/length byte
   position is simply omitted from the stream. Inferable: partially — the
   opcode layout is inferable from the kept signature shape plus format
   conventions, the flag-bit ordering is the detail.
4. A copy longer than the 64KB opcode ceiling is emitted as repeated
   maxCopySize copies advancing the offset — not truncated, not clamped.
   Inferable: doc — the const comment and upstream link are kept.
5. A match shorter than the 16-byte fingerprint block is emitted as
   literal insert bytes rather than a copy — the sub-block tail is never
   indexed. Inferable: partially — `s` and `blksz` consts are visible, the
   tie-break is the detail.
6. When the source is smaller than the block size the whole target falls
   back to literals — findMatch signals it with a negative length and the
   encoder copies the rest verbatim. Inferable: partially — the sentinel
   is named in the kept doc comment.
7. When fewer than a block's bytes remain in the target, findMatch returns
   the remaining length directly — the tail is "matched" without consulting
   the index. Inferable: partially.
8. Pending literal bytes are flushed BEFORE each copy instruction — insert
   opcodes never interleave inside a single copy run. Inferable: partially.
9. The index scans the source BACKWARD block by block, and a block hashing
   to the same key as the previous block overwrites that chain head rather
   than appending — consecutive equal blocks collapse to their newest
   offset. Inferable: no.
10. Hash chains longer than the 64-entry cap are severed during the census
    pass — the scan stays linear instead of quadratic — and entries are
    relinked contiguously with the hash table pointing at the first slot.
    Inferable: doc — the maxChainLength const and the linear-complexity
    comment are kept.
11. The hash table is sized to the next power of two at least the worst-case
    block count, via a hand-rolled leading-zeros — never a smaller table.
    Inferable: partially — tableSize's shape is visible, the pow2 policy is
    the detail.
12. GetDelta returns a MemoryObject typed OFSDeltaObject whose size is the
    delta byte length — not the base's, not the target's. Inferable:
    partially.
13. Reader failures on either object propagate before any delta bytes are
    produced, and both readers are closed through the shared CheckClose
    path. Inferable: partially — ioutil.CheckClose is visible.
