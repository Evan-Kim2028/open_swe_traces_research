# Contract (L2) — errfmt

- `ErrorResponse.Error` renders the components it has: request method and URL
  plus status code and message when the request is known; status code and
  message when only the response is known; the message alone otherwise. The
  exact separators and verbs are implementation detail.
- `RateLimitError.Error` renders the request line (method, URL, status,
  message) plus a reset clause derived from the rate's reset time: a future
  reset adds output beyond the bare rendering.
- `AbuseRateLimitError.Error` adds a retry clause only when `RetryAfter` is
  set and positive, carrying the delay as a whole-second count; a nil
  `RetryAfter` produces no such clause.
- `AcceptedError.Error` returns a fixed non-empty message independent of the
  `Raw` payload.
- `RedirectionError.Error` renders the status code and the redirect target.
- `Error.Error` renders the populated code, field and resource values.
- `Error.UnmarshalJSON` accepts a bare JSON string, placing it in `Message`,
  and accepts the structured object form filling `Resource`, `Field`, `Code`.
- `sanitizeURL` removes the *values* of sensitive query parameters from the
  rendered URL while leaving unrelated parameters and the host intact; the
  redaction marker spelling is implementation detail.
- The rate-reset rendering distinguishes future resets from past ones
  (different output for each direction), always carries a reset indication,
  and resolves sub-second differences to the same rendering.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — three-level component fallback (shape; verbs not pinned) |
| `TestDetail02` | 2 — request line + reset-derived suffix (shape) |
| `TestDetail03` | 3 — positive `RetryAfter` lengthens output carrying seconds; nil does not (shape) |
| `TestDetail04` | 4 — fixed message, payload-independent (shape) |
| `TestDetail05` | 5 — status code + location target present (shape) |
| `TestDetail06` | 6 — code/field/resource values present (shape) |
| `TestDetail07` | 7 — bare-string fallback lands in `Message`; object form decodes (shape) |
| `TestDetail08` | 8 — secret values absent, unrelated params and host intact (shape; marker not pinned) |
| `TestDetail09` | 9 — future/past divergence, reset indication, sub-second rounding (shape) |
