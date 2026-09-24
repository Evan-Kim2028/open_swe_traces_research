# Exported API — uploadreq

`Client.NewUploadRequest(ctx, urlStr, reader, size, mediaType, opts...)`
builds the POST request used for release-asset and similar uploads.
Callers rely on:

- `urlStr` resolved against the client's upload base URL, with `..`
  path segments rejected and absolute destinations restricted to
  configured origins (`ErrUntrustedDestination`).
- `reader` becoming the request body with `ContentLength = size`.
- `GetBody` populated when `reader` supports rewinding, so redirects and
  HTTP/2 retries can replay the body.
- `mediaType` defaulting when empty.
