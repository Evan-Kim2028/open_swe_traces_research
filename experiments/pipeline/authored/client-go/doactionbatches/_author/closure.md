# Closure — doactionbatches

Package: txnkv/transaction (2pc.go).

Removed functions (bodies stubbed): doActionOnMutations, groupMutations, doActionOnGroupMutations, doActionOnBatches, preSplitRegion.

Exported entry point(s): committer doActionOnMutations (drives prewrite/commit/cleanup/pessimistic ops).
