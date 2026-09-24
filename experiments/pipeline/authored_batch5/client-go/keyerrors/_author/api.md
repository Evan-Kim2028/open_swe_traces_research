# Exported API — keyerrors

Package `error` (module `example.internal/kvstore/v2`).

- `ExtractKeyErr(*kvrpcpb.KeyError) error` — maps a KV KeyError proto to a
  typed Go error.
- Predicates: `IsErrKeyExist`, `IsErrWriteConflict`, `IsErrorUndetermined`.
- `NewErrWriteConflictWithArgs(startTs, conflictTs, conflictCommitTs,
  key, reason) *ErrWriteConflict`; `NewErrPDServerTimeout(msg) error`.
- `Error()` on `ErrDeadlock`, `PDError`, `ErrKeyExist`, `ErrWriteConflict`,
  `ErrWriteConflictInLatch`, `ErrRetryable`, `ErrTxnTooLarge`,
  `ErrEntryTooLarge`, `ErrPDServerTimeout`, `ErrGCTooEarly`,
  `ErrTokenLimit`, `ErrAssertionFailed`, `ErrLockOnlyIfExists*` and
  `ErrQueryInterruptedWithSignal`.

Deliberately NOT in the closure: `IsErrNotFound` (owned by an existing
unit), `Log` (side-effect helper), sentinel vars.

Callers: every RPC response path (`internal/client`, `txnkv`) routes
`keyErr` through `ExtractKeyErr`; predicates classify failures for retry.
