# Exported API — ratehdrs

`Response.Rate` and `Response.TokenExpiration` expose rate-limit and
token-expiry metadata lifted straight off the response headers.

- `parseRate` (unexported, excised) — fills `Rate` from
  `X-RateLimit-*` headers.
- `parseTokenExpiration` (unexported, excised) — fills
  `TokenExpiration` from the `GitHub-Authentication-Token-Expiration`
  header.
