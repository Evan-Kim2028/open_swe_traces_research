# Contract (L2) — httpxray

Server middleware that cannot reach the collector returns an error. When the request context has no trace id or span id, the next handler runs unchanged. Otherwise a segment named for the service is opened, the incoming request is recorded (empty namespace), a parent span id is copied when present, an in-progress copy is sent, the segment is stored on the context, and the handler writes through a wrapping writer that records status and byte count; 429 sets throttle, other 4xx set fault, 5xx set error. The wrapper hijacks if the inner writer can. Client wrappers (a Doer and a RoundTripper) are no-ops without a context segment; with one they open a remote subsegment named for the request host, record the request, update the span in context, and on return record a response or an error and close. Write errors are recorded as segment errors.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestNew` | a bad collector address fails middleware construction |
| `TestMiddleware` | requests with trace metadata open a segment; without metadata they do not |
| `TestWrapDoer` | a Doer without a context segment is a pass-through; with one it opens a remote subsegment |
| `TestTransport` | a RoundTripper records a remote call when a segment is present |
| `TestTransportNoSegmentInContext` | a RoundTripper without a segment does not open one |
| `TestTransportExample` | wrapping the default transport still round-trips |
| `TestRecordRequest` | the recorded request URL, method, and client address match the incoming request |
| `TestRecordResponse` | 429/4xx/5xx set throttle, fault, or error on the segment |
| `TestRace` | concurrent writes to the wrapping writer do not race |
