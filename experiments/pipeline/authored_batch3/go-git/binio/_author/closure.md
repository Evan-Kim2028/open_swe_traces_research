# Closure — binio

Package: `utils/binary`. Files: `read.go`, `write.go`.

Removed (13 functions stubbed): `Read`, `ReadUntil`, `ReadUntilFromBufioReader`,
`ReadVariableWidthInt`, `ReadUint64`, `ReadUint32`, `ReadUint16`, `IsBinary`, `Write`,
`WriteVariableWidthInt`, `WriteUint64`, `WriteUint32`, `WriteUint16`.

Kept: `maskContinue`/`maskLength`/`lengthBits` consts, `sniffLen` const, `sniffPool`,
`ErrIntegerOverflow`, the long Git-VLQ doc comment (offset scheme + reference C code —
the `doc` inferable).

Tests deleted: `read_test.go`, `write_test.go`.
