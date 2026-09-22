# Exported API — newformreq

`Client.NewFormRequest(ctx, urlStr, formBuf)` builds a multipart POST
against the API base URL. Callers rely on the same header contract as
NewRequest (User-Agent, API version) plus the form-destination rules:

- The resolved form URL must be a configured destination when absolute,
  and `urlStr` itself must not resolve to a foreign host
  (`ErrUntrustedDestination`).
