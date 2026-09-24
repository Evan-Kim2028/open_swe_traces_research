# Exported API — inforefs

Package `plumbing/protocol/packp` (module `example.internal/gitkit/v6`).

Type kept: `InfoRefs{References []*plumbing.Reference}` — the ref list advertised by an HTTP
dumb server's `info/refs` file (the `update-server-info` output). Error var kept:
`ErrInvalidInfoRefs`.

Methods: `(*InfoRefs).Decode(r io.Reader) error`, `(*InfoRefs).Encode(w io.Writer) error`.

Unexported helper (stubbed): `usableInfoRefsName`.

Callers: `plumbing/transport/http` dumb-protocol fetch (`dumb.go` reads this body first), and
`transport/serverinfo.go`-adjacent paths. In-tree tests removed: whole `packp` suite.
