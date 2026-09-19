# Contract (L2) — onregionerror

Every request to a region goes through a retry loop: pick an RPC context, send, then classify the outcome. Region errors each have a prescribed reaction: NotLeader switches the target peer (using the hint when provided) and retries without backoff; EpochNotMatch may refresh split/merged region info and retry; ServerIsBusy/estimated-busy backs off and may fall back to another replica or a proxy store; StaleCommand/fast-retry errors retry in place; store-not-match and store-tombstone drop the cached store; DiskFull/DataIsNotReady/ReadIndexNotReady wait and re-resolve; unrecognized errors retry a bounded number of times then propagate. A send-level failure (network/RPC error) is distinct from a region error: it marks the store unreachable, re-resolves the address or fails over to the next replica, schedules a region reload, and consumes one retry budget. Whether a region must be reloaded after a response is a separate decision driven by flags on the RPC context.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRegionRequestToSingleStore` | a send to one store exercises the retry loop for each injected region error: each error kind either retries on a different peer, retries in place, or propagates, exactly per contract |
| `TestRegionRequestToThreeStores` | with three stores, send failures fail over across replicas and region errors select the correct next target |
| `TestRegionCacheStaleRead` | stale-read sends pick a replica with valid data and the error path preserves the resolved context |
