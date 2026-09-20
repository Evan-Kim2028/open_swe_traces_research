# Closure — negside

Package: `plumbing/protocol/packp`. Files: `shallowupd.go`, `pushopts.go`.

Removed (8 functions stubbed): `ShallowUpdate.Decode`, `decodeShallowLine`,
`decodeUnshallowLine`, `decodeLine`, `ShallowUpdate.Encode`, `PushOptions.Encode`,
`PushOptions.Decode`, `isNotGraphic`.

Kept: `ShallowUpdate{Shallows, Unshallows []plumbing.Hash}`, `PushOptions{Options []string}`,
line-length constants, `ErrInvalidPushOption`, `MaxPayloadSize`, doc comments, `pktline.Scanner`
(kept source — visible that `Err()` returns nil on plain EOF).

All 23 `*_test.go` files in the package deleted (`shallowupd_test.go`, `pushopts_test.go`).
