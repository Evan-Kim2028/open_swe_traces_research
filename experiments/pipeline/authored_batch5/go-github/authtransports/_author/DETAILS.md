# DETAILS — authtransports

1. Credentials (client_id/client_secret params, OTP header, or basic
   auth) are attached only when the request URL's origin is allowed —
   matching `AllowedOrigins`, or the transport's configured base URL
   origin when `AllowedOrigins` is empty. Inferable: doc — the field is
   documented as the credential boundary; the empty-list fallback is the
   documented default.
2. Requests to a disallowed origin pass through with no credential
   material added. Inferable: doc — credential non-leakage is the whole
   point of the field.
3. The `OTP` header is sent only when `OTP` is non-empty.
   Inferable: partially — conditional header emission is derivable; the
   header name spelling is arbitrary.
4. client_id/client_secret are added as URL query parameters preserving
   existing query values. Inferable: partially — query-param encoding is
   a documented API convention.
