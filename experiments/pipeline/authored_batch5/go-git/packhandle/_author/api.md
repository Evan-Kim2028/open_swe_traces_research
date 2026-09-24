# Exported API — packhandle

Package `internal/packhandle` — `PackHandle`, the refcounted reader over
one pack triple (.pack/.idx/.rev): cursor constructors, pack-size cache,
verified `Meta`, lazy `Index`, close/idle-release lifecycle, `cursorReader`
(Read/ReadAt/Seek/Close), `PathSource` and the `Source`/`Sources` openers.

`New`, `NewWithPool`, `OpenPackReader`, `OpenRandomReader`, `Meta`,
`Index`, `Close`, `CloseIdleDescriptors`; `parsePackMeta`;
error vars in `errors.go` kept.

Caller: filesystem storage packfiles. In-tree tests removed: all 8
packhandle test files.
