# Closure — tagparse

Package: `plumbing/object`. Files: `tag.go`, `tag_scanner.go`.

Removed (15 functions stubbed): `Tag.Decode`, `Tag.Encode`,
`Tag.EncodeWithoutSignature`, `Tag.matchesSource`, `Tag.encode`, `isZeroSignature`,
`tagScanner.readLine`, `tagScanner.pushBack`, `scanTagObject`, `scanTagType`,
`scanTagName`, `scanTagTagger`, `scanTagHeaders`, `scanTagPgp256Cont`, `scanTagMessage`.

Kept: `Tag`/`tagScanner`/`TagIter` types, `GetTag`, `DecodeTag`, `ID`, `Type`, `reset`,
`Commit`/`Tree`/`Blob`/`Object`/`String`/`Verify` accessors, `TagIter`, all doc comments
(which mirror upstream's parse order — they are the `doc` inferables), shared helpers in
`commit_scanner.go`/`signature.go`/`commit.go` (kept source: `splitHeader`, `isBlankLine`,
`parseObjectIDHex`, `parseSignedBytes`, `stripObjectSignatures`, `signatureEqual`).

Tests deleted: `tag_test.go` only.
