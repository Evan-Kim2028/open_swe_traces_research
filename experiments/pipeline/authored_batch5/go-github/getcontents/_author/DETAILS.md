# DETAILS — getcontents

1. The request path is escaped as a URL path segment (spaces, `+`, `..`
   normalized) and a trailing `/` is trimmed, so `path` never breaks
   the route. Inferable: partially — escaping is derivable; the
   url.URL{Path} idiom and trim are arbitrary mechanics.
2. The response body is decoded twice: object → `fileContent`, else
   array → `directoryContent`. Exactly one is non-nil on success.
   Inferable: yes — the endpoint legitimately returns both shapes.
3. When neither shape decodes, the error reports both failures rather
   than just one. Inferable: no — the combined-error wording is
   arbitrary; assert an error surfaces, not the message.
4. `opts` (ref/sha query) apply to the request URL. Inferable: yes.
