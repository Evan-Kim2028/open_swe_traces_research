# Closure — deadlock

Package: `internal/mockstore/deadlock`. File:
`internal/mockstore/deadlock/deadlock.go` (151 lines).

Removed (all bodies stubbed): `NewDetector`, `ErrDeadlock.Error`,
`Detector.Detect`, `Detector.doDetect`, `Detector.register`,
`Detector.CleanUp`, `Detector.CleanUpWaitFor`, `Detector.Expire`.

Kept: `Detector`, `txnList`, `txnKeyHashPair`, `ErrDeadlock` type
definitions.

Tests edited: `internal/mockstore/deadlock/deadlock_test.go` deleted — it is
solely dedicated to this closure. No other package's tests touch the
detector directly (mockkv calls it through `Detect`).
