# Closure — execfmt

Package: `util`. File: `util/execdetails.go` (764 lines).

Removed (16 bodies stubbed): `FormatDuration`, `getUnit`,
`ScanDetail.Merge`, `ScanDetail.MergeFromScanDetailV2`,
`WriteDetail.MergeFromWriteDetailPb`, `WriteDetail.Merge`,
`TimeDetail.MergeFromTimeDetail`, `ResolveLockDetail.Merge`,
`NewRUDetails`, `NewRUDetailsWith`, `RUDetails.Clone`,
`RUDetails.Merge`, `RUDetails.RRU`, `RUDetails.WRU`,
`RUDetails.RUWaitDuration`, `RUDetails.Update`.

Kept: all `String()` formatters, `CommitDetails`/`LockKeysDetails`
merge families (mutex+slowest-request bookkeeping — a separate
closure), context keys, `NewTiKVExecDetails`/`TiKVExecDetails.String`.

Tests: none dedicated to these funcs; exercised transitively by
snapshot/txn detail plumbing.
