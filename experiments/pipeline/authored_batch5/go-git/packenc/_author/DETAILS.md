# Details — packenc

1. The trailing checksum is the object-format hash of every byte written so
   far — the writer tees through a hasher from the first header byte. The hash
   construction follows the repository's configured object format (SHA-256
   repos get a SHA-256 trailer), defaulting to SHA-1 when the storer exposes
   no config. Inferable: partially — format-aware hashing is visible in the
   kept hasher plumbing, the tee is internal.
2. `Encode` obtains the object list from the configured selector; a caller-
   supplied selector replaces only list production — the encoder's own delta
   selector still performs write-phase recovery. Inferable: doc — the option's
   doc comment states the split.
3. A delta whose base has not been written yet forces the base to be written
   FIRST, recursively — a delta never precedes its base in the stream.
   Inferable: partially — pack readers require it, the recursion point is the
   detail.
4. An object revisited while already marked for writing is silently restored
   to its original (undeltified) representation — the cycle is broken rather
   than recursed into forever. Inferable: doc — the marker method's doc names
   the delta-chain-loop purpose.
5. The written/not-yet tracking uses the offset field with a sentinel:
   unwritten is 0, want-write is exactly 1, written is anything above 1 — the
   boundary is strict (`>1`, not `>=1`). Inferable: no — sentinel arithmetic
   is an arbitrary choice.
6. OFS-delta headers carry the base's offset as a backward distance from the
   delta entry's own type byte, encoded with the offset-VLQ helper — a
   non-positive distance is an error, never written. Inferable: partially —
   the VLQ helper is visible, the distance direction is the detail.
7. Every delta in one pack uses a single kind: all OFS-delta by default, or
   all REF-delta when the flag is set — the encoder never mixes kinds in one
   stream even though readers accept mixed packs. Inferable: no.
8. REF-delta headers write the base's raw object ID; OFS-delta headers write
   the relative offset — the kind choice selects which. Inferable: partially.
9. The entry header is the type-tagged varint: object type in bits 4–6 of the
   first byte, the low 4 size bits beside it, then 7-bit size groups with the
   continuation bit — least-significant group first. Inferable: partially —
   the kept mask/shift constants reveal the layout, assembly order is the
   detail.
10. `ObjectToPack.Type`/`Hash`/`Size` answer from the ORIGINAL object when
    present, then from saved original metadata, then — for Type only — the
    base's type, and finally the delta object's own accounting; Hash/Size
    require the object to expose the delta interface at that point. Inferable:
    no — the fallback chain is internal bookkeeping.
11. `Type`/`Hash`/`Size` panic when no source can answer — there is no error
    return on the accessors. Inferable: no.
12. `SetOriginal` still snapshots type/size/hash when handed a nil object —
    the previously resolved metadata is kept, not cleared. Inferable: doc —
    the method's doc comment states it.
13. A delta's recorded depth is always its base's depth plus one at link time;
    de-deltifying resets it to zero. Inferable: partially.
14. The zlib stream for each entry is reset onto the pack writer per object —
    deflate output is interleaved with the outer hash tee, not buffered whole.
    Inferable: partially — the pooled writer is visible.
