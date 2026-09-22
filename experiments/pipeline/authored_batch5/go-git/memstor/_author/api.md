# Exported API — memstor

Package `storage/memory` — the whole in-memory storer: `NewStorage`,
`Storage.SetObjectFormat`/`SupportsExtension`, `ConfigStorage`,
`IndexStorage`, `ObjectStorage` (raw writer, per-type lookup,
iteration, `ForEachObjectHash`), `TxObjectStorage` (commit/rollback),
`ReferenceStorage` (incl. `CheckAndSetReference`), `ShallowStorage`,
`ModuleStorage`, `ReflogStorage`, `lazyCloser`.

Kept visible: all struct/map types and field layouts,
`ErrUnsupportedObjectType`/`errNotSupported`, doc comments (including
the racy-git note on `SetIndex`).

Callers: nearly every test harness and ephemeral repository. In-tree
tests removed: 1 (`storage_test.go`).
