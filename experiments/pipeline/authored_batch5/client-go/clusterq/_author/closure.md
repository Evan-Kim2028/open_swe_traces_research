# Closure — clusterq

Package: `internal/mockstore/mockkv`. File:
`internal/mockstore/mockkv/cluster.go` (776 lines).

Removed (24 bodies stubbed): `NewCluster`, `Cluster.AllocID`,
`Cluster.AllocIDs`, `Cluster.allocID`, `Cluster.GetAllRegions`,
`Cluster.GetStore`, `Cluster.GetAllStores`, `Cluster.StopStore`,
`Cluster.StartStore`, `Cluster.CancelStore`, `Cluster.UnCancelStore`,
`Cluster.GetStoreByAddr`, `Cluster.GetAndCheckStoreByAddr`,
`Cluster.MarkTombstone`, `Cluster.MarkPeerDown`,
`Cluster.RemoveDownPeer`, `Cluster.GetRegion`, `Cluster.GetRegionByKey`,
`Cluster.getRegionByKeyNoLock`, `Cluster.GetPrevRegionByKey`,
`Cluster.getDownPeers`, `Cluster.GetRegionByID`, `Cluster.ScanRegions`,
`Region.leaderPeer`.

Kept real (setup surface): `AddStore`, `RemoveStore`, `Bootstrap`,
`PutRegion`, `Split`, `SplitRaw`, `SplitKeys`, `Merge`, peer mutations,
`newRegion`/`newStore`, epoch helpers, `ScheduleDelay`, `splitRange`
family, `regionContains` (lives in mvcc.go — untouched).

Sibling unit `clusterops` owns the mutation half of the same file —
disjoint symbol sets.

Tests: `cluster_test.go`/`mocktikv` tests exercise the closure
transitively through RPC handlers; retained.
