# Closure — mvccread

Package: `internal/mockstore/mockkv`. File:
`internal/mockstore/mockkv/mvcc.go` (partial — read-path half).

Removed (all bodies stubbed): `mvccLock.lockErr`, `mvccLock.check`,
`mvccEntry.Less`, `mvccEntry.Get`, `regionContains`.

Kept (same file, different closure): the marshal codec family
(`MarshalBinary`/`UnmarshalBinary`, `marshalHelper`, `writeFull`,
`NewMvccKey`, `MvccKey.Raw` — see `mvcccodec`), all types and interfaces.

Tests edited: `mvcc_test.go` deleted — its only test is
`TestRegionContains`, dedicated to this closure.
