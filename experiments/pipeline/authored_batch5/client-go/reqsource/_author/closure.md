# Closure — reqsource

Package: `util`. File: `util/request_source.go` (187 lines).

Removed (all bodies stubbed): `RequestSource.SetRequestSourceInternal`,
`.SetRequestSourceType`, `.SetExplicitRequestSourceType`,
`.GetRequestSource`, `WithInternalSourceType`,
`WithInternalSourceAndTaskType`, `BuildRequestSource`,
`IsRequestSourceInternal`, `RequestSourceFromCtx`, `IsInternalRequest`,
`WithResourceGroupName`, `ResourceGroupNameFromCtx`.

Kept: all consts/`ExplicitTypeList`, `RequestSource` struct, context key
types and vars.

Tests edited: `util/request_source_test.go` deleted — both tests
(`TestGetRequestSource`, `TestBuildRequestSource`) are dedicated to this
closure.
