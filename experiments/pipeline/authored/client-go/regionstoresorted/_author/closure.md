# Closure — regionstoresorted

Package: internal/locate (sorted_btree.go + region_cache.go).

Removed functions (bodies stubbed): SortedRegions.ReplaceOrInsert/SearchByKey/AscendGreaterOrEqual/Clear/ValidRegionsInBtree; RegionCache.insertRegionToCache/InvalidateCachedRegionWithReason/searchCachedRegionByKey/UpdateLeader/tryFindRegionByKey.

Exported entry point(s): RegionCache.LocateKey / findRegionByKey + SortedRegions ops.
