# Exported API — codecbytes

Package `util/codec` (module `example.internal/kvstore/v2`) — memcomparable
byte-string encoding (MyRocks-style grouped padding).

`func EncodeBytes(b []byte, data []byte) []byte` — append the memcomparable
encoding of `data` to `b`.
`func DecodeBytes(b []byte, buf []byte) ([]byte, []byte, error)` — decode one
value, returning the leftover input and the decoded bytes (may reuse `buf`).

Callers: `internal/apicodec/mem_codec.go`, `internal/mockstore/mockkv`
(mvcc key layout). In-tree tests removed: none.
