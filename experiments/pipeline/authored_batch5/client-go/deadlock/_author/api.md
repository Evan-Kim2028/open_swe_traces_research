# Exported API — deadlock

Package `internal/mockstore/deadlock` (module `example.internal/kvstore/v2`).

- `NewDetector() *Detector`
- `(*Detector) Detect(sourceTxn, waitForTxn, keyHash uint64) *ErrDeadlock`
- `(*Detector) CleanUp(txn uint64)`
- `(*Detector) CleanUpWaitFor(txn, waitForTxn, keyHash uint64)`
- `(*Detector) Expire(minTS uint64)`
- `(*ErrDeadlock) Error() string`; field `KeyHash uint64`

Callers: `internal/mockstore/mockkv/mvcc_leveldb.go` keeps a
`deadlockDetector` and calls `Detect` when a lock wait begins, `CleanUp` /
`CleanUpWaitFor` when waits resolve. In-tree test `deadlock_test.go`
(`TestDeadlock`) was removed with the closure.
