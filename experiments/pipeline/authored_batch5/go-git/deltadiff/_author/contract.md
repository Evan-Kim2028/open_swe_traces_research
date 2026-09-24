# Contract — deltadiff

Delta generation: the block-hash index, the rolling scanner, and the
insert/copy opcode encoder. Every commitment below is covered by a hidden
test; every hidden test maps to a commitment.

## Commitments

1. **Stream header.** A delta stream starts with the source size then the
   target size, each little-endian LEB128, before any instruction. Covered
   by `TestDetail01`.
2. **Insert chunking.** Insert instructions carry a literal length byte
   with the high bit clear and are chunked at 127 bytes — a longer literal
   run is split, never merged into one opcode. Covered by `TestDetail02`.
3. **Copy encoding.** Copy instructions set the high bit and pack offset
   into flag-selected bytes plus length into flag-selected bytes; a decoded
   copy lands on the matching source bytes. Covered by `TestDetail03`.
4. **Copy ceiling.** A copy longer than the 64KB opcode ceiling is emitted
   as repeated bounded copies — not truncated, not clamped, and together
   covering the whole match. Covered by `TestDetail04`.
5. **Sub-block matches.** A match shorter than the fingerprint block is
   emitted as literal insert bytes — the sub-block tail is never indexed.
   Covered by `TestDetail05`.
6. **Tiny-source fallback.** When the source is smaller than the block size
   the whole target falls back to literals — the match probe signals it with
   a negative length. Covered by `TestDetail06`.
7. **Tail handling.** When fewer than a block's bytes remain in the target,
   the probe returns the remaining length directly. Covered by
   `TestDetail07`.
8. **Flush ordering.** Pending literal bytes are flushed before each copy
   instruction — decoded ops reproduce the target sequentially, and a
   literal run precedes the copy that follows it. Covered by `TestDetail08`.
9. **Backward scan (shape).** The index scans the source backward block by
   block; for identical non-consecutive blocks the match probe reports the
   earliest occurrence (the last block scanned becomes the chain head). The
   consecutive-collapse relinking is internal; the observable shape is which
   occurrence is returned. Covered by `TestDetail09`.
10. **Chain severing.** Hash chains longer than the documented cap are
    severed — a source of many identical blocks still indexes and matches
    correctly. Covered by `TestDetail10`.
11. **Table sizing.** The hash table is sized to a power of two at least the
    worst-case block count; the leading-zeros helper matches the standard
    bit count. Covered by `TestDetail11`.
12. **Result shape.** The delta-producing entrypoint returns an in-memory
    object typed as a delta whose size is the delta byte length — not the
    base's, not the target's. Covered by `TestDetail12`.
13. **Error propagation.** Reader failures on either object propagate before
    any delta bytes are produced, and both readers are closed. Covered by
    `TestDetail13`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | doc |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | partially |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | no — shape only |
| TestDetail10 | 10 | doc |
| TestDetail11 | 11 | partially |
| TestDetail12 | 12 | partially |
| TestDetail13 | 13 | partially |

Note: the hidden suite also defines `randBytes`, a helper referenced by
surviving in-tree tests whose original definition was removed by the
excision; without it the package's test binary does not build.
