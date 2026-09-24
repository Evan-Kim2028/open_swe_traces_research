# Details — execfmt

1. `FormatDuration` returns `d.String()` unmodified when `d <= 1µs`
   (inclusive boundary); otherwise it truncates to the largest unit
   (s/ms/µs) and keeps 2 decimals below 10 units, 1 decimal at or above —
   `9.412345ms`→`9.41ms`, `10.412345ms`→`10.4ms`, `100.45µs`→`100.5µs`,
   and `5.999s`→`6s` (rounding can carry into the next unit's display).
   Inferable: doc — the prune rule is spelled out in the comment.
2. `MergeFromTimeDetail` prefers `TimeDetailV2` (nanosecond fields incl.
   `ProcessSuspendWallTimeNs` → `SuspendTime`) and ignores the V1 arg when
   V2 is non-nil; the V1 path uses `*Ms` fields for wait/process/kvread
   but `TotalRpcWallTimeNs` (nanoseconds!) for total, and never sets
   SuspendTime. Inferable: no — mixed units are an accident of schema
   evolution.
3. `ScanDetail.MergeFromScanDetailV2` is nil-safe and RENAMES fields:
   `TotalVersions`→`TotalKeys`, `ProcessedVersions`→`ProcessedKeys`,
   `ProcessedVersionsSize`→`ProcessedKeysSize`; it also accumulates all
   six RocksDB counters plus the two nanosecond durations.
   Inferable: partially.
4. `ScanDetail.Merge` / `WriteDetail.Merge` / `ResolveLockDetail.Merge`
   accumulate every field atomically (concurrent-safe). Inferable:
   partially — field list is mechanical, atomicity is the contract.
5. `RUDetails` wraps `uatomic` cells: `Update` adds `consumption.RRU`,
   `.WRU`, and `waitDuration`, and is a no-op when the receiver OR the
   consumption is nil; `Clone`/`Merge` snapshot the loads. Inferable:
   partially — the nil-receiver guard is unusual.
6. `WriteDetail.MergeFromWriteDetailPb` maps all 13 `*Nanos` fields to
   `time.Duration`s; nil pb is a no-op. Inferable: yes — mechanical.
