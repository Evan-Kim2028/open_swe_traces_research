# Exported API — credhdrs

Credential-carrying transports (`BasicAuthTransport`,
`UnauthenticatedRateLimitedTransport`, OAuth apps) inject an Authorization
header per request without mutating the caller's request.

- `setCredentialsAsHeaders(req, id, secret) *http.Request` (unexported,
  excised body) — returns a copy of req with basic-auth credentials applied.
- `BasicAuthTransport` / `UnauthenticatedRateLimitedTransport` — RoundTrippers
  routing through it.
