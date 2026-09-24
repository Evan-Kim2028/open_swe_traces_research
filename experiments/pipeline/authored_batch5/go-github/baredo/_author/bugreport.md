# Bug report — baredo

Every request through the client bypasses rate-limit discipline and
loses error metadata:

- Cached exhausted rate limits no longer short-circuit — calls hit the
  network even when the client knows the limit is spent.
- Response headers no longer update the client's rate state, so
  `Client.Rate`-adjacent bookkeeping never changes.
- Secondary/abuse limits are not honored: no `Retry-After` /
  `X-RateLimit-Reset` sleep-and-retry, no recorded secondary limit.
- Cancelling the request context surfaces the transport's url.Error
  instead of `context.Canceled`, and the error URL leaks its query
  string.
- `*AcceptedError` responses (e.g. 202 deferred-create endpoints) lose
  their raw body — `AcceptedError.Raw` is empty — and the network body
  is left unclosed on error paths.

Expected: the documented rate-limit, cancellation, sanitization, and
accepted-error behaviors all hold.

Got: a bare send with none of the above.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
