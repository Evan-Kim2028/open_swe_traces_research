# Exported API — updreq

Package `plumbing/protocol/packp` (module `example.internal/gitkit/v6`).

Types kept: `UpdateRequests{Capabilities capability.List; Commands []*Command; Shallows []plumbing.Hash}`,
`Command{Name plumbing.ReferenceName; Old, New plumbing.Hash}`, `Action string` with constants
`Create`, `Update`, `Delete`, `Invalid`. Error vars kept: `ErrEmptyCommands`, `ErrMalformedCommand`,
`ErrEmpty`.

Methods: `(*Command).Action() Action`; `(*UpdateRequests).Decode(r io.Reader) error`;
`(*UpdateRequests).Encode(w io.Writer) error`.

Unexported helpers (same package, stubbed): `validateUpdateRequests`, `Command.validate`,
`parseCommand`, `parseHash`, `UpdateRequests.encodeShallow`, `UpdateRequests.encodeCommands`,
`formatCommand`.

Callers: `send-pack`/`receive-pack` client and server paths in `plumbing/transport` and `packp`
(`updreq` is the reference-update request used on push). In-tree tests removed: the whole
`packp` suite (`updreq_test.go`, `updreq_decode_test.go`, `updreq_encode_test.go`,
`conformance_test.go`, `fetch_test.go`, `smart_test.go`, and the rest of the package's `*_test.go`).
