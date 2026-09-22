# Details — clusterq

1. `GetRegionByKey`/`getRegionByKeyNoLock` return the region whose
   `[StartKey, EndKey)` contains the key (empty EndKey = unbounded), plus
   its leader peer, buckets, and the peers currently marked down via
   `MarkPeerDown` filtered to that region's peer set. Inferable: partially
   — the down-peer join is documented shape; which peers is data.
2. `GetPrevRegionByKey` finds the region whose `EndKey` equals the
   containing region's `StartKey`; returns nils when the containing
   region's StartKey is empty, and PANICS (nil deref) when no region
   contains the key. Inferable: no — adjacency-by-boundary and the missing
   nil guard are arbitrary.
3. `ScanRegions` sorts regions by StartKey, keeps those intersecting
   `[startKey, endKey)` (empty endKey unbounded; a region whose EndKey ==
   startKey is excluded), applies `limit` after filtering, and fills
   `Leader` with a non-nil empty `&metapb.Peer{}` for leaderless regions.
   Inferable: no.
4. `GetAndCheckStoreByAddr` returns `context.Canceled` when ANY store in
   the cluster has `cancel` set — even stores that do not match the
   address. Inferable: no.
5. `GetStore`, `GetAllStores`, `GetRegion`, `GetStoreByAddr`, and the
   region lookups return proto.Cloned metas: mutating a returned object
   must not corrupt cluster state. Inferable: partially — comment says
   "return a deep copy" on SplitRaw only.
6. `MarkTombstone` panics on a missing store (unchecked deref) while
   `StopStore`/`StartStore`/`CancelStore` silently no-op on missing ids.
   Inferable: no.
7. `AllocID` yields 1, 2, 3, ... and `AllocIDs(n)` continues the same
   counter. Inferable: yes — unique-id doc.
8. `MarkPeerDown` is cluster-global; a down peer surfaces in
   `GetRegionByKey`/`ScanRegions`/`GetRegionByID` only when it belongs to
   the returned region. `RemoveDownPeer` clears it. Inferable: partially.
