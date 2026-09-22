# Exported API — execfmt

Package `util` (module `example.internal/kvstore/v2`).

- `FormatDuration(time.Duration) string` — precision-pruned duration.
- `(*ScanDetail) Merge(*ScanDetail)`, `MergeFromScanDetailV2(*kvrpcpb.
  ScanDetailV2)`.
- `(*WriteDetail) Merge(*WriteDetail)`, `MergeFromWriteDetailPb(
  *kvrpcpb.WriteDetail)`.
- `(*TimeDetail) MergeFromTimeDetail(*kvrpcpb.TimeDetailV2,
  *kvrpcpb.TimeDetail)` — V2-preferred merge.
- `(*ResolveLockDetail) Merge(*ResolveLockDetail)`.
- `RUDetails`: `NewRUDetails()`, `NewRUDetailsWith(rru, wru, wait)`,
  `Clone`, `Merge`, `RRU`, `WRU`, `RUWaitDuration`,
  `Update(*rmpb.Consumption, time.Duration)`.

Callers: `ExecDetails`/`StmtExecSummary` aggregation and the slow-query
log; left real: `String()` formatters, `CommitDetails`, `LockKeysDetails`
(kept — same file, different closure family).
