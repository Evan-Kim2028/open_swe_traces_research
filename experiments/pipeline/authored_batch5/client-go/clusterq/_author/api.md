# Exported API — clusterq

Package `internal/mockstore/mockkv` (module `example.internal/kvstore/v2`).

- `NewCluster(MVCCStore) *Cluster` — simulated cluster (stores, regions,
  down-peers, delay events).
- Identity: `(*Cluster) AllocID() uint64`, `AllocIDs(int) []uint64`.
- Store reads: `GetStore(uint64) *metapb.Store`,
  `GetAllStores() []*metapb.Store`, `GetStoreByAddr(string) *metapb.Store`,
  `GetAndCheckStoreByAddr(string) ([]*metapb.Store, error)`.
- Store state: `StopStore`, `StartStore`, `CancelStore`, `UnCancelStore`,
  `MarkTombstone` (all `func(uint64)`).
- Region reads: `GetRegion(uint64) (*metapb.Region, uint64)`,
  `GetRegionByKey([]byte) (*metapb.Region, *metapb.Peer, *metapb.Buckets,
  []*metapb.Peer)`, `GetPrevRegionByKey` (same signature),
  `GetRegionByID` (same), `GetAllRegions() []*Region`,
  `ScanRegions(start, end []byte, limit int, ...pd.GetRegionOption)
  []*pd.Region`.
- Down peers: `MarkPeerDown(uint64)`, `RemoveDownPeer(uint64)`.

Callers: `mocktikv` RPC handlers and `cluster.Cluster` test harnesses.
Kept-real collaborators: `AddStore`, `Bootstrap`, `SplitRaw`,
`PutRegion`, `AddPeer`, `RemovePeer`, `ChangeLeader`, `SplitKeys`,
`Merge`, `regionContains` — tests set up state through them.
