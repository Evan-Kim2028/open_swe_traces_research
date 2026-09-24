# Exported API — newreq

`Client.NewRequest(ctx, method, urlStr, body, opts...)` builds the
`*http.Request` every service method sends. Callers and tests rely on:

- The request URL resolved against the client's base URL.
- `Accept`, `Content-Type` (when body present), `User-Agent`, and
  `X-Github-Api-Version` headers populated from client configuration.
- `opts` applied last, so callers can override anything above.
