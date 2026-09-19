# Exported API — pessimisticlock

KVTxn.LockKeys(ctx,lockCtx,keys...) -> error via committer; pessimistic lock ctx carries forUpdateTS, wait timeout, killed flag, return-values mode, lock-if-exists mode; rollback releases locks

## Pre-existing callers

KVTxn.LockKeys, pessimistic txn write path, fair-locking (lock-only-if-exists) callers.
