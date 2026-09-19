# Exported API — regionstoresorted

NewSortedRegions(degree); ReplaceOrInsert; SearchByKey(key,isEndKey); AscendGreaterOrEqual(start,end,limit); Clear; ValidRegionsInBtree(ts); RegionCache.LocateKey/TryLocateKey/LocateEndKey/LocateRegionByID; InvalidateCachedRegion(WithReason); UpdateLeader

## Pre-existing callers

every LocateKey/GroupKeys path, replica selector target lookup, GC loop, region error handlers.
