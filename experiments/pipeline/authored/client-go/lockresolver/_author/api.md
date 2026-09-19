# Exported API — lockresolver

NewLockResolver; ResolveLocks(bo, callerStartTS, locks)->(msBeforeExpired,err); ResolveLocksForRead(...)->(msBeforeTxnExpired, canIgnore, canAccess, err); GetTxnStatus; BatchResolveLocks; TxnStatus{IsCommitted,IsRolledBack,CommitTS,TTL,Action,StatusCacheable}; NewLock; ExtractLockFromKeyErr; RecordResolvingLocks/UpdateResolvingLocks/ResolveLocksDone

## Pre-existing callers

KVTxn commit/pessimistic paths, snapshot reads (ResolveLocksForRead), GC/safepoint tools.
