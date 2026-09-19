# Contract (L2) — rangetask

A range task splits [startKey,endKey) into per-region subranges and processes them with a fixed pool of workers fed by a channel. Region boundaries are re-resolved as the task walks: each batch of regions (up to regionsPerTask) is queued as one unit and the walk resumes from the last processed end key — a region split/merge mid-task must not skip or double-process keys. Workers call the handler per region range with a fresh backoffer derived from the task context; a handler error cancels the whole task and propagates the first error; context cancellation stops promptly. Completed/Failed region counters and stat logging track progress.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `range_task_test.go: TestRangeTask*/table cases` | ranges covering multiple regions are processed exactly once each; cancel/error paths stop cleanly |
