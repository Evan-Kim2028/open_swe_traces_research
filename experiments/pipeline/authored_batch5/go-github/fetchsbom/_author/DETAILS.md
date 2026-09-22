# DETAILS — fetchsbom

1. With a non-nil `followRedirectsClient`, the redirect target is
   fetched through that client — not the API client — and the decoded
   `*SBOM` is returned with an empty `redirectURL`. Inferable: doc — the
   parameter exists for this; the credential-exclusion rationale is in
   the doc comment.
2. With a nil `followRedirectsClient`, `redirectURL` is returned and no
   fetch occurs. Inferable: yes.
3. The redirect fetch is validated: an error HTTP status propagates as
   an error rather than decoding the error page. Inferable: partially.
4. The original network body is closed on every path (success and
   error). Inferable: yes.
5. The downloaded payload decodes as `SBOMInfo` wrapped in `*SBOM`.
   Inferable: partially — field placement is arbitrary; assert a
   populated SBOM, not the wrapper shape.
