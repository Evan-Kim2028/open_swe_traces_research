# Exported API — batchbuild

Package `internal/client` (module `example.internal/kvstore/v2`).

- `newBatchCommandsBuilder(maxBatchSize uint) *batchCommandsBuilder`.
- `(*batchCommandsBuilder) push(*batchCommandsEntry)`, `len() int`,
  `hasHighPriorityTask() bool`, `buildWithLimit(limit int64,
  collect func(id uint64, e *batchCommandsEntry))
  (*tikvpb.BatchCommandsRequest, map[string]*tikvpb.BatchCommandsRequest)`,
  `cancel(error)`, `reset()`.
- `(*batchCommandsEntry) isCanceled() bool`, `priority() uint64`,
  `error(err error)`.

Callers: `batchConn.fetchAllPendingRequests`/`sendBatchRequest` drive
the builder each flush; `client_test.go` exercises it directly.
Collaborator kept real: `PriorityQueue` (owned by unit `priorityqueue`),
`highTaskPriority` const, `batchConn`.
