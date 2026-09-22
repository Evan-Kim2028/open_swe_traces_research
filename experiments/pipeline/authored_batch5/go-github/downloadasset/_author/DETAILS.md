# DETAILS — downloadasset

1. When the API responds without a redirect (`loc == nil`), the
   response body is returned directly as the stream. Inferable: yes.
2. On redirect with a non-nil `followRedirectsClient`, the redirect
   target is fetched through that client — not the API client — and
   the fetched body is returned with an empty `redirectURL`.
   Inferable: doc — the parameter exists precisely to download through
   a separate credential-free client.
3. On redirect with a nil `followRedirectsClient`, `redirectURL` is
   returned and no fetch occurs. Inferable: yes.
4. The redirect-fetch response is validated (`CheckResponse`): an error
   status propagates and the original network body is closed; a success
   returns its body for the caller to close. Inferable: partially —
   validation is derivable; which body must be closed on the error path
   is internal (assert no leak, not the mechanism).
5. When a redirect is followed or returned, the original API response
   body is closed first. Inferable: partially.
