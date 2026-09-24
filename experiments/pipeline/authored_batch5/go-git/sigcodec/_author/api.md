# Exported API — sigcodec

Package `plumbing/object` — the `Signature` codec (`Decode`/`Encode`/
`decodeTimeAndTimeZone`/`encodeTimeAndTimeZone`/`String`), the generic
`GetObject`/`DecodeObject` dispatch, `ObjectIter`, and the whole `Blob`
type (`GetBlob`/`DecodeBlob`/`ID`/`Type`/`Decode`/`Encode`/`Reader`/
`BlobIter`).

Kept visible: `Signature`/`Blob`/`ObjectIter` types, `timeZoneLength`
const, doc comments; `signature.go` block-stripping (sigblock unit) is a
separate file with disjoint symbols.

Callers: commit/tag decode paths, `Repository.Blob`. In-tree tests
removed: 6 (blob/object/decode-fuzz/commit/commit-stats/tag — all reach
signature or blob decoding).
