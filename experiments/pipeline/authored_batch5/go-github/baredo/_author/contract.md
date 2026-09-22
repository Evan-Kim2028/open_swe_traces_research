# Contract — baredo

`Client.bareDo` enforces cached rate limits before the network call, updates
rate state from responses, honors secondary limits, sanitizes error URLs,
and manages response-body lifetime. Every commitment below is covered by a
hidden test; every hidden test maps to a commitment.

## Commitments

1. **Cached limit short-circuits.** Before any network call, an exhausted
   cached rate limit for the request's category returns `*RateLimitError`
   with a populated `Response`, and `BypassRateLimitCheck` in the context
   skips the check. Covered by `TestDetail01`.
2. **Rate state updated.** After a response the client's per-category rate
   state is updated from the response headers — unless rate checks are
   disabled. Covered by `TestDetail02`.
3. **Secondary limit recorded.** An abuse/secondary rate-limit error
   surfaces as `*AbuseRateLimitError` and the secondary limit is recorded
   so subsequent calls short-circuit without new network traffic. The
   bounded retry mechanics are asserted at behavior level — the limit is
   honored — not by pinning a sleep duration. Covered by `TestDetail03`.
4. **Caller cancellation wins.** When the caller's context is cancelled
   mid-flight the returned error is the context's error
   (`errors.Is(err, context.Canceled)`). Covered by `TestDetail04`.
5. **Error URL sanitized (shape).** A `*url.Error` surfaced through
   `bareDo` does not retain sensitive query material on its `URL` field —
   asserted as error type plus absence of the secret, not the redaction
   literal. Covered by `TestDetail05`.
6. **AcceptedError capture.** An `*AcceptedError` response has its raw
   body captured into `AcceptedError.Raw` and the original body closed.
   Covered by `TestDetail06`.
7. **Error body closed.** On any error response the original network body
   is closed. Covered by `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially — asserted as error type + recorded-window short-circuit |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | no — shape only |
| TestDetail06 | 6 | doc |
| TestDetail07 | 7 | yes |
