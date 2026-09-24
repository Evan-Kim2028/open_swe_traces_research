# Contract — memstor

`storage/memory.Storage` — the whole in-memory storer: per-type object
maps, deferred raw-writer landing, transactions, CAS references,
index/config lazy materialization, modules, and the not-supported
surface. Every commitment below is covered by a hidden test; every
hidden test maps to a commitment.

## Commitments

1. **Per-type indexing.** Stored objects land in the matching per-type
   map (`Blobs`, `Commits`, …) as well as `Objects`, and
   `IterEncodedObjects` yields only the requested type. Covered by
   `TestDetail01`.
2. **Unknown-type store (shape).** `SetEncodedObject` on an unsupported
   type returns `ErrUnsupportedObjectType` yet the object still lands in
   the main map and stays retrievable. Covered by `TestDetail02`.
3. **Type-filtered lookup.** `EncodedObject` reports
   `ErrObjectNotFound` for a stored object queried under a different
   type; `AnyObject` skips the check. Covered by `TestDetail03`.
4. **Deferred raw write.** `RawObjectWriter` does not publish the object
   until the writer's `Close` runs — `HasEncodedObject` is false before
   and true after. Covered by `TestDetail04`.
5. **Transaction isolation.** Objects added to a transaction are
   invisible to base storage and visible inside the transaction;
   `Commit` publishes them, `Rollback` discards them. Covered by
   `TestDetail05`.
6. **Reference CAS.** `CheckAndSetReference` returns
   `ErrReferenceNotFound` when no current ref exists with a non-nil
   `old`, `ErrReferenceHasChanged` on a hash mismatch (leaving the stored
   ref untouched), sets unconditionally on nil `old`, and treats a nil
   `ref` as a no-op. Covered by `TestDetail06`.
7. **SetIndex stamps ModTime.** Storing an index sets its `ModTime` to
   approximately now, simulating filesystem behaviour for the racy-git
   fast path. Covered by `TestDetail07`.
8. **Lazy materialization.** `Index()` and `Config()` on a fresh
   storage return non-nil defaults; repeated `Index()` returns the same
   materialized instance. Covered by `TestDetail08`.
9. **ErrStop convention.** `ForEachObjectHash` treats `storer.ErrStop`
   as a clean end (nil error) while other callback errors propagate.
   Covered by `TestDetail09`.
10. **Module memoization.** `Module(name)` returns the same `Storage`
    instance for the same name and distinct instances for distinct
    names. Covered by `TestDetail10`.
11. **Format switching.** `SetObjectFormat` accepts sha1/sha256 on an
    empty store, rejects an unrecognised format, and errors on a
    populated store. Covered by `TestDetail11`.
12. **Unsupported surface.** `ObjectPacks` is empty, pack GC is a no-op
    returning nil, and loose-object/alternate operations return errors.
    Covered by `TestDetail12`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | no — shape only |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | doc-adjacent |
| TestDetail07 | 7 | doc |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | partially |
| TestDetail10 | 10 | partially |
| TestDetail11 | 11 | partially |
| TestDetail12 | 12 | doc |
