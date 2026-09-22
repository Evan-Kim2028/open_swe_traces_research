# API — leafmsg

Module: `example.internal/msgkit/v2`, package `server`, file `server/leafnode.go`.

Excised symbols:

- `func (c *client) processLeafMsgArgs(arg []byte) error` — parses the
  argument bytes of an inbound `LMSG` protocol line into `c.pa`
  (subject, reply, queues, szb, size).
- `func (c *client) processLeafHeaderMsgArgs(arg []byte) error` — same
  for `LHMSG`, additionally filling `hdb`/`hdr` (header length).
- `func keyFromSub(sub *subscription) string` — builds the routed-sub
  map key `"subject"` / `"subject queue"`.
- `func keyFromSubWithOrigin(sub *subscription) string` — builds the
  collision-safe key used in `acc.rm`/`acc.lws`, prefixed with a kind
  byte (`R` plain routed, `N` routed leaf sub without origin, `L`
  routed leaf sub with origin) and suffixed with the origin cluster
  name when present.

Callers: the leaf-node read loop invokes the arg parsers after the
protocol line scanner splits `LMSG`/`LHMSG`; route and leaf
subscription bookkeeping (`acc.rm`, `acc.lws`, `smap`) uses the key
builders everywhere a routed or leaf-borne subscription is recorded.

Retained dependencies: `parseSize` (util.go — banked separately),
`maxPayloadViolation`, `ErrMaxPayload`, `subscription` fields.
