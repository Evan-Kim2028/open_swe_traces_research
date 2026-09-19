# Contract (L2) — lockresolver

Encountering a lock: query the owning txn's status on its primary. ttl>0 means the txn is alive — wait until lock expiry (minCommitTS pushes count); ttl==0 with commitTS>0 rolls forward the lock to commitTS; ttl==0 with no commit rolls the lock back unless the lock says the primary is already committed. Async-commit locks carry secondary lists and min-commit-ts: all secondaries must be checked across regions (grouped per region, batch-checked) to decide commit vs rollback, and the resolved write must carry the computed commit ts. Pessimistic locks roll back only non-persisted versions and must not roll back a persisted one. Status results are cached when cacheable. For reads, a lock whose txn is resolved and whose commit ts is below the reader's ts can be ignored; resolved large-txn locks can be accessed without blocking; unresolved-but-expired locks are resolved inline. The resolving-locks record/update/done trio tracks in-flight resolution so concurrent resolves don't deadlock.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestLock` | locks left by dead txns are resolved and reads/writes proceed |
| `TestLockWithTiKV` | same against a real store |
| `TestLockTTL` | expired vs live locks: wait-until-expiry vs immediate resolve |
| `TestLockWaitTimeLimit` | waits respect the caller's time limit |
| `TestPessimisticRollbackWithRead` | pessimistic locks roll back only non-persisted writes |
| `TestPessimisticTxnResolveAsyncCommitLock` | async-commit secondaries are batch-checked and committed at minCommitTS |
