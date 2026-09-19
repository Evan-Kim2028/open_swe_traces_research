# Exported API — doactionbatches

twoPhaseCommitter internal; observable via KVTxn.Commit/Prewrite/Rollback semantics and CommitDetails batch counts

## Pre-existing callers

prewrite, commit, cleanup, pessimistic lock/rollback drivers.
