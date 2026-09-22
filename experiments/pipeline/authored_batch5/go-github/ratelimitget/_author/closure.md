# Closure — ratelimitget

Package: github (root). File: github/rate_limit.go.
Removed bodies: RateLimitService.Get — stubbed to decode the response
and return `response.Resources` only, dropping the
`BypassRateLimitCheck` context marker (the endpoint isn't itself
rate-limited) and the loop that writes each populated
`Resources.<category>` back into `client.rateLimits[category]` under
`rateMu`. The request, decode, and return keep working.
Kept: RateLimits/Rate structures, all category constants, client rate
fields.
Tests removed: 3 funcs in github/rate_limit_test.go (TestRateLimits,
TestRateLimits_overQuota, TestRateLimits_bypassRateLimitCheckContext).
