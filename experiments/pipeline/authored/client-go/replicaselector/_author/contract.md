# Contract (L2) — replicaselector

A selector walks a region's replicas in a defined order: leader first (when leader-read), otherwise replicas ordered by a load-aware score, optionally bounded attempts; followers only for follower/stale reads or as fallback after the leader is unreachable; a proxy store may forward to the real target when the target is unreachable, subject to proxy eligibility. Each candidate is checked for liveness (estimated slow score / unreachable marks / deadline) before building the context; dead stores are skipped, not retried forever. On send failure the selector marks the store unreachable and advances; on not-leader it updates the leader index (hint peer wins) and may retry the new leader; on server-busy it decides whether another replica or the proxy is eligible; when all candidates are exhausted or the region is invalid, it invalidates the cached region so the next attempt reloads it. Busy/slow scores feed candidate ordering, and successful sends record access stats.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestReplicaReadStaleReadAccessPathByCase` | stale reads pick exactly the replica/config path each case table prescribes, including fallback and proxy decisions |
| `TestRegionRequestToSingleStore` | candidate order and failover match the prescribed leader/follower/proxy sequence |
| `TestRegionCacheStaleRead` | stale-read replica choice respects data-readiness and liveness |
