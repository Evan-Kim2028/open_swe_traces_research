# Closure — clusterops

Package: `internal/mockstore/mockkv`. File:
`internal/mockstore/mockkv/cluster.go` (776 lines).

Removed (36 bodies stubbed): `Cluster.AddStore`, `.RemoveStore`,
`.UpdateStoreAddr`, `.UpdateStorePeerAddr`, `.UpdateStoreLabels`,
`.Bootstrap`, `.PutRegion`, `.AddPeer`, `.RemovePeer`, `.ChangeLeader`,
`.GiveUpLeader`, `.Split`, `.SplitRaw`, `.SplitRegionBuckets`, `.Merge`,
`.SplitKeys`, `.splitRange`, `.getEntriesGroupByRegions`,
`.createNewRegions`, `.evacuateOldRegionRanges`, `.getRegionsCoverRange`,
`.firstStoreID`, `.ScheduleDelay`, `.handleDelay`, `newPeerMeta`,
`newRegion`, `Region.addPeer`, `Region.removePeer`, `Region.changeLeader`,
`Region.split`, `Region.merge`, `Region.updateKeyRange`,
`Region.incConfVer`, `Region.incVersion`, `newStore`,
`Store.mergeLabels`.

Kept real: `NewCluster`, all getters/`ScanRegions`, `StopStore`/
`CancelStore`/`MarkTombstone`, down-peer marks, `Region.leaderPeer` —
the observation surface tests assert through. Sibling unit `clusterq`
owns those symbols.

Tests: mocktikv tests reach the closure through RPC-driven cluster
configuration; retained.
