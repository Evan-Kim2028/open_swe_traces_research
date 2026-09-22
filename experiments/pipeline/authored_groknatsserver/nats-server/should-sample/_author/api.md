# API left after excision

```go
func shouldSample(l *serviceLatency, c *client) (bool, http.Header)
```

Decides whether a service-latency measurement should be recorded for this request, and which tracing headers to propagate. `newB3Header`, `newUberHeader`, `newTraceCtxHeader`, and the `trc*` header-name constants remain.
