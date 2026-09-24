# Exported API — authtransports

Two `http.RoundTripper` implementations the client uses for
authentication:

- `UnauthenticatedRateLimitedTransport` adds `client_id`/`client_secret`
  query parameters (and `OTP` when `OTP != ""`) to raise rate limits.
- `BasicAuthTransport` sets HTTP Basic auth from `Username`/`Password`.

Both share the `AllowedOrigins` contract: credentials are attached only
to requests whose URL origin is allowed (configured `AllowedOrigins`, or
the client base URL origin when the list is empty). Foreign origins go
through untouched.
