# Closure — revfile

Package: `plumbing/format/revfile`. Files: `decoder.go`, `encoder.go`.

Removed (15 functions stubbed): `Decode`, `readMagicNumber`, `readVersion`,
`readHashFunction`, `readEntries`, `readPackChecksum`, `readRevChecksum`, `Encode`,
`encoder.buildReverseIndex`, `writeHeader`, `writeVersion`, `writeHashFunction`,
`writeEntries`, `writePackChecksum`, `writeRevChecksum`.

Kept: `decoder`/`encoder` state structs + `stateFn` types, `revHeader` magic, version and
hash-function constants, all error vars, doc comments (incl. "Decode closes out when done").

Tests deleted: `decoder_test.go`, `encoder_test.go`, `decoder_fuzz_test.go` (all 3).
