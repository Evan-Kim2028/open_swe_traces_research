# Exported API — httpxray

```
func New(service, daemon string) (func(http.Handler) http.Handler, error)
func WrapDoer(doer goahttp.Doer) goahttp.Doer
func WrapTransport(rt http.RoundTripper) http.RoundTripper

func (s *HTTPSegment) RecordRequest(req *http.Request, namespace string)
func (s *HTTPSegment) RecordResponse(resp *http.Response)
func (s *HTTPSegment) WriteHeader(code int)
func (s *HTTPSegment) Write(p []byte) (int, error)
func (s *HTTPSegment) Hijack() (net.Conn, *bufio.ReadWriter, error)
```

New is the server middleware: UDP connect, skip when context has no trace/span, otherwise open a segment, record the incoming request, write through the segment, close on return. WrapDoer / WrapTransport open a remote subsegment for the request host when a segment is in context. HTTPSegment copies request/response fields, classifies 429/4xx/5xx, and implements hijack.

## Pre-existing callers

Generated HTTP servers and clients with X-Ray enabled; package tests.
