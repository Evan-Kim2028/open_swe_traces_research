# Exported API — grpctrace

```
func UnaryServerTrace(opts ...middleware.TraceOption) grpc.UnaryServerInterceptor
func StreamServerTrace(opts ...middleware.TraceOption) grpc.StreamServerInterceptor
func UnaryClientTrace() grpc.UnaryClientInterceptor
func StreamClientTrace() grpc.StreamClientInterceptor
func TraceIDFunc(f middleware.IDFunc) middleware.TraceOption
func SpanIDFunc(f middleware.IDFunc) middleware.TraceOption
func SamplingPercent(p int) middleware.TraceOption
func MaxSamplingRate(r int) middleware.TraceOption
func SampleSize(s int) middleware.TraceOption
func DiscardFromTrace(discard *regexp.Regexp) middleware.TraceOption
```

Server interceptors insert a trace id only when incoming metadata has none, the method is not discarded, and the sampler accepts; they always mint a new span id when tracing. Client interceptors copy trace-id and parent-span-id (the current span) onto outgoing metadata when a trace id is on the context.

## Pre-existing callers

Generated gRPC servers/clients; package tests; xray middleware (depends on these context keys).
