# Details — wshandshake

1. `wsAcceptKey` returns base64(SHA1(key + wsGUID)) where wsGUID is the RFC
   6455 magic string — computed under `fips140.WithoutEnforcement` because
   SHA-1 is used for protocol compat, not security. Inferable: doc — RFC
   formula is standard, the FIPS carve-out is a choice.
2. `wsMakeChallengeKey` returns base64 of 16 crypto-random bytes.
   Inferable: doc — RFC 6455 requires 16 bytes.
3. `wsHeaderContains` splits each header value on `,`, trims ` \t`, and
   matches case-INSENSITIVELY (`EqualFold`). Inferable: partially — token
   matching is implied by usage, the trim set and fold are internal.
4. `wsPMCExtensionSupport` scans `Sec-Websocket-Extensions` values for the
   `permessage-deflate` token and returns (supported, noContextTakeover);
   with `checkPMCOnly` it stops at the extension name. Otherwise it scans
   ONLY the parameters of that extension for `server_no_context_takeover`
   AND `client_no_context_takeover` — both must be present for the second
   return to be true; an extension param before the name or on a different
   extension doesn't count. Inferable: partially — header name visible,
   param-scoping rule is not.
5. `wsGetHostAndPort` splits host:port; a "missing port" AddrError is
   swallowed and replaced with "443" (tls) or "80" (non-tls); the host is
   lowercased; other split errors propagate. Inferable: no.
6. `checkOrigin`: if neither `sameOrigin` nor a non-empty allowed list is
   configured it accepts; a MISSING Origin header (also checked as
   `Sec-Websocket-Origin`) accepts unconditionally. Inferable: doc — the
   comment cites the RFC rationale.
7. `checkOrigin` parses Origin as a request URI and extracts host/port via
   `wsGetHostAndPort` (so a scheme-implied default port applies); under
   `sameOrigin` host, port, AND scheme must match the request's (scheme
   compared case-insensitively, request scheme derived from `r.TLS != nil`);
   under an allowed list the (scheme, port) pair must match an entry for
   that host — both checks can apply (sameOrigin does not skip the list).
   Inferable: partially — the dual-check fallthrough is body-internal.
8. Under `sameOrigin` with an explicit non-default port in the request Host
   header (e.g. `host.com:443` with TLS), `wsGetHostAndPort` keeps it —
   `443`==`443` but the comparison is against the request's own default
   resolution, so `host.com:443` request vs `host.com` origin fails on
   port. Inferable: no.
