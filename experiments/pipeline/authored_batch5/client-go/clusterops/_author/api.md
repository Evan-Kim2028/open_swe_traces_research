# Exported API — clusterops

Package `internal/mockstore/mockkv` (module `example.internal/kvstore/v2`).

- Store mutation: `(*Cluster) AddStore(id, addr, labels...)`,
  `RemoveStore(id)`, `UpdateStoreAddr(id, addr, labels...)`,
  `UpdateStorePeerAddr(id, peerAddr, labels...)`,
  `UpdateStoreLabels(id, []*metapb.StoreLabel)`.
- Region lifecycle: `Bootstrap(regionID, storeIDs, peerIDs, leaderID)`,
  `PutRegion(regionID, confVer, ver, storeIDs, peerIDs, leaderID)`.
- Peer mutation: `AddPeer(regionID, storeID, peerID)`,
  `RemovePeer(regionID, peerID)`, `ChangeLeader(regionID, peerID)`,
  `GiveUpLeader(regionID)`.
- Topology: `Split(regionID, newID, key, peerIDs, leaderID)` (encodes
  key), `SplitRaw(...rawKey...) *metapb.Region`, `Merge(id1, id2)`,
  `SplitKeys(start, end []byte, count int)`,
  `SplitRegionBuckets(regionID, keys [][]byte, bucketVer uint64)`.
- Delay events: `ScheduleDelay(startTS, regionID uint64, d
  time.Duration)`.

Kept-real collaborators: the whole query surface (`GetRegion*`,
`ScanRegions`, `GetStore*`, down-peer marks) so tests can observe
mutations. Sibling unit `clusterq` owns the read half — disjoint.
