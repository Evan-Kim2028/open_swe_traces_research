# Bug report

`internal/mockstore/mockkv`'s `Cluster` mutation surface panics:
`AddStore`, `Bootstrap`, `PutRegion`, peer/leader changes, `SplitRaw`,
`Merge`, `SplitRegionBuckets`, label/address updates are all stubbed, so
no mocktikv test can configure the simulated cluster.

Expected (probes on `NewCluster(nil)`): after `AddStore(1,"s1",zone=z1)`
and `AddStore(2,"s2")`, `Bootstrap(10,{1,2},{101,102},101)` yields a
region with 2 peers and leader 101; `AddPeer(10,2,105)` grows it to 3
peers; `ChangeLeader(10,105)` + `GiveUpLeader(10)` move the leader to 0;
`RemovePeer(10,101)` leaves 2 peers; `SplitRaw(10,20,[]byte("m"),
{103,104},103)` leaves region 10 = `["","m")` and region 20 = `["m","")`
with 2 peers; `Merge(10,20)` restores `["","")`; `PutRegion(30,7,9,{1},
{301},301)` exposes ConfVer=7/Version=9; `UpdateStoreLabels(1,{zone:z2,
dc:d1})` merges to `{zone:z2,dc:d1}`; `UpdateStoreAddr(2,"s2-new")`
sets both Address and PeerAddress to `s2-new`, then
`UpdateStorePeerAddr(2,"p2-new")` changes only PeerAddress;
`SplitRegionBuckets(10,{b,g},5)` attaches a Buckets{Version:5} with 2
keys.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
