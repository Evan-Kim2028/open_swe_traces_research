# Exported API — mvccread

Package `internal/mockstore/mockkv` (package name `mocktikv`; module
`example.internal/kvstore/v2`).

Read-path predicates (package-internal):

- `(*mvccLock) check(ts uint64, key []byte, resolvedLocks []uint64)
  (uint64, error)` — lock visibility under SI.
- `(*mvccLock) lockErr(key []byte) error` — builds `ErrLocked` for the raw
  key.
- `(*mvccEntry) Get(ts, isoLevel, resolvedLocks) ([]byte, error)` —
  committed-version selection.
- `(*mvccEntry) Less(than btree.Item) bool` — btree ordering on encoded
  keys.
- `regionContains(startKey, endKey, key []byte) bool` — `[start,end)` with
  empty end unbounded.

Callers: `mvcc_leveldb.go` Get/Scan/BatchGet paths call `entry.Get` and
`lock.check`; `regionContains` guards scan bounds. In-tree `mvcc_test.go`
(`TestRegionContains`) removed with the closure.
