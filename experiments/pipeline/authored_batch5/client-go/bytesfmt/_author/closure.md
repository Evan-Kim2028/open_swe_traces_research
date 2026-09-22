# Closure — bytesfmt

Package: `util`. File: `util/misc.go` (212 lines).

Removed (all bodies stubbed): `CompatibleParseGCTime`, `FormatBytes`,
`getByteUnit`, `BytesToString`, `String`, `ToUpperASCIIInplace`,
`EncodeToString`, `HexRegionKey`, `HexRegionKeyStr`.

Kept: `GCTimeFormat`/`gcTimeFormatOld`/`byteSize*` consts, `WithRecovery`,
`SetSessionID`, `SessionID` key (context/recovery helpers are outside the
formatting closure).

Tests edited: `util/misc_test.go` deleted — its only test is
`TestCompatibleParseGCTime`, dedicated to this closure.
