# Exported API — baredo

`Client.bareDo(ctx, caller, req)` is the single egress point behind
`Client.Do`. Everything observable flows through it:

- Per-category rate-limit checks before the network call; a cached
  exhausted limit short-circuits without touching the network.
- Response headers update the client's rate state (`Rate` fields),
  including secondary/abuse limits and `Retry-After` /
  `X-RateLimit-Reset` handling, with bounded sleep-and-retry.
- `context.Canceled` from the caller's context surfaces as a
  cancellation error, not an HTTP error.
- Accepted-error responses (e.g. 202 with an `AcceptedError`) capture
  the raw body into `AcceptedError.Raw`.
- Error messages carry a sanitized URL (query stripped as configured).
- On error the original response body is always closed.
