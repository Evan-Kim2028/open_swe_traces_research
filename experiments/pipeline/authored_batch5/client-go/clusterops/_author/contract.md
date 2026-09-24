# Contract (L2) — clusterops

Cluster structural mutations keep `RegionEpoch` bookkeeping consistent:
peer changes bump `ConfVer`; key-range changes bump `Version`. Removing a
peer that is the region's leader leaves the region leaderless. `Bootstrap`
and `newRegion` panic when the store and peer id lists differ in length,
and `split` panics when the existing peer count does not match the new
peer-id list. `SplitRaw` carves the new region as `[key, oldEnd)` and
truncates the source to `[oldStart, key)`, with the new region inheriting
the source's stores positionally under the caller's peer ids.
`UpdateStoreAddr` sets both `Address` and `PeerAddress` to the same value;
`UpdateStorePeerAddr` preserves `Address` and changes only `PeerAddress`.
`mergeLabels` unions labels by key with the new labels winning conflicts,
and a label-less store takes the argument wholesale. `SplitKeys` on a
range inside one existing region evacuates the range — the covering
region splits into two surrounding regions — and scanned pairs are
distributed across region groups by quotient with the first `remainder`
groups receiving one extra. `Merge` extends the first region's `EndKey`
over the second's and deletes the second; `SplitRegionBuckets` stores
MVCC-encoded boundary keys; a scheduled `(startTS, regionID)` delay is
consumed once.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | peer mutations bump `ConfVer`, key-range mutations bump `Version` |
| `TestDetail02` | removing the leader peer leaves the region leaderless |
| `TestDetail03` | `Bootstrap`/`newRegion`/`split` panic on store/peer id list length mismatch |
| `TestDetail04` | `SplitRaw` produces `[key, oldEnd)` plus `[oldStart, key)`, stores inherited positionally |
| `TestDetail05` | `UpdateStoreAddr` sets both address fields; `UpdateStorePeerAddr` changes only `PeerAddress` |
| `TestDetail06` | `mergeLabels` unions by key, new labels win, empty store takes the slice |
| `TestDetail07` | `SplitKeys` evacuates an interior range into two surrounding regions; pairs distribute by quotient+remainder |
| `TestDetail08` | `Merge` joins region2 into region1 and removes region2; bucket splits are MVCC-encoded; delays are consumed once |
