# Contract (L2) — pessimisticlock

Pessimistic locking per region batch: each lock key is sent with forUpdateTS, ttl, and flags (return-value, lock-only-if-exists, no-fair-locking). On success the response may carry per-key values (when return-values requested) and per-key existence/ttl info; locks acquired must be recorded. On a locked-key error the waiter blocks up to wait-timeout respecting the killed flag and deadlocks are reported; on write-conflict the forUpdateTS is bumped and the lock retried; region errors re-split and retry the affected keys only; ttl-manager keeps lock ttl alive during long waits. In lock-only-if-exists (fair locking) mode the response reports which keys existed — missing keys get no lock and reported values come from the response; normal mode reports values per the return-values flag. Rollback sends a best-effort release for acquired locks per region and must not fail the caller on unreachable regions. Dedup: already-locked keys are not re-locked; the primary key is locked first and only once.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestPessimisticPrewriteRequest` | lock then prewrite carries right flags |
| `TestPessimisticLockedKeysDedup` | keys locked once |
| `TestPessimisticTTL` | ttl kept alive during wait |
| `TestPessimisticLockReturnValues` | returned values match stored |
| `TestPessimisticLockIfExists` | only existing keys locked |
| `TestPessimisticLockCheckExistence` | existence map correct |
| `TestPessimisticLockAllowLockWithConflict` | conflict path allowed when flag set |
| `TestPessimisticLockAllowLockWithConflictError` | conflict error when flag unset |
| `TestPessimisticLockPrimary` | primary locked first |
| `TestPessimisticRollbackWithRead` | rollback doesn't delete persisted locks |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./txnkv/transaction/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
