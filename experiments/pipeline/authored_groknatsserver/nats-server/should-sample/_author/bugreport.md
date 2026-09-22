Service-latency tracing no longer honours inbound trace headers.

A request with `Uber-Trace-Id: 0:0:0:1` (or `...:01`, or `...:5`) used to be sampled and the Uber headers forwarded. The same header ending in `:0` or `:00` used to be skipped. `X-B3-Sampled: 1` used to force a sample; `X-B3-Sampled: 0` used to force a skip.

After the last change every request is treated as unsampled, so exported services never emit latency advisories even when the caller asked for a trace.
