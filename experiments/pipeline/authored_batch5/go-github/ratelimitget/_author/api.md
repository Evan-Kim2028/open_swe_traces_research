# Exported API — ratelimitget

`RateLimitService.Get(ctx)` fetches `/rate_limit`. The endpoint itself
isn't rate-limited, so the call marks the context
`BypassRateLimitCheck`; it also writes each returned category's limits
back into the client's shared `rateLimits` map so subsequent calls in
those categories short-circuit correctly.
