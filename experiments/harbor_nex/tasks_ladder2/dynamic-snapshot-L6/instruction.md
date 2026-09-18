# Missing behavior

The in-memory buffer can take a snapshot of its current keys. Keys inserted
after that snapshot is taken must not appear in it: a live read sees the new
key, the snapshot reports not-exist. Keys that already existed at the
checkpoint remain readable. A snapshot lookup of a key that was never written
is not-exist. A snapshot iterator must not yield keys inserted after the
checkpoint.

Lookups after thousands of inserts must stay fast under parallel readers.
Scanning every stored pair on each lookup is too slow: 5000 inserts then
parallel snapshot lookups must stay under the ns/op ceiling recorded from
the reference implementation (gold time × 3). A getter that always reports
not-exist is wrong: expected the stored value, actual not-exist.

Coverage the hidden checks enforce:

- Live vs snapshot after staging; missing key is not-exist; iterator yields the older value.
- Keys other than the staging example still round-trip.
- Parallel snapshot lookup throughput vs gold × 3.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/unionstore/
go test -count=1 -timeout 15m -bench=^BenchmarkSnapshotGet$ -benchtime=5000x ./internal/unionstore/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestSnapshotStagingVisible`, `TestSnapshotUnmentionedKeys`: Staging vs snapshot visibility plus gold×3 parallel lookup gate.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
