# Exported API — erris

Every package error type supports `errors.Is` comparison, so callers can
match an error value against an expected instance field-by-field.

- `ErrorResponse.Is`, `RateLimitError.Is`, `AbuseRateLimitError.Is`,
  `RedirectionError.Is`, `AcceptedError.Is` — per-field equality behind
  `errors.As` type checks.
- `compareHTTPResponse` (unexported, excised) — nil-safe StatusCode
  comparison shared by all Is methods.
- `equalDurationPtr` (unexported, excised) — nil-safe *time.Duration
  comparison for RetryAfter.
