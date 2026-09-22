# Bug report

`util/execdetails.go`'s duration formatter and the detail-merge family
are stubbed: `FormatDuration`, `ScanDetail`/`WriteDetail`/`TimeDetail`/
`ResolveLockDetail` merges, and all `RUDetails` methods panic.

Expected (probes): `FormatDuration` yields `999ns`→`999ns`,
`1µs`→`1µs`, `1.001µs`→`1µs`, `9.412345ms`→`9.41ms`,
`10.412345ms`→`10.4ms`, `5.999s`→`6s`, `100.45µs`→`100.5µs`,
`1.5s`→`1.5s`, `1m1s`→`1m1s`; `MergeFromTimeDetail` with V2
{Wait100,Proc200,Susp50,KvRead5,Total300 (ns)} yields
{100ns,200ns,50ns,5ns,300ns} even when a V1 detail is also passed; with
V1 only {Wait9ms,Proc7ms,KvRead3ms} yields those three durations;
`MergeFromScanDetailV2`{TotalVersions:4,ProcessedVersions:3,
ProcessedVersionsSize:99} yields TotalKeys=4/ProcessedKeys=3/
ProcessedKeysSize=99 and a nil arg is a no-op; `ScanDetail.Merge` adds
fields; `NewRUDetailsWith(1.5,2.5,3s)` + `Update({RRU:.5,WRU:.5},1s)` →
RRU=2, WRU=3, wait=4s; `Clone` copies; `Merge` adds.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
