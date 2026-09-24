# Closure — packparse

Package: `plumbing/format/packfile`. Files: `parser.go`, `parser_cache.go`.

Removed (20 functions stubbed): `growHint`, `NewParser`,
`Parser.storeOrCache`, `Parser.resetCache`, `Parser.Parse`,
`Parser.ensureContent`, `Parser.resolveDeltas`, `Parser.processDelta`,
`checkDeltaChainDepth`, `ObjectHeader.isDeltaOnDisk`, `Parser.parentReader`,
`Parser.applyPatchBaseHeader`, `Parser.forEachObserver`, `Parser.onHeader`,
`Parser.onInflatedObjectHeader`, `Parser.onInflatedObjectContent`,
`Parser.onFooter`; `newParserCache`, `parserCache.Add`, `parserCache.Reset`.

Kept: all error vars (`ErrReferenceDeltaNotFound`, `ErrNotSeekableSource`,
`ErrDeltaNotCached`, `ErrParserConsumed`), the `maxObjectPreallocBytes` /
`maxDeltaChainDepth` / `maxObjectsPrealloc` consts with rationale comments,
`Parser`/`LowMemoryCapable`/`parserCache`/`Observer` type declarations, the
`resolveDeltas`/`checkDeltaChainDepth` doc comments (they carry the
upstream-cited algorithm), and the whole scanner + patch-delta sibling code
(visible read/apply path).

Tests deleted: all 19 `*_test.go` in the package.
