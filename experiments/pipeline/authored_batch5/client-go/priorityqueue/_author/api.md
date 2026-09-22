# Exported API — priorityqueue

Package `internal/client` (module `example.internal/kvstore/v2`).

- `NewPriorityQueue() *PriorityQueue`
- `(*PriorityQueue) Len() int`
- `(*PriorityQueue) Push(item Item)`
- `(*PriorityQueue) Take(n int) []Item`
- `Item` interface: `priority() uint64`, `isCanceled() bool` — unexported
  methods, so implementations must live in-package (`batchConnEntry` in
  `client_batch.go`).

Package-internal helpers: `pop`, `highestPriority`, `all`, `clean`, `reset`.

Callers: `client_batch.go` (`batchCommands` keeps `entries *PriorityQueue`,
uses `highestPriority` for the high-priority check, `Take` to drain).
In-tree test `priority_queue_test.go` was removed with the closure.
