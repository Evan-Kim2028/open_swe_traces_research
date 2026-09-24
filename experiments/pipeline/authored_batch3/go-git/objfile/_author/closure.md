# Closure — objfile

Package: `plumbing/format/objfile`. Files: `reader.go`, `writer.go`.

Removed (11 functions stubbed): `Reader.Header`, `Reader.readUntil`,
`Reader.prepareForRead`, `Reader.Read`, `Reader.Hash`, `Writer.WriteHeader`,
`Writer.writeHeader`, `Writer.prepareForWrite`, `Writer.Write`, `Writer.Hash`,
`Writer.Close`.

Kept: `Reader`/`Writer` structs (fields visible), `NewReader`, `NewWriter`,
`Reader.Close`, all error vars, `maxHeaderLen` const + its canonical-git doc comment,
`prepareForRead`/`prepareForWrite` bodies removed but their 3-line wiring role documented.

Tests deleted: `common_test.go`, `reader_test.go`, `writer_test.go`, `reader_fuzz_test.go`,
`writer_fuzz_test.go` (all 5 in the package).
