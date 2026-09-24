# Closure — srvresp

Package: `plumbing/protocol/packp`. File: `srvresp.go`.

Removed (6 functions stubbed): `ACKStatus.String`, `ServerResponse.Decode`,
`ServerResponse.decodeLine`, `ServerResponse.decodeACKLine`, `ServerResponse.Encode`,
`encodeServerResponse`.

Kept: types `ServerResponse{ACKs []ACK}`, `ACK{Hash, Status}`, `ACKStatus byte` and its constants
`ACKContinue`, `ACKCommon`, `ACKReady`, the `ackLineLen` constant, shared wire constants
(`ack`, `nak`, `eol`) and `ErrNilWriter`.

All 23 `*_test.go` files in the package deleted (`srvresp_test.go`, `conformance_test.go`,
`fetch_test.go` exercise this codec).
