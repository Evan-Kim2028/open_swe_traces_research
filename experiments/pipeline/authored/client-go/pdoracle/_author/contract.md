# Contract (L2) — pdoracle

A background goroutine refreshes the last-allocated timestamp from the meta service at the update interval, per txn scope. GetTimestamp fast-paths: if the caller's arrival time minus the last fetch is below the update interval it returns lastTS without a meta round-trip, else it fetches a fresh TSO and updates lastTS. Returned timestamps must be monotonic per scope. Futures block until the fetch completes. IsExpired compares the physical part of lockTS+TTL against lastTS's physical part (no fetch). UntilExpired returns remaining ms (negative when already expired). Low-resolution timestamps come from a separate periodically-updated value with its own interval (settable). GetStaleTimestamp returns a ts prevSecond seconds before now, never newer than lastTS, and must not fabricate a future ts when the meta service is unreachable.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestPDOracle_UntilExpired` | until-expired reports remaining ms vs lock ts |
| `TestPdOracle_GetStaleTimestamp` | stale ts is prevSecond back and never ahead of lastTS |
| `TestPdOracle_SetLowResolutionTimestampUpdateInterval` | low-res interval updates apply |
| `TestNonFutureStaleTSO` | an unreachable meta service yields no future ts |
