# Contract (L2) — grpcxray

A unary or streaming server interceptor that cannot reach the collector at the given host:port returns an error and no interceptor. When the incoming context has no trace id or span id, the interceptor is a no-op and does not open a segment. Otherwise it opens a segment for the service name, records the RPC method as the request, copies a parent span id when present, emits an in-progress copy, stores the segment on the context, and after the handler returns records a response on success or an error on failure, then closes. A unary or streaming client interceptor with no segment in context is a no-op. With a segment, it opens a remote subsegment named for the host, updates the outgoing span, records the request, and on unary calls records error or response and closes. On streams, a wrapper records the first non-EOF failure (EOF is a clean close) and closes the subsegment only once, even if later messages also fail.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestNewUnaryServer` | a bad collector address fails interceptor construction |
| `TestNewStreamServer` | a bad collector address fails interceptor construction |
| `TestUnaryServerMiddleware` | unary server opens a segment, records the RPC, and records errors |
| `TestStreamServerMiddleware` | streaming server wraps the stream and records success or failure |
| `TestUnaryClient` | unary client is a no-op without a context segment and records a remote subsegment with one |
| `TestStreamClient` | stream client treats EOF as success and records other errors once |
