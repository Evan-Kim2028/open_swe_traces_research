# Exported API — pdoracle

NewPdOracle(pdClient, updateInterval)->oracle.Oracle; GetTimestamp(ctx,opt)->(ts,err); GetTimestampAsync->Future{Wait}; IsExpired(lockTS,TTL); UntilExpired(lockTS,TTL)->ms; GetLowResolutionTimestamp(+Async); GetStaleTimestamp(scope,prevSecond); SetLowResolutionTimestampUpdateInterval; Set/GetExternalTimestamp; Close

## Pre-existing callers

KV store txn start (startTS), snapshot ts, lock TTL checks, stale reads.
