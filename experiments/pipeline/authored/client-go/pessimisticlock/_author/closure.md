# Closure — pessimisticlock

Package: txnkv/transaction.

Removed functions (bodies stubbed): actionPessimisticLock.handleSingleBatch, actionPessimisticRollback.handleSingleBatch, handleRegionError, handleKeyErrorForResolve, handlePessimisticLockResponseNormalMode, handlePessimisticLockResponseForceLockMode, pessimisticLockMutations, pessimisticRollbackMutations.

Exported entry point(s): KVTxn.LockKeys / committer pessimisticLockMutations / pessimisticRollbackMutations.
