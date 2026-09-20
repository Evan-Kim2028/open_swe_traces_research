# Exported API — objfile

Package `plumbing/format/objfile` (module `example.internal/gitkit/v6`) — zlib-compressed
loose-object codec.

`NewReader(r io.Reader, objectFormat format.ObjectFormat) (*Reader, error)`;
`Reader.Header() (plumbing.ObjectType, int64, error)`, `Read`, `Hash() plumbing.Hash`,
`Close`. `NewWriter(w io.Writer, objectFormat format.ObjectFormat) *Writer`;
`Writer.WriteHeader(t plumbing.ObjectType, size int64) error`, `Write`, `Hash()`,
`Close`.

Error vars kept: `ErrClosed`, `ErrHeader`, `ErrHeaderTooLong`, `ErrHeaderNotRead`,
`ErrNegativeSize`, `ErrOverflow`. Const `maxHeaderLen` kept with its doc comment.

Callers: filesystem storage backend reading/writing `.git/objects/xx/…` loose files.
In-tree tests removed: all 5 in the package.
