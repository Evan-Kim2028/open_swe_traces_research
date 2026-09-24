# Exported API — lsrefs

Package `plumbing/protocol/packp` (module `example.internal/gitkit/v6`).

Types kept: `LsRefsArgs{Peel, Symrefs, Unborn bool; RefPrefixes []string}` — the command-specific
arguments of the protocol-v2 `ls-refs` command; `LsRefsOutput{References []*plumbing.Reference}` —
the server's answer.

Methods: `(*LsRefsArgs).Encode(w io.Writer) error`, `(*LsRefsArgs).Decode(r io.Reader) error`,
`(*LsRefsOutput).Encode(w io.Writer) error`, `(*LsRefsOutput).Decode(r io.Reader) error`.

Unexported helpers (stubbed): `validateRefPrefix`, `parseLsRefsLine`, `parseFullHash`.

Callers: protocol-v2 transport (`transport/http` v2 handshake, `command` request args implement
`CommandArgs`). In-tree tests removed: whole `packp` suite.
