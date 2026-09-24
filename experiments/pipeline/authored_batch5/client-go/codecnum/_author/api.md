# Exported API — codecnum

Package `util/codec` (module `example.internal/kvstore/v2`) — memcomparable
integer encodings used for on-disk keys.

Fixed-width (8-byte big-endian): `EncodeInt`, `EncodeIntDesc`, `EncodeUint`,
`EncodeUintDesc` append to a slice; `DecodeInt`, `DecodeIntDesc`,
`DecodeUint`, `DecodeUintDesc` return `(rest, value, err)`. Sign mapping:
`EncodeIntToCmpUint`, `DecodeCmpUintToInt`.

Variable-width: `EncodeVarint`/`DecodeVarint` (zig-zag), `EncodeUvarint`/
`DecodeUvarint`, and the mem-comparable `EncodeComparableVarint`,
`EncodeComparableUvarint`, `DecodeComparableVarint`, `DecodeComparableUvarint`.

Callers: `internal/apicodec/mem_codec.go`, `internal/mockstore/mockkv` (mvcc
key layout). In-tree tests removed: none (package has no `_test.go`).
