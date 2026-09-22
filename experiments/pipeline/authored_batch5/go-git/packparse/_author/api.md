# Exported API — packparse

Package `plumbing/format/packfile` (module `example.internal/gitkit/v6`) —
full packfile decoder: drives the scanner, resolves OFS/REF delta chains
(incl. thin-pack external bases), notifies `Observer`s, returns the pack
checksum.

`NewParser(io.Reader, ...ParserOption)`, `Parser.Parse() (Hash, error)`;
sentinels `ErrReferenceDeltaNotFound`, `ErrNotSeekableSource`,
`ErrDeltaNotCached`, `ErrParserConsumed`; `LowMemoryCapable` iface; consts
`maxObjectPreallocBytes`, `maxDeltaChainDepth`, `maxObjectsPrealloc`.

Callers: index building, pack ingestion on fetch/clone. In-tree tests
removed: 19.
