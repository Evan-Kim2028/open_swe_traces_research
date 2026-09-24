# DETAILS — baredo

1. Before any network call, an exhausted cached rate limit for the
   request's category returns `*RateLimitError` with a populated
   `Response` — no HTTP request is made. `BypassRateLimitCheck` in the
   context skips the check. Inferable: doc — documented on the method.
2. After a response, the client's per-category rate state is updated
   from response headers — unless rate checks are disabled or the
   response carries `X-From-Cache`. Inferable: partially — updating is
   derivable; the cache-header exemption is an arbitrary carve-out.
3. A secondary/abuse limit (`*AbuseRateLimitError` with `Retry-After` or
   `X-RateLimit-Reset`) sleeps then retries once, and records the
   secondary limit so subsequent calls short-circuit.
   Inferable: partially — retry-once is derivable; sleep bounds are
   arbitrary.
4. When the caller's context is cancelled mid-flight, the returned error
   is the context's error, not the transport error.
   Inferable: partially — preferring ctx.Err() is conventional.
5. `*url.Error` values have their URL sanitized (query stripped) before
   being returned. Inferable: no — the sanitizer's exact redaction set
   is arbitrary; assert shape (error type + no leaked query).
6. An `*AcceptedError` response has its raw body captured into
   `AcceptedError.Raw` and the original body is closed. Inferable: doc —
   the Raw field exists for this; large bodies are truncated (bound is
   arbitrary — assert non-empty + closed, not the cap).
7. On any error response the original network body is closed.
   Inferable: yes.
