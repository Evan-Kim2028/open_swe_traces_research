# Closure — reqframe

Package: `plumbing/protocol/packp`. Files: `command.go`, `gitproto.go`.

Removed (6 functions stubbed): `CommandRequest.Encode`, `CommandRequest.Decode`,
`GitProtoRequest.validate`, `validateGitProtoField`, `GitProtoRequest.Encode`,
`GitProtoRequest.Decode`.

Kept: types `CommandRequest{Command; Capabilities capability.List; Args CommandArgs}`,
`GitProtoRequest{RequestCommand; Pathname; Host; ExtraParams []string}`, the `CommandArgs`
interface, `ErrInvalidGitProtoRequest`, the wire-format doc comments on both types (visible),
shared constants.

All 23 `*_test.go` files in the package deleted (`command_test.go`, `gitproto_test.go`).
