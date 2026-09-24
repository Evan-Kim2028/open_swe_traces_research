# Closure — authtransports

Package: github (root). File: github/github.go.
Removed bodies: UnauthenticatedRateLimitedTransport.RoundTrip and
BasicAuthTransport.RoundTrip — stubbed to variants that attach
credentials (clientID/secret query params, OTP header; or HTTP basic
auth) unconditionally, dropping the `isAllowedOrigin(req.URL)` gate that
limits credential forwarding to `AllowedOrigins` (or the configured
baseURL when the allowlist is empty). Transport/OTP plumbing and the
RoundTrip call itself keep working; a redirect or foreign-host URL now
leaks credentials.
Kept: isAllowedOrigin, UnauthenticatedRateLimitedTransport fields,
BasicAuthTransport fields, Client.
Tests removed: 2 funcs in github/github_test.go
(TestUnauthenticatedRateLimitedTransport_originScope,
TestBasicAuthTransport_originScope).
