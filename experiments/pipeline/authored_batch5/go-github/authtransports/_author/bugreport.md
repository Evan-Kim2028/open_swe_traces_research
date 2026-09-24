# Bug report — authtransports

Auth transports leak credentials to arbitrary origins. Following a
redirect, or issuing a request to a host outside `AllowedOrigins`,
sends `client_id`/`client_secret` query params (and the `OTP` header, or
HTTP basic auth) to that foreign host.

Expected: credential material is attached only when the destination
origin is in `AllowedOrigins` (or the configured base-URL origin when
the list is empty); other requests pass through unmodified.

Got: credentials are attached to every request regardless of origin.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
