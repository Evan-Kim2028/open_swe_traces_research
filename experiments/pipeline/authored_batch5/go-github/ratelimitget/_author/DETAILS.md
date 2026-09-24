# DETAILS — ratelimitget

1. The request is made with `BypassRateLimitCheck` set in the context,
   unless the client has rate checks disabled entirely. Inferable: doc —
   the endpoint is documented as not rate-limited; the bypass marker
   exists for this.
2. Each populated `Resources.<category>` in the response is written back
   into `client.rateLimits[category]` under `rateMu`.
   Inferable: partially — that Get refreshes the client's view is
   derivable; the per-category mapping is internal (assert the map
   updates, not the field list).
3. Nil `Resources` or a nil category pointer leaves the map untouched
   for that category. Inferable: partially.
4. `response.Resources` is returned to the caller. Inferable: yes.
