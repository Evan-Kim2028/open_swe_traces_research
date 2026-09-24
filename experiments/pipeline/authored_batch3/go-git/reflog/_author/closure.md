# Closure — reflog

Package: `plumbing/format/reflog`. File: `reflog.go`.

Removed (7 functions stubbed): `Decoder.Next`, `Decode`, `quoteBounded`, `decodeLine`,
`decodeTimestamp`, `normalizeMessage`, `Encode`.

Kept: `Signature`, `Entry`, `Decoder` types, `NewDecoder`, `maxQuotedLen` const + comments,
the format doc comments (`<old> <new> <name> <<email>> <ts> <tz>\t<msg>` — visible).

Tests deleted: `reflog_test.go` (the package's only test).
