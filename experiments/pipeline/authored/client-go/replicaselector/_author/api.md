# Exported API — replicaselector

internal to SendReqCtx; observable via which store receives each attempt and via GetTiKVRPCContext options (replica-read type, follower seed, store-selector options).

## Pre-existing callers

RegionRequestSender.SendReqCtx / sendReqToRegion.
