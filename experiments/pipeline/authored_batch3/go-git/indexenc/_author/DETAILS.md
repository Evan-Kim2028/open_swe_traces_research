# Details — indexenc

1. The file opens with the four-byte `DIRC` signature, the version, and the entry count as
   big-endian u32s; a version above the supported bound fails before anything is written.
   Inferable: doc — the signature constant and bound are visible, and the sibling decoder
   shows the header layout.
2. Entries are written sorted by name and then by stage — the caller's slice order is
   discarded — and two entries with the same name come out in stage order. Inferable: partially.
3. Each entry's fixed part is ten u32s (ctime s/ns, mtime s/ns, dev, ino, mode, uid, gid,
   size), then the hash, then the 16-bit flags. Inferable: doc — the intact decoder reveals
   the order.
4. Flags carry stage in bits 12-13 and the name length in the low 12 bits, SATURATING at
   0xFFF for long names rather than failing. Inferable: partially.
5. Intent-to-add and skip-worktree set an extended-flags bit and append a SECOND u16 with the
   two extra bits — the second flags word exists only when one of them is set. Inferable:
   partially.
6. A zero timestamp encodes as a 0/0 pair without error; a NEGATIVE seconds or nanoseconds
   value is an invalid-timestamp failure. Inferable: partially.
7. Versions 2 and 3 write the raw name bytes then pad the whole entry (fixed part + hash +
   name) to an 8-byte boundary with NULs — the padding always includes at least one NUL so
   the name stays terminated. Inferable: partially.
8. Version 4 writes NO padding; instead each name is prefix-compressed: a varint count of
   bytes to strip from the END of the previous entry's name, then the remaining suffix plus a
   NUL. Inferable: doc — the v4 comment in the kept source sketches it.
9. The FIRST v4 entry strips nothing and carries its full name — there is no previous name to
   compress against. Inferable: partially.
10. The padding length counts the 2-byte extended-flags word when present — it is part of the
    entry, not an extra. Inferable: no.
11. The trailer is the hash of every byte written; under the skip-hash option the writer
    skips the running-hash tee and appends a hash-length run of ZERO bytes instead.
    Inferable: no.
12. A raw extension is `4-char signature + u32 length + payload` — a signature of any other
    length is rejected. Inferable: partially.
