# Closure — msgtrace

Package `server`, file `server/msgtrace.go`.

Removed (stubbed): `genHeaderMapIfTraceHeadersPresent`, `getConnName`,
`getCompressionType`, `sample`, `(*client).msgTraceSupport`.

Retained: the `MsgTrace*` types and their `UnmarshalJSON`, `initMsgTrace`
and the whole event assembly/send path, `msgTrace` methods,
`setHopHeader`, `hdrLine`/`crLFAsBytes`/`dashAsBytes` constants,
`traceParentHdr`, `traceDestDisabled`, `MsgTraceProto`, compressionType
enum, all imports except `math/rand/v2` and `strconv` which are blanked in
the excised tree (only this closure used them).

Tests snipped in server/msgtrace_test.go: `TestMsgTraceGenHeaderMap`
(28-case table), `TestMsgTraceConnName`. No test files deleted; the
integration tests (TestMsgTraceBasic, Ingress errors, routes/leaf/gateway
variants) still drive initMsgTrace → the closure and panic under the bare
excision.
