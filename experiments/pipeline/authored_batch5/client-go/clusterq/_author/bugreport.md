# Bug report

`internal/mockstore/mockkv`'s `Cluster` read surface panics: id
allocation, store lookups/state toggles, region lookups, `ScanRegions`,
and down-peer bookkeeping are stubbed, so no mocktikv test can query the
simulated cluster.

Expected (probes after `NewCluster(nil)`, `AddStore(1,"s1")`,
`AddStore(2,"s2")`, `Bootstrap(10,{1,2},{101,102},101)`,
`SplitRaw(10,20,[]byte("m"),{103,104},103)`, `MarkPeerDown(102)`):
`GetRegionByKey("a")` → region 10, leader peer 101, one down peer;
`GetRegionByKey("z")` → region 20, leader 103, zero down peers;
`GetPrevRegionByKey("n")` → region 10; `GetPrevRegionByKey("a")` → nils;
`ScanRegions("a",nil,0)` → [10,20] sorted; `ScanRegions("a","p",1)` →
[10]; `ScanRegions("x",nil,0)` → [20]; `GetRegion(10)` → (meta,101);
`GetRegion(99)` → (nil,0); `GetStoreByAddr("s2")` → store 2;
`GetAndCheckStoreByAddr("s2")` → 1 store, no error; after
`CancelStore(2)` → `context.Canceled`; `MarkTombstone(1)` then
`GetStore(1).State` → Tombstone; `AllocID()` → 1,2; `AllocIDs(3)` →
[3,4,5].

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
