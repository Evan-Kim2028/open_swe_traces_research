# Closure — memstor

Package: `storage/memory`. File: `storage.go`.

Removed (41 functions stubbed): the entire storage surface —
`NewStorage`, `SetObjectFormat`, `SupportsExtension`, config/index
getters/setters, `lazyCloser.Close`, `RawObjectWriter`,
`NewEncodedObject`, `SetEncodedObject`, `HasEncodedObject`,
`EncodedObjectSize`, `EncodedObject`, `IterEncodedObjects`,
`flattenObjectMap`, `Begin`, `ForEachObjectHash`, pack/loose/alternate
stubs, the `TxObjectStorage` lifecycle, all of `ReferenceStorage`
including `CheckAndSetReference`, `ShallowStorage`, `ModuleStorage`,
`ReflogStorage`.

Kept: all types and field layouts, `ErrUnsupportedObjectType`,
`errNotSupported`, doc comments. Disjoint from `idxindex` (pack index)
and `indexops`/`indexdec`/`indexenc` (the index format) — this is the
in-memory storer backend itself.

Tests deleted: `storage/memory/storage_test.go` (1 — the package's
only test file).
