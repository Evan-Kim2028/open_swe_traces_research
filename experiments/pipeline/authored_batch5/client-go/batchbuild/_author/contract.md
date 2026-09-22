# Contract (L2) — batchbuild

`newBatchCommandsBuilder(maxBatchSize)` returns a builder that owns a
`PriorityQueue`; `push`/`len` proxy the queue. `buildWithLimit(limit,
collect)` emits queued entries in heap order — the queue is a max-heap
on `priority()`, so the largest priority value is emitted first — and calls
`collect(id, entry)` once per emitted entry with a fresh request id, so every
emitted entry appears in `RequestIds` exactly once. Entries whose priority
reaches `highTaskPriority` do not consume limit and keep the build draining
past an exhausted limit; normal entries stop emitting once `limit` is
reached. Entries with a non-empty `forwardedHost` are collected into the
per-host request map instead of the main request; both sides still get
request ids. Canceled entries are never emitted. A build over no emittable
entries returns a nil request. `reset()` removes canceled entries from the
queue (a post-reset build never emits a canceled entry) and leaves the
builder usable. `cancel(err)` delivers the error to every queued entry —
the entry records the error and its result channel is closed — and empties
the queue.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | entries at or above `highTaskPriority` emit without consuming limit, so a high-priority head is sent even at limit 0 and a build at limit 2 over {high, low, low} emits all three; normal entries stop at the limit |
| `TestDetail02` | entries with a non-empty `forwardedHost` land in the per-host request map rather than the main request, and every emitted entry consumes one unique request id |
| `TestDetail03` | after `reset()` a build never emits a canceled entry and the builder still accepts and emits new pushes |
| `TestDetail04` | a canceled queued entry is not emitted by the build |
| `TestDetail05` | `cancel(err)` records the error on every queued entry, closes its result channel, and empties the queue |
| `TestDetail06` | the highest-priority queued entry is emitted first (heap order, not insertion order) and an empty build returns a nil request |
