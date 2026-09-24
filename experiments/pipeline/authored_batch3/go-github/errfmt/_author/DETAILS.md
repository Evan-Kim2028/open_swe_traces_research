# DETAILS — errfmt

1. `ErrorResponse.Error` renders `METHOD url: CODE message [errors]` when the
   request is known, `CODE message [errors]` when only the response is known,
   `message [errors]` otherwise. Inferable: partially — the three-level
   fallback shape is derivable, the exact format verbs are arbitrary.
2. `RateLimitError.Error` is `METHOD url: CODE message <reset-suffix>` where
   the suffix comes from formatRateReset. Inferable: no — format literal is
   arbitrary.
3. `AbuseRateLimitError.Error` appends ` [retry after D]` only when
   RetryAfter is set and positive, D rounded to seconds. Inferable: no —
   arbitrary literal and edge rule.
4. `AcceptedError.Error` returns a fixed literal message. Inferable: no —
   the literal is arbitrary.
5. `RedirectionError.Error` renders `METHOD url: CODE location <sanitized>`.
   Inferable: no — arbitrary literal.
6. `Error.Error` renders `<code> error caused by <field> field on <resource>
   resource`. Inferable: no — arbitrary literal.
7. `Error.UnmarshalJSON` first tries the structured object; on failure it
   retries unmarshalling a bare JSON string into Message. Inferable: no —
   the string-fallback rule is arbitrary.
8. `sanitizeURL` rewrites `client_secret`, `access_token`, `token` query
   values to REDACTED and leaves other params untouched; nil URL → nil.
   Inferable: partially — the sensitiveParams list survives, the REDACTED
   marker and nil handling are arbitrary.
9. `formatRateReset` renders `[rate reset in MmSSs]` for positive durations
   and `[rate limit was reset MmSSs ago]` for negative, rounding to the
   nearest second and dropping the minute field when under a minute.
   Inferable: no — literal and rounding rules are arbitrary.
