# Contract — ratelimitget

`RateLimitService.Get` fetches the rate-limit status without tripping the
client's own pre-checks, and refreshes the client's cached per-category
rate state from the response. Every commitment below is covered by a
hidden test; every hidden test maps to a commitment.

## Commitments

1. **Bypass marker.** The request is made with `BypassRateLimitCheck` in
   the context — an exhausted cached limit does not short-circuit `Get`
   (unless rate checks are disabled entirely). Asserted behaviorally: a
   seeded exhausted limit does not stop the request from reaching the
   network. Covered by `TestDetail01`.
2. **State written back.** Each populated `Resources.<category>` in the
   response is written into `client.rateLimits[category]` — asserted via
   the map contents after the call, not the field list. Covered by
   `TestDetail02`.
3. **Nil inputs leave the map.** A nil `Resources` or a nil category
   pointer leaves the map untouched for that category. Covered by
   `TestDetail03`.
4. **Resources returned.** `response.Resources` is returned to the
   caller. Covered by `TestDetail04`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | yes |
