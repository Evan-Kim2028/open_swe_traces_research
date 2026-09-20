# Exported API — negside

Package `plumbing/protocol/packp` (module `example.internal/gitkit/v6`).

`ShallowUpdate{Shallows []plumbing.Hash; Unshallows []plumbing.Hash}` — shallow/unshallow
boundary updates the server streams during fetch. `Encode(io.Writer) error`,
`Decode(io.Reader) error`.

`PushOptions{Options []string}` — transport push options carried as pkt-line payloads.
`Encode(io.Writer) error`, `Decode(io.Reader) error`. `MaxPayloadSize` bounds a single option
payload. `ErrInvalidPushOption`.

Callers: fetch/push negotiation paths and transports. In-tree tests removed: whole `packp`
suite.
