# Closure — sigcodec

Package: `plumbing/object`. Files: `object.go`, `blob.go`.

Removed (21 functions stubbed): `GetObject`, `DecodeObject`,
`Signature.Decode`, `.Encode`, `.decodeTimeAndTimeZone`,
`.encodeTimeAndTimeZone`, `.String`; `NewObjectIter`, `ObjectIter.Next`,
`.ForEach`, `.toObject`; `GetBlob`, `DecodeBlob`, `Blob.ID`, `.Type`,
`.Decode`, `.Encode`, `.Reader`; `NewBlobIter`, `BlobIter.Next`,
`.ForEach`.

Kept: `Object`/`Signature`/`ObjectIter`/`Blob`/`BlobIter` types,
`timeZoneLength` const, `ObjectIter` doc comments; `signature.go` PGP
block stripping is intact (sigblock owns those symbols); commit/tag/tree
decoders intact (commitobj/tagparse/treeobj own theirs).

Tests deleted: `blob_test.go`, `object_test.go`, `decode_fuzz_test.go`,
`commit_test.go`, `commit_stats_test.go`, `tag_test.go` (6 — all assert
on decoded signatures or blobs; signature_test stays, it covers the
kept signature.go surface).
