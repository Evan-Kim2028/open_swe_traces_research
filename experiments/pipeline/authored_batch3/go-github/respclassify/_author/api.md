# Exported API — respclassify

`CheckResponse` is the funnel every API call passes through: it turns an
HTTP status + headers + body into the concrete error type callers
type-switch on.

- `CheckResponse` — maps responses onto nil, ErrorResponse,
  TwoFactorAuthError, RateLimitError, AbuseRateLimitError,
  RedirectionError, AcceptedError.
- `parseSecondaryRate` (unexported, excised) — derives the retry delay
  from `Retry-After` or `X-RateLimit-Reset` headers for secondary limits.
