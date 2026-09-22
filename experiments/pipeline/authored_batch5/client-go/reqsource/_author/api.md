# Exported API — reqsource

Package `util` (module `example.internal/kvstore/v2`).

- `RequestSource` struct + `SetRequestSourceInternal`, `SetRequestSourceType`,
  `SetExplicitRequestSourceType`, `GetRequestSource() string`
- `BuildRequestSource(internal bool, source, explicitSource string) string`
- `IsRequestSourceInternal(*RequestSource) bool`, `IsInternalRequest(string) bool`
- `WithInternalSourceType(ctx, source)`, `WithInternalSourceAndTaskType(ctx,
  source, task)`, `RequestSourceFromCtx(ctx) string`
- `WithResourceGroupName(ctx, name)`, `ResourceGroupNameFromCtx(ctx) string`
- Kept consts: `Internal*`, `ExplicitType*`, `ExplicitTypeList`,
  `InternalRequest`, `ExternalRequest`, `SourceUnknown`, context keys.

Callers: RPC request building across `internal/client` and `txnkv` sets the
source label for resource tracking; `IsInternalRequest` gates metrics.
In-tree test `request_source_test.go` removed with the closure.
