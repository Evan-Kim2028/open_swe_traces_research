# Contract (L2) — regionstoresorted

Cached regions live in a btree ordered by start key. Search finds the region containing a key (end-key searches treat the range end as exclusive — a key equal to a region's end belongs to the next region). Insert replaces a same-start entry, drops regions the new one covers, and invalidates stale versions: a cached region is only evicted when the incoming one has a newer-or-equal epoch (id + confVer + version); older epochs keep the cache. Invalidate marks a region with a reason (stale/suspect) — invalidated entries are skipped by search but remain until GC or replace. UpdateLeader changes the cached leader peer index only when the region epoch still matches. Cache hits must return a region whose [start,end) actually contains the key; miss triggers a meta load and insert.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRegionCache` | locate/insert/invalidate/leader-update behavior across splits |
| `TestRegionCacheWithDelay` | delayed invalidation semantics |
| `TestBackgroundRunner` | cache GC/eviction loop |
| `TestRegionRequestToSingleStore` | cached context correctness through sends |
