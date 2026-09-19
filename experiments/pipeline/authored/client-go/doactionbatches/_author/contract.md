# Contract (L2) — doactionbatches

Mutations are grouped by the region containing their first key, preserving mutation order; a group may be pre-split when it spans a just-created region boundary. Single-group actions run inline; multi-group actions run concurrently — one goroutine per group bounded by commit concurrency — and the first error cancels the rest. Inside a group, mutations are chopped into batches: each batch is capped both in key count and in total byte size (sum of key+value sizes), the primary key's batch is sent first and alone, and batch boundaries respect per-region limits. Send failures trigger a region re-lookup and regrouping retry of the affected keys; a group whose region splits mid-action is re-batched. Error classification decides retry vs abort (e.g. undetermined results abort immediately).

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `2pc_test.go committer suite` | multi-region commits/prewrites produce correct per-region requests; oversized txns split into size/count-bounded batches; region errors regroup correctly; concurrent groups preserve primary-first ordering |
| `prewrite_test.go` | prewrite batching boundaries |
| `async_commit/1pc tests (pre-excision)` | group dispatch still feeds the decision layer |
