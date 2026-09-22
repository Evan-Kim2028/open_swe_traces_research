# Exported API — msgtrace

Package `server` (module `example.internal/msgkit/v2`) — message-trace
header processing: lifting a NATS header block into a case-preserving map
when tracing headers are present, W3C `traceparent` detection with
sampling-bit semantics, plus small helpers for remote-name and
compression negotiation.

Surface (msgtrace.go):
- `genHeaderMapIfTraceHeadersPresent(hdr []byte) (map[string][]string,
  bool)` — nil unless a `MsgTraceDest` header or a valid sampled
  `traceparent` exists; second return = "lifted by external header only".
- `getConnName(c *client) string` — remote name by kind
  (ROUTER/GATEWAY/LEAF) falling back to `c.opts.Name`.
- `getCompressionType(cts string) compressionType` — accept-encoding
  mapping.
- `sample(sampling int) bool` — percentage sampler.
- `(c *client) msgTraceSupport() bool` — remote protocol capability.

Callers: `initMsgTrace`, `sendMsgTraceIngressErrEvent`, ingress message
processing, remote reply/forwarding paths. Constants `hdrLine`,
`MsgTraceDest`, `traceParentHdr`, `traceDestDisabled`, `MsgTraceProto`,
the `compressionType` enum stay visible.
