# Exported API — hdrsurgery

Package `server` (module `example.internal/msgkit/v2`) — NATS header-block
surgery and reply-subject classification used on the inbound message path.

Surface (client.go):
- `getHeader(key string, hdr []byte) []byte` — copied value for key.
- `sliceHeader(key string, hdr []byte) []byte` — zero-alloc borrowed slice
  of the value (cap limited to the value).
- `getHeaderKeyIndex(key string, hdr []byte) int` — offset of a key that
  starts a header line, or -1.
- `removeHeaderStatusIfPresent(hdr []byte) []byte` — strip the
  `NATS/1.0...\r\n` status line.
- `removeHeaderIfPresent(hdr []byte, key string) []byte` — remove all
  exact-key lines; nil when only the empty header line remains.
- `removeHeaderIfPrefixPresent(hdr []byte, prefix string) []byte` — remove
  all lines whose key starts with prefix.
- `isServiceReply(reply []byte) bool` — `_R_.` prefix.
- `isJSAckSubject(subject []byte) bool` — `$JS.ACK.` prefix.
- `jsAckDeliverIdx(reply []byte) int` — offset of the deliver-`@` suffix
  in an encoded JS ack, or -1.
- `replyHasJSAckSuffix(reply []byte) bool`.
- `isReservedReply(reply []byte) bool` — service reply, JS ack, or gateway
  routed reply.
- `splitSubjectQueue(sq string) ([]byte, []byte, error)` — "subj [queue]"
  field split + validation.
- `splitArg(arg []byte) [][]byte` — whitespace tokenization of protocol
  args (space, tab, CR, LF).

Callers: `processInboundClientMsg` paths, `setHeader`, stream/consumer
helpers, `parseSub`, subject-mapping and gateway import paths. Constants
`replyPrefix="_R_."` (accounts.go), `jsAckPre`, `emptyHdrLine`, `hdrLine`,
`LEN_CR_LF` stay visible.
