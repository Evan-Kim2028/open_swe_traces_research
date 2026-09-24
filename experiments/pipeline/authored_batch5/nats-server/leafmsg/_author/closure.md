# Closure — leafmsg

File: `server/leafnode.go`

Stubbed (4 symbols):

- `client.processLeafMsgArgs` — `LMSG` arg parser
- `client.processLeafHeaderMsgArgs` — `LHMSG` arg parser
- `keyFromSub` — routed-sub key `"subject[ queue]"`
- `keyFromSubWithOrigin` — kind-prefixed key `"R|N|L subject[ queue][ origin]"`

Retained as scaffolding (visible in excised tree): `parseSize`
(util.go), `maxPayloadViolation`, the leaf read loop that dispatches
`LMSG`/`LHMSG`, `writeLeafSub`, all leafnode validation and connect
handling, and the `subscription`/`client` types.

Test coverage snipped (restored for gold/cheat runs):
`TestParseLeafMsgBadSize`, `TestParseLeafHeaderMsgBadSize`
(parser_test.go), `TestRouteSubUnsubRaceLosesRemoteInterest`,
`TestClusterQueueGroupWeightTrackingLeak` (routes_test.go),
`TestLeafNodeRoutedSubKeyDifferentBetweenLeafSubAndRoutedSub`
(leafnode_test.go).

Boundary: protocol arg decode + key encoding only. No read-loop, no
payload copy, no sublist insertion — those call `pa`/`key` consumers
outside the closure.
