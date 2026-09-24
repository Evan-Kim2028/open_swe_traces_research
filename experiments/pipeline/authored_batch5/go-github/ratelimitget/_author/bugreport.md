# Bug report — ratelimitget

`RateLimitService.Get` goes through the rate-limit gate instead of
bypassing it (the endpoint itself is exempt), and the fetched limits
are never written back — the client's `rateLimits` map stays stale, so
subsequent calls in over-quota categories keep hitting the network.

Expected: the request bypasses the rate check, and each returned
category refreshes `client.rateLimits`.

Got: the call is rate-checked like any other, and the client's rate
state never updates.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
