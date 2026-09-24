# Exported API — protoparse

Package `server` (module `example.internal/msgkit/v2`) — inbound wire-protocol
parser for client, route, gateway, and leaf-node connections.

Surface: `(c *client) parse(buf []byte) error` — byte-oriented state machine
driven by `c.state`/`c.pa`/`c.argBuf`/`c.msgBuf` on the embedded `parseState`,
re-enterable across `readLoop` buffer splits. Recognised ops: `CONNECT`,
`PUB`, `HPUB`, `SUB`, `UNSUB`, `PING`, `PONG`, `+OK`, `-ERR`, `INFO`,
`A+`/`A-` (account subs), `RS+`/`RS-`/`RMSG`, `LS+`/`LS-`/`LMSG`, `HMSG`
(route/leaf variants), all case-insensitive. Helpers:
`protoSnippet(start, max int, buf []byte) string` (error-message excerpt),
`(c *client) overMaxControlLineLimit(arg []byte, mcl int32) error`,
`(c *client) clonePubArg(lmsg bool) error`,
`(ps *parseState) getHeader() http.Header`.

Callers: `client.go` `readLoop` (same package), `mqtt.go`. In-tree tests
removed: `split_test.go`, `parser_fuzz_test.go`, all `Test*` in
`parser_test.go` (23 tests); `dummyClient`/`dummyRouteClient` helpers kept
for `jetstream_test.go`/`mqtt_test.go`.
