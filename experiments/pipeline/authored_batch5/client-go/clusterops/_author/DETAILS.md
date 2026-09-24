# Details — clusterops

1. `AddPeer`/`RemovePeer` bump `RegionEpoch.ConfVer`; `updateKeyRange`,
   `split`, and `merge` bump `RegionEpoch.Version` — epoch bookkeeping is
   part of every structural mutation. Inferable: partially — the epoch
   fields exist in the meta but which operation bumps which is
   convention.
2. `RemovePeer` zeroes the region's leader when the removed peer WAS the
   leader. Inferable: partially — the doc comment says the region "will
   have no leader".
3. `Bootstrap` and `newRegion` panic when `len(storeIDs) !=
   len(peerIDs)`; `Region.split` panics when `len(Peers) !=
   len(peerIDs)`. Inferable: no — a defensive panic is a choice.
4. `SplitRaw` carves the NEW region as `[key, oldEnd)` and truncates the
   source to `[oldStart, key)`; the new region inherits the source's
   storeIDs positionally with the caller's peerIDs. Inferable: partially.
5. `UpdateStoreAddr` replaces the store meta setting BOTH `Address` and
   `PeerAddress` to the same value; `UpdateStorePeerAddr` preserves
   `Address` and changes only `PeerAddress`. Inferable: no.
6. `mergeLabels` unions labels by key with the NEW labels winning on
   conflicts; a store with zero existing labels takes the argument slice
   wholesale. Inferable: partially.
7. `SplitKeys` on a range inside one existing region EVACUATES the range:
   the covering region splits into `["",start)` and `[end,"")` — so
   splitting empty data into count=3 yields 2 regions, not 3.
   `getEntriesGroupByRegions` distributes scanned pairs by
   quotient+remainder (first `remainder` groups get one extra).
   Inferable: no.
8. `Merge` sets region1's EndKey to region2's and deletes region2.
   `SplitRegionBuckets` stores MVCC-encoded keys. `ScheduleDelay`/
   `handleDelay` consume a `(startTS,regionID)`-keyed delay once.
   Inferable: partially.
