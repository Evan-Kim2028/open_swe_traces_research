# Why hard — regionstoresorted

ordered-index invariants under concurrent insert/invalidate: exclusive-end search semantics, epoch-gated eviction (newer wins — get it backwards and stale regions poison the cache), invalidated-but-present tombstones, and leader index updates gated on epoch.
