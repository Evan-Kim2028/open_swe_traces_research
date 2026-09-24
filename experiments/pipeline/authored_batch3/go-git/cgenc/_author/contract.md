# Contract — cgenc

Hidden suite: `tests/hidden/plumbing/format/commitgraph/cgenc_bb_test.go`
(package `commitgraph`, in-package). One `TestDetailNN` per DETAILS.md line.
All tests parse the emitted byte stream directly (header, TOC, chunk slices,
checksum) — no reliance on unexported symbols.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | Bytes 0-3 are `CGPH`; byte 4 = 1; byte 5 = 1 (hash version for the 20-byte hash in use); byte 7 = 0; header chunk count equals the number of TOC entries minus the terminator. The >255-chunk refusal is not exercised (impractical input size). |
| TestDetail02 | 2 | doc | TOC entries are 4-byte sig + u64 absolute offset; first offset = `8 + (nChunks+1)*12`; last TOC sig is all-zero and its offset = `len(file)-20` (checksum start); offsets ascend. |
| TestDetail03 | 3 | partially | A minimal index emits exactly `OIDF OIDL CDAT` + terminator — fixed order, no optional chunks. |
| TestDetail04 | 4 | partially | `OIDF` is 1024 bytes; `fanout[i]` is cumulative (hashes with first byte ≤ i) — checked at boundaries 0, 0x0f, 0x10, 0xfe, 0xff; `OIDL` holds the hashes sorted bytewise. |
| TestDetail05 | 5 | doc | `CDAT` row = 36 bytes: tree hash, two u32 parent slots, u64 packing `generation<<34 \| unixTime` — generation in the top 30 bits, time in the low 34. |
| TestDetail06 | 6 | doc | Parent slot = parent's sorted position (0); missing parent = `0x70000000`; root commit has both slots = `0x70000000`; a parent hash not in the index makes `Encode` return an error. |
| TestDetail07 | 7 | partially | 4-parent commit: slot 2 has MSB set and low bits index into `EDGE`; `EDGE` entries at that index are parent positions 1, 2, and `3\|0x80000000` (last-marker bit on the final entry). |
| TestDetail08 | 8 | no | SHAPE: a generation-v2 value ≥ 2^31 produces a `GDA2` chunk whose u32 slot carries the MSB flag and indexes a u64 overflow area appended after the last TOC offset, holding the full value; a fitting v2 value sits unflagged in its slot. |
| TestDetail09 | 9 | doc | Last 20 bytes = SHA-1 of every preceding byte. |
| TestDetail10 | 10 | no | SHAPE: encoding the same index twice is byte-identical; a minimal index emits no `EDGE`/`GDA2`/`GDO2` chunks. |
| TestDetail11 | 11 | no | SHAPE: two oversized generation-v2 values produce a 16-byte overflow area containing both u64s in file order. The input-slice aliasing mechanic itself is unobservable and not asserted. |
| TestDetail12 | 12 | no | SHAPE: an empty index encodes without error and still emits `OIDF` (1024 zero bytes), `OIDL` and `CDAT` (both zero-length). |

Refusals/softening: line 1's >255-chunk refusal is not asserted (constructing >255 chunks
needs >255 commits — possible but the observable check would dwarf its value; flagged here).
Line 8 does not pin a `GDO2` TOC signature or slot-vs-list numbering — only that the full
u64s are recoverable in order from a flagged-slot scheme. Line 11's slice-reuse is internal.
