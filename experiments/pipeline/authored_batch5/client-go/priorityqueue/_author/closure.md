# Closure — priorityqueue

Package: `internal/client`. File: `internal/client/priority_queue.go`
(146 lines).

Removed (all bodies stubbed): `prioritySlice.Len/Less/Swap/Push/Pop`,
`NewPriorityQueue`, `PriorityQueue.Len`, `.Push`, `.pop`, `.Take`,
`.highestPriority`, `.all`, `.clean`, `.reset`.

Kept: `Item` interface, `prioritySlice`/`PriorityQueue` type definitions,
`container/heap` import (blanked in the excised tree).

Tests edited: `internal/client/priority_queue_test.go` deleted — it is
solely dedicated to this closure (`FakeItem` + `TestPriority`). Other
client tests exercise the queue only transitively through batch commands.
