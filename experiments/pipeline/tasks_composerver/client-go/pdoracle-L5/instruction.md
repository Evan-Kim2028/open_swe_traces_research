# Contract (L2) — pdoracle

A background goroutine refreshes the last-allocated timestamp from the meta service at the update interval, per txn scope. GetTimestamp fast-paths: if the caller's arrival time minus the last fetch is below the update interval it returns lastTS without a meta round-trip, else it fetches a fresh TSO and updates lastTS. Returned timestamps must be monotonic per scope. Futures block until the fetch completes. IsExpired compares the physical part of lockTS+TTL against lastTS's physical part (no fetch). UntilExpired returns remaining ms (negative when already expired). Low-resolution timestamps come from a separate periodically-updated value with its own interval (settable). GetStaleTimestamp returns a ts prevSecond seconds before now, never newer than lastTS, and must not fabricate a future ts when the meta service is unreachable.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestPDOracle_UntilExpired` | until-expired reports remaining ms vs lock ts |
| `TestPdOracle_GetStaleTimestamp` | stale ts is prevSecond back and never ahead of lastTS |
| `TestPdOracle_SetLowResolutionTimestampUpdateInterval` | low-res interval updates apply |
| `TestNonFutureStaleTSO` | an unreachable meta service yields no future ts |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./oracle/...`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestPdOracleUntilExpiredProperty`, `TestPdOracleIsExpiredProperty`, `TestPdOracleGetStaleTimestampProperty`, `TestPdOracleNonFutureStaleProperty`, `TestPdOracleTimestampMonotonicProperty`, `TestPdOracleAsyncFutureProperty`: TestPdOracleUntilExpiredProperty

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
