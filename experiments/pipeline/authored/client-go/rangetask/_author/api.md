# Exported API — rangetask

NewRangeTaskRunner(store,concurrency,handler)->*Runner; SetRegionsPerTask; SetStatLogInterval; RunOnRange(ctx,start,end); CompletedRegions/FailedRegions; NewLocateRegionBackoffer(ctx)

## Pre-existing callers

delete-range / unsubscribed-range helpers, examples/gcworker.
