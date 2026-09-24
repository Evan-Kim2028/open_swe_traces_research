# Exported API — reflog

Package `plumbing/format/reflog` (module `example.internal/gitkit/v6`) — reflog entry codec
(`.git/logs/*` line format).

`Entry{OldHash, NewHash plumbing.Hash; Committer Signature{Name, Email, When time.Time};
Message string}`. `NewDecoder(r io.Reader) *Decoder`, `Decoder.Next() (*Entry, error)`,
`Decode(r io.Reader) ([]*Entry, error)`, `Encode(w io.Writer, e *Entry) error`.

Callers: filesystem storage's reflog reading/writing. In-tree tests removed: 1.
