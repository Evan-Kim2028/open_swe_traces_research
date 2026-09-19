# Exported API — onregionerror

RegionRequestSender.SendReqCtx(bo, req, region, timeout, opts...) (*tikvrpc.Response, *RPCContext, error); NewRegionRequestSender(cache, client); SendReq wraps SendReqCtx.

## Pre-existing callers

kvclient snapshot/txn paths, txnkv committer batch sends, rawkv client, internal client retry wrapper.
