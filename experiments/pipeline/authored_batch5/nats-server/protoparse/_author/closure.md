# Closure — protoparse

Package: `server`. File: `parser.go`.

Removed (5 funcs stubbed): `client.parse`, `protoSnippet`,
`client.overMaxControlLineLimit`, `client.clonePubArg`,
`parseState.getHeader`.

Kept: `parserState`/`parseState`/`pubArg` types, all `OP_*`/`*_ARG` state
constants, every callee the parser dispatches to (`processPub`,
`processHeaderPub`, `processConnect`, `parseSub`, `processUnsub`,
`processRemoteSub`, `processRemoteUnsub`, `processGatewayRSub`,
`processGatewayRUnsub`, `processLeafSub`, `processLeafUnsub`,
`processRoutedMsgArgs`, `processRoutedOriginClusterMsgArgs`,
`processRoutedHeaderMsgArgs`, `processLeafMsgArgs`,
`processLeafHeaderMsgArgs`, `processAccountSub`, `processAccountUnsub`,
`processInboundMsg`, `processPing`, `processPong`, `processErr`,
`processInfo`, `mqttParse`, `selectMappedSubject`, `initMsgTrace`,
`traceInOp`, `traceMsg`, `sendErr`, `authViolation`,
`checkAuthentication`, `connectionTypeAllowed`, `clearAuthTimer`,
`awaitingAuth`, `isMqtt`, `kindString`, `removeSecretsFromTrace`,
`needsCompression`), and constants (`MAX_CONTROL_LINE_SIZE`,
`LEN_CR_LF`, `PROTO_SNIPPET_SIZE`, `MAX_CONTROL_LINE_SNIPPET_SIZE`,
`MAX_HMSG_ARGS`).
Imports `bufio`, `bytes`, `fmt`, `net/textproto` blanked.

Tests deleted: `server/split_test.go`, `server/parser_fuzz_test.go`.
Tests snipped: all 23 `Test*` funcs in `server/parser_test.go`;
helpers `dummyClient`/`dummyRouteClient` retained (used by
`jetstream_test.go`, `mqtt_test.go`).
