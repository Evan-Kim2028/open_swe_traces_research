# Contract (L2) — doactionbatches

Mutations are grouped by the region containing their first key, preserving mutation order; a group may be pre-split when it spans a just-created region boundary. Single-group actions run inline; multi-group actions run concurrently — one goroutine per group bounded by commit concurrency — and the first error cancels the rest. Inside a group, mutations are chopped into batches: each batch is capped both in key count and in total byte size (sum of key+value sizes), the primary key's batch is sent first and alone, and batch boundaries respect per-region limits. Send failures trigger a region re-lookup and regrouping retry of the affected keys; a group whose region splits mid-action is re-batched. Error classification decides retry vs abort (e.g. undetermined results abort immediately).

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `2pc_test.go committer suite` | multi-region commits/prewrites produce correct per-region requests; oversized txns split into size/count-bounded batches; region errors regroup correctly; concurrent groups preserve primary-first ordering |
| `prewrite_test.go` | prewrite batching boundaries |
| `async_commit/1pc tests (pre-excision)` | group dispatch still feeds the decision layer |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./txnkv/transaction/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestPrewriteBatchSizeProperty`, `TestMultiRegionDispatchProperty`, `TestPrimaryFirstPrewriteProperty`, `TestDoActionBatchesContractExamples`, `TestDoActionBatchesUnmentionedRandom`: TestPrewriteBatchSizeProperty

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
