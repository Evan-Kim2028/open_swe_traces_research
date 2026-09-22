# Details — packlookup

1. WithPackHandle makes the resolver own the descriptor — the
   constructor closes the file argument and Close never touches the
   resolver's handle. Inferable: doc — the option comment says so.
2. init runs once: resolves the handle, scans the pack signature, reads
   the trailing checksum as the pack ID (via the handle's PackHash when
   a resolver is in play, else a seek-to-end read), and only then serves
   reads. Inferable: partially.
3. Every read path re-checks closed AFTER taking the mutex — a
   concurrent Close between the early check and the lock returns
   fs.ErrClosed rather than racing a torn-down scanner. Inferable:
   partially — the double-check is subtle.
4. Get resolves hash→offset through the index, seeks the scanner to the
   entry, and reads the header; a scan that ends cleanly without an
   object header is ErrObjectNotFound, not the scanner's error.
   Inferable: partially.
5. Non-delta objects on a filesystem-backed pack are returned as
   lazy FSObjects — the bytes are not inflated until Reader is called.
   Deltas always inflate through getMemoryObject. Inferable: doc.
6. A delta object's stored type is replaced by its BASE's type — the
   returned object reports the resolved type, not OFSDelta/REFDelta.
   Inferable: partially.
7. REFDelta bases resolve through the hash index first (cache, then
   Get); OFSDelta bases resolve by offset — the two delta kinds use
   different base lookups. Inferable: partially.
8. A delta whose base can't be found errors — partial packs with
   missing bases are not silently inflated. Inferable: partially.
9. FSObject.Reader probes the descriptor with a 1-byte ReadAt and
   reopens the pack by path when the FD was closed underneath it —
   never on transient errors. Inferable: no — probe/reopen is internal.
10. FSObject readers are concurrency-safe because they read through
    SectionReader/ReadAt — never Seek on the shared handle. Inferable:
    doc — Reader's comment states it.
11. The iterator resolves delta headers to their base type when
    filtering by type — a delta entry counts as its resolved type, and
    non-matching entries are skipped, not errored. Inferable: partially.
12. Close is idempotent and only closes what the packfile owns — the
    scanner cursor under a resolver, or the file itself otherwise.
    Inferable: doc.
