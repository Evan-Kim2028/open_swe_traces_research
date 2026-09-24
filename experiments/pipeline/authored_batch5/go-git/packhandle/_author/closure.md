# Closure — packhandle

Package: `internal/packhandle`. Files: `packhandle.go`, `pack_meta.go`,
`cursor_reader.go`, `index.go`, `source.go`.

Removed (17 functions stubbed): `New`, `NewWithPool`,
`PackHandle.OpenPackReader`, `.OpenRandomReader`, `.packSize`, `.Close`,
`.doClose`, `.Meta`, `.CloseIdleDescriptors`, `.Index`; `parsePackMeta`;
`newCursorReader`, `cursorReader.Read/.ReadAt/.Seek/.Close`; `PathSource`.

Kept: `PackHandle`/`Source`/`Sources`/`PackMeta`/`cursorReader` types and
field layouts, all five error vars, `packMagic`, `defaultGracePeriod`,
`PackReader`/`RandomReader` interfaces, every doc comment — the
refcount/grace/pool contract is fully written down; `sharedfile` and
`fdpool` stay implemented (readable siblings).

Tests deleted: all 8 `*_test.go` in the package (they exist only for this
surface).
