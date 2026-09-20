# Exported API — binio

Package `utils/binary` (module `example.internal/gitkit/v6`) — BigEndian scalar I/O, Git
offset-VLQ, delimiter reads, binary sniffing.

`Read(r, ...any)`, `Write(w, ...any)`, `ReadUint16/32/64`, `WriteUint16/32/64`,
`ReadUntil(r, delim) []byte`, `ReadUntilFromBufioReader(*bufio.Reader, delim)`,
`ReadVariableWidthInt(r) int64`, `WriteVariableWidthInt(w, int64)`, `IsBinary(r)
(bool, error)`, `ErrIntegerOverflow`.

Callers: idx/pack/revfile/commit-graph codecs, diff binary detection. In-tree tests
removed: 2.
