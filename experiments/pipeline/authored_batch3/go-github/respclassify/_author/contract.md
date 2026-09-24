# Contract (L2) — respclassify

- A 202 Accepted response yields `*AcceptedError`; other 2xx statuses yield
  nil.
- The error path decodes the body into `ErrorResponse` and restores the body
  afterward so the caller can re-read it.
- A 401 carrying an OTP-required header yields `*TwoFactorAuthError`, a cast
  of the parsed error response (message carried through); a plain 401 stays
  a plain `*ErrorResponse`.
- A 403 or 429 with rate-limit-remaining 0 yields `*RateLimitError` carrying
  the parsed rate and the response message.
- A 403/429 whose `documentation_url` ends in the secondary-limit fragment
  documented on the error type yields `*AbuseRateLimitError`, with
  `RetryAfter` filled from the secondary-rate parsing when derivable.
- Redirect statuses yield `*RedirectionError` carrying the status code and
  the parsed `Location` header; the suite asserts the documented members of
  the redirect set.
- The secondary-rate retry delay prefers the `Retry-After` header (seconds)
  and otherwise derives a duration-until from the rate-reset epoch header.
- Any other status yields the plain `*ErrorResponse` — none of the
  specialized types.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — 202 → `*AcceptedError`; other 2xx → nil |
| `TestDetail02` | 2 — body decoded into `ErrorResponse` and restored re-readable |
| `TestDetail03` | 3 — 401 + OTP-required → `*TwoFactorAuthError` with message; plain 401 stays plain |
| `TestDetail04` | 4 — 403/429 + remaining=0 → `*RateLimitError` with rate and message |
| `TestDetail05` | 5 — documented secondary-limit fragment → `*AbuseRateLimitError` + `RetryAfter` (shape) |
| `TestDetail06` | 6 — redirect statuses → `*RedirectionError` with status + location (shape) |
| `TestDetail07` | 7 — `Retry-After` preferred; reset-epoch fallback as duration-until |
| `TestDetail08` | 8 — other statuses → plain `*ErrorResponse` only |
