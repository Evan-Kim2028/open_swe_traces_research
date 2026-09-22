# Contract (L2) — clusterq

`GetRegionByKey` returns the region whose `[StartKey, EndKey)` contains
the key (empty `EndKey` is unbounded) together with its leader peer,
buckets, and the cluster's down peers filtered to that region's peer set.
`GetPrevRegionByKey` finds the region whose `EndKey` equals the containing
region's `StartKey`, returns nils when the containing region has an empty
`StartKey`, and panics when no region contains the key. `ScanRegions`
returns regions sorted by `StartKey`, keeps only those intersecting
`[startKey, endKey)` (a region whose `EndKey` equals `startKey` is
excluded), applies `limit` after filtering, and reports a non-nil empty
leader peer for leaderless regions. `GetAndCheckStoreByAddr` returns
`context.Canceled` when any store in the cluster is cancelled, even one
that does not match the address. Store and region metas are returned as
deep copies — mutating a returned object does not corrupt cluster state.
`MarkTombstone` panics on a missing store while `StopStore`/`StartStore`/
`CancelStore` silently no-op on missing ids. `AllocID` yields 1, 2, 3, ...
and `AllocIDs` continues the same counter. `MarkPeerDown` is
cluster-global but a down peer surfaces in region results only when it
belongs to the returned region; `RemoveDownPeer` clears it.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | `GetRegionByKey` returns the containing region, leader, buckets, and down peers filtered to its peer set |
| `TestDetail02` | `GetPrevRegionByKey` finds the boundary-adjacent region, nils at the head, panics when no region contains the key |
| `TestDetail03` | `ScanRegions` sorts by `StartKey`, filters to the range, excludes `EndKey == startKey`, limits after filtering, emits a non-nil empty leader |
| `TestDetail04` | `GetAndCheckStoreByAddr` returns `context.Canceled` when any store is cancelled, matching or not |
| `TestDetail05` | returned store and region metas are deep copies |
| `TestDetail06` | `MarkTombstone` panics on a missing store; `Stop`/`Start`/`Cancel` no-op on missing ids |
| `TestDetail07` | `AllocID` counts up from 1 and `AllocIDs` continues the counter |
| `TestDetail08` | `MarkPeerDown` is cluster-global but surfaces only for member regions; `RemoveDownPeer` clears it |
