# Exported API — grpcxray

```
func NewUnaryServer(service, daemon string) (grpc.UnaryServerInterceptor, error)
func NewStreamServer(service, daemon string) (grpc.StreamServerInterceptor, error)
func UnaryClient(host string) grpc.UnaryClientInterceptor
func StreamClient(host string) grpc.StreamClientInterceptor
```

Server interceptors dial a UDP collector, skip tracing when the incoming context has no trace/span IDs, otherwise open a segment named for the service, record the request, honor a parent span ID, submit an in-progress copy, stash the segment in context, invoke the handler, and record error or response. Client interceptors open a remote subsegment of the context segment when one exists, push the new span into context, and on streams wrap the client stream so the first real error (not io.EOF) records a failure and closes the subsegment once.

## Pre-existing callers

Generated gRPC servers/clients that enable AWS X-Ray; package tests.
