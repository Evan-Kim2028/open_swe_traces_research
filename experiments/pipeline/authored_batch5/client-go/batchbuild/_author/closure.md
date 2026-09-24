# Closure — batchbuild

Package: `internal/client`. File: `internal/client/client_batch.go`
(898 lines).

Removed (10 bodies stubbed): `batchCommandsEntry.isCanceled`,
`.priority`, `.error`, `batchCommandsBuilder.len`, `.push`,
`.hasHighPriorityTask`, `.buildWithLimit`, `.cancel`, `.reset`,
`newBatchCommandsBuilder`.

Kept real: `PriorityQueue` (separate unit `priorityqueue`),
`batchConn`/`batchCommandsClient` send/fetch loops, `highTaskPriority`
const, all streaming/recv code.

Tests edited: `TestBatchCommandsBuilder` and `TestLimitConcurrency`
surgically removed from `internal/client/client_test.go` (both solely
probe this closure). `TestPrioritySentLimit`,
`TestForwardMetadataByBatchCommands`, `TestBatchClientRecoverAfter-
ServerRestart` retained — they reach the closure through RPCClient.
