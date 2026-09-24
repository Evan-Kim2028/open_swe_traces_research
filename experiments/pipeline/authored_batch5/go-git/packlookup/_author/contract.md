# Contract — packlookup

`plumbing/format/packfile.Packfile`/`FSObject` — descriptor ownership,
single-run init, hash→offset resolution, delta base lookup, lazy
filesystem objects, type-filtered iteration, and close discipline.
Every commitment below is covered by a hidden test; every hidden test
maps to a commitment.

## Commitments

1. **Resolver ownership.** With `WithPackHandle` the file argument is
   closed by the constructor and the resolver-owned handle is never
   closed by `Packfile.Close`. Covered by `TestDetail01`.
2. **Single init.** The handle is resolved once across repeated `Get`
   calls; `ID` returns the pack's trailing checksum — via the handle's
   `PackHash` under a resolver, via a tail read of the file otherwise;
   a bad signature surfaces as an init error on first use. Covered by
   `TestDetail02`.
3. **Post-lock closed check.** A `Get` racing `Close` returns
   `fs.ErrClosed` (or completes) — never a torn-scanner error — and
   after `Close` all reads fail `fs.ErrClosed`. Covered by
   `TestDetail03`.
4. **Lookup semantics.** `Get` resolves hash→offset through the index
   and answers `ErrObjectNotFound` for a hash the index doesn't know.
   Covered by `TestDetail04`.
5. **Lazy FS objects.** With `WithFs`, `Get` returns an `*FSObject`
   whose `Reader` streams the inflated content; without it, a memory
   object comes back. Covered by `TestDetail05`.
6. **Delta type resolution.** A delta's stored type is its base's
   type — a resolved OFS delta reports `BlobObject` and its content
   inflates correctly. Covered by `TestDetail06`.
7. **Both delta kinds.** OFS-delta and REF-delta packs both resolve
   their bases to the correct type and content — the two lookup paths
   are exercised independently. Covered by `TestDetail07`.
8. **Missing base (shape).** A delta entry whose base hash is absent
   from the pack/index fails `Get` with an error rather than silently
   inflating. Covered by `TestDetail08`.
9. **Probe/reopen (shape).** The 1-byte probe distinguishes a live
   descriptor from a closed one; when the packfile's FD is closed
   underneath an `FSObject`, `Reader` reopens the pack by path and
   still serves the object bytes. Covered by `TestDetail09`.
10. **Concurrent readers.** Concurrent `Get`+`Reader` on shared
    filesystem objects all return correct content — reads go through
    `ReadAt`, never a shared seek. Covered by `TestDetail10`.
11. **Iterator type resolution.** `GetByType(BlobObject)` yields delta
    entries resolved to their base type alongside plain blobs;
    `GetByType(CommitObject)` skips them without error. Covered by
    `TestDetail11`.
12. **Idempotent Close.** `Close` runs once: in legacy mode the file
    argument is closed exactly once; under a resolver the scanner
    cursor is released while the handle is untouched. Covered by
    `TestDetail12`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | doc |
| TestDetail06 | 6 | partially |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | partially — error presence, not its text |
| TestDetail09 | 9 | no — probe shape + observable reopen |
| TestDetail10 | 10 | doc |
| TestDetail11 | 11 | partially |
| TestDetail12 | 12 | doc |
