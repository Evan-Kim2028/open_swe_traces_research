# Missing behavior

Concurrent transactions that touch the same key must not both sit in the
critical section. The second caller waits until the first releases. After
release, waiters are woken in order. The same-key overlapping hold must be
race-detector clean under `go test -race`; dropping the per-slot mutex so
the queue is mutated without synchronization is wrong.

Worked case: the first caller takes a hold on key `k` and the second waits;
the observed enter-order is first then second. expected the second waits,
actual both proceed (or a data race).

Coverage the hidden checks enforce:

- Exclusive overlapping hold: second waits, order is first then second.
- Concurrent same-key holds are race-detector clean.

Reproduce with:

```
go test -count=1 -timeout 15m -race ./internal/latch/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestLatchExclusiveOverlap`, `TestLatchConcurrentSameKey`: Exclusive overlapping hold plus concurrent same-key race gate.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
