# Exported API — clientclone

`Client.Clone(opts...)` returns a second client that behaves like the
receiver unless overridden by `opts`. Callers rely on:

- Configuration carry-over: base/upload URLs, user agent, API version
  range, auth token, rate-limit-check flags, marketplace stub.
- An uninitialized (`client == nil`) receiver returning
  `errUninitialized`.
- A token-bearing clone installing its auth transport against its own
  allowed origins — re-scoping credentials rather than reusing the
  parent's wrapped transport.
- Rate-limit state shared with the parent (one `rateLimits` map) so
  either client sees limits the other learned.
- `CheckRedirect`, `Jar`, `Timeout` preserved on the HTTP client.
