# Exported API — srvresp

Package `plumbing/protocol/packp` (module `example.internal/gitkit/v6`).

Types kept: `ServerResponse{ACKs []ACK}`, `ACK{Hash plumbing.Hash; Status ACKStatus}`,
`ACKStatus byte` with constants `ACKContinue`, `ACKCommon`, `ACKReady` (iota+1 — zero means
"no status", the non-multi_ack case).

Methods: `(*ServerResponse).Decode(r io.Reader) error`, `(*ServerResponse).Encode(w io.Writer) error`,
`(ACKStatus).String() string`.

Unexported helpers (stubbed): `ServerResponse.decodeLine`, `ServerResponse.decodeACKLine`,
`encodeServerResponse`.

Callers: fetch negotiation — the upload-pack server answers `have` lines with this response.
In-tree tests removed: whole `packp` suite.
