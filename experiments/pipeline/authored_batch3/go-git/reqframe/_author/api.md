# Exported API — reqframe

Package `plumbing/protocol/packp` (module `example.internal/gitkit/v6`).

Types kept: `CommandRequest{Command string; Capabilities capability.List; Args CommandArgs}` —
the protocol-v2 command request (`command`, capability-list, delim, args, flush);
`GitProtoRequest{RequestCommand, Pathname, Host string; ExtraParams []string}` — the git://
service request; `CommandArgs` interface (`Encoder` + `Decoder`); error var
`ErrInvalidGitProtoRequest`.

Methods: `(*CommandRequest).Encode/Decode(io.Writer/io.Reader) error`,
`(*GitProtoRequest).Encode/Decode(io.Writer/io.Reader) error`.

Unexported helpers (stubbed): `GitProtoRequest.validate`, `validateGitProtoField` (its doc
comment stays in the tree).

Callers: transport clients — v2 command channel (`lsrefs`, `fetch` args implement `CommandArgs`),
git transport handshake, http smart-protocol request line. In-tree tests removed: whole `packp`
suite.
