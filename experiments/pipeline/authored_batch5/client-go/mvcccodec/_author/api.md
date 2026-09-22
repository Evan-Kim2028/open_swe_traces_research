# Exported API — mvcccodec

Package `internal/mockstore/mockkv` (package name `mocktikv`; module
`example.internal/kvstore/v2`).

Package-internal binary codec for MVCC records:

- `(*mvccLock) MarshalBinary() / UnmarshalBinary(data) error`
- `(mvccValue) MarshalBinary() / (*mvccValue) UnmarshalBinary(data) error`
- `marshalHelper.WriteNumber/WriteSlice/ReadNumber/ReadSlice`, `writeFull`
- `NewMvccKey(key []byte) MvccKey`, `(MvccKey) Raw() []byte`

Callers: `mvcc_leveldb.go` marshals entries into the embedded store and
encodes keys via `NewMvccKey`. In-tree test `marshal_test.go`
(`TestMarshalmvccLock`, `TestMarshalmvccValue`) removed with the closure.
