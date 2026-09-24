# Details — memstor

1. Objects are indexed in a per-type map as well as the main map —
   `IterEncodedObjects(commit)` scans only commits, never the full map.
   Inferable: partially — the field layout is visible.
2. `SetEncodedObject` on an unknown type returns ErrUnsupportedObjectType
   but still stores it in the main map. Inferable: no.
3. `EncodedObject` honors the type filter — an existing object of a
   different type reports ErrObjectNotFound, and AnyObject skips the
   check. Inferable: partially.
4. `RawObjectWriter` defers storage to Close — the object lands in the
   map only when the writer's Close runs, via the lazyCloser. Inferable:
   partially.
5. A transaction's objects are invisible to the base storage until
   Commit — `tx.EncodedObject` reads only the transaction's own map.
   Inferable: partially.
6. `CheckAndSetReference` is a CAS: it fails with ErrReferenceNotFound
   when no current ref exists and ErrReferenceHasChanged when the stored
   hash differs from `old` — a nil `old` sets unconditionally, and a nil
   `ref` is a no-op. Inferable: doc-adjacent — the method comment says
   "if the old reference matches".
7. `SetIndex` stamps `idx.ModTime` to now — memory storage pretends to
   be a filesystem index so the racy-git metadata optimization engages.
   Inferable: doc — the comment says it simulates filesystem behavior.
8. `Index`/`Config` lazily materialize a default value on first read —
   never nil. Inferable: partially.
9. `ForEachObjectHash` treats storer.ErrStop as a clean end, not an
   error. Inferable: partially — standard storer convention.
10. `Module` memoizes — the same name returns the same Storage instance
    across calls. Inferable: partially.
11. `SetObjectFormat` on a populated store is an error — the format can
    only be set on empty storage, and only sha1/sha256 are legal.
    Inferable: partially.
12. Loose-object operations and alternates are uniformly not supported
    — ObjectPacks is empty, pack GC is a no-op. Inferable: doc.
