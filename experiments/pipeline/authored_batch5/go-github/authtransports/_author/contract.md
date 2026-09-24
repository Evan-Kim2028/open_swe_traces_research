# Contract — authtransports

`UnauthenticatedRateLimitedTransport` and `BasicAuthTransport` attach
credentials only to requests whose origin is allowed. Every commitment
below is covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Allowed origins get credentials.** Credentials are attached when the
   request origin matches `AllowedOrigins` — or the transport's configured
   default origin when the list is empty. Covered by `TestDetail01`.
2. **Disallowed origins pass clean.** A request to a disallowed origin is
   passed through with no credential material added — no auth header and no
   credential query parameters — while existing query values survive.
   Covered by `TestDetail02`.
3. **OTP conditional.** The OTP header is sent only when the transport's
   OTP is non-empty. Covered by `TestDetail03`.
4. **Query preserved.** Credentials are conveyed while preserving existing
   query values on the request URL. Covered by `TestDetail04`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | doc |
| TestDetail04 | 4 | partially |
