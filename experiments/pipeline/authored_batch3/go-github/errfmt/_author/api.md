# Exported API — errfmt

Every API error type renders a human-readable `Error()` string embedding the
failing request, and credential-bearing query params are scrubbed before
display.

- `ErrorResponse.Error`, `RateLimitError.Error`, `AbuseRateLimitError.Error`,
  `RedirectionError.Error`, `AcceptedError.Error`, `Error.Error` — message
  formats.
- `Error.UnmarshalJSON` — tolerant decoder for GitHub's polymorphic `errors`
  payloads.
- `sanitizeURL` (unexported, excised) — redacts `client_secret`,
  `access_token`, `token` query params.
- `formatRateReset` (unexported, excised) — renders the rate-limit reset
  countdown inside error strings.
