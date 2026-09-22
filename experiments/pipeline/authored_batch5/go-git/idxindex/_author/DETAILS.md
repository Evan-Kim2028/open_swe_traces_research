# Details — idxindex

1. Hash lookup buckets by first byte through the fanout, then binary-
   searches only that bucket's names — misses inside a present bucket
   report not-found without touching other buckets. Inferable: partially —
   the fanout layout is documented on the struct fields.
2. MayContain answers from the fanout mapping alone — a hash whose first
   byte maps to an empty bucket is excluded without any name comparison.
   Inferable: doc — the method comment states the empty-bucket rule.
3. A 32-bit offset whose top bit is set is not an offset — the low 31 bits
   index the 64-bit overflow table. Inferable: partially — isO64Mask is a
   kept const, the indirection is the detail.
4. FindHash builds a reverse offset→hash map on first use — lazily, once,
   under a sync.Once — not per call. Inferable: no — the once-field is
   visible but the lazy-trigger is internal.
5. Entries walks fanout order (hash-sorted) while EntriesByOffset sorts a
   collected slice by pack offset — two different orderings, not views of
   one. Inferable: partially.
6. EntriesWithPrefix reads only the matching bucket and stops at the first
   name that no longer prefixes — it never scans other buckets. Inferable:
   doc — the iterator comment describes the early-stop.
7. LazyIndex resolves lookups by ReadAt arithmetic on the idx sections —
   fanout → names → crc → offsets — sharing one refcounted descriptor
   across concurrent readers, never loading the file. Inferable:
   partially — the section-offset fields are visible, the read sequence
   is the detail.
8. Lazy hash-to-offset uses the .rev file when present — positions map
   through rev entries, not a binary search over idx names. Inferable:
   no.
9. The Writer rejects Index() before OnFooter and on a count mismatch
   between the announced header and the objects actually added.
   Inferable: partially.
10. Add dedupes by hash — the same object twice keeps the first position.
    Inferable: no.
11. OnInflatedObjectContent records hash+position+crc and ignores the
    payload bytes; OnInflatedObjectHeader is a pure no-op. Inferable:
    partially — the observer contract is visible, the no-op is the
    detail.
12. Offsets beyond 32 bits are appended to the overflow table in
    encounter order and referenced by index, not value. Inferable:
    partially — addOffset64's signature is kept.
13. Iterators released on Close poison further Next calls or release the
    shared handles — Close is not idempotent-for-free across lazy
    iterators. Inferable: no.
14. The prefix iterator's slices alias the parent's bucket storage — it
    is invalid once the index is closed. Inferable: doc — the lifetime
    comment states the aliasing.
