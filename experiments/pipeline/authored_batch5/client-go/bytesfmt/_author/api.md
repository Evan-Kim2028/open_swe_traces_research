# Exported API — bytesfmt

Package `util` (module `example.internal/kvstore/v2`).

- `FormatBytes(numBytes int64) string` — human-readable size with pruned
  precision.
- `BytesToString(numBytes int64) string` — readable size, raw `%v` floats.
- `CompatibleParseGCTime(value string) (time.Time, error)` — parses
  gc_worker's stored time in old or new format.
- `EncodeToString(src []byte) []byte` — hex encode to bytes.
- `HexRegionKey(key []byte) []byte`, `HexRegionKeyStr(key []byte) string` —
  uppercase-hex region keys for logs.
- `ToUpperASCIIInplace(s []byte) []byte`, `String(b []byte) string`.
- `GCTimeFormat` const kept.

Callers: metrics/logging paths (`HexRegionKeyStr` in region logs), GC
safepoint loading (`CompatibleParseGCTime`), `String`/`EncodeToString` in
key formatting. In-tree test `misc_test.go` (`TestCompatibleParseGCTime`)
removed with the closure.
