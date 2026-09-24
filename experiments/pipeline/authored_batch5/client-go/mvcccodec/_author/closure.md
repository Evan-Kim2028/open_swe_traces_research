# Closure — mvcccodec

Package: `internal/mockstore/mockkv`. File:
`internal/mockstore/mockkv/mvcc.go` (partial — serialization half).

Removed (all bodies stubbed): `mvccLock.MarshalBinary`,
`mvccLock.UnmarshalBinary`, `mvccValue.MarshalBinary`,
`mvccValue.UnmarshalBinary`, `marshalHelper.WriteSlice`,
`marshalHelper.WriteNumber`, `marshalHelper.ReadSlice`,
`marshalHelper.ReadNumber`, `writeFull`, `NewMvccKey`, `MvccKey.Raw`.

Kept (same file, different closure): `mvccLock.check`, `mvccLock.lockErr`,
`mvccEntry.Get`, `mvccEntry.Less`, `regionContains` — the read-path
predicate family (see `mvccread`). All types, consts and interfaces stay.

Tests edited: `marshal_test.go` deleted — solely dedicated to the marshal
closure.
