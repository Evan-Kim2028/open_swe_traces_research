# Contract — wshandshake

WebSocket upgrade validation: handshake key derivation, header token
matching, permessage-deflate negotiation, and origin checking. Every
commitment below is covered by a hidden test; every hidden test maps to
a commitment.

## Commitments

1. **Accept key.** The accept key is base64 of SHA-1 over the challenge
   key concatenated with the RFC 6455 magic GUID. Covered by
   `TestDetail01`.
2. **Challenge key.** The outbound challenge key is base64 of 16
   crypto-random bytes. Covered by `TestDetail02`.
3. **Header token match.** Header values are split on commas, trimmed,
   and matched case-insensitively. Covered by `TestDetail03`.
4. **Extension negotiation.** The permessage-deflate token reports
   support; the second return requires both no-context-takeover params
   on that extension itself; params on other extensions do not count;
   the check-only mode skips the param scan. Covered by `TestDetail04`.
5. **Host/port split.** A missing port is substituted with the scheme
   default (443 tls / 80 plain), the host is lowercased, and other
   split errors propagate. Covered by `TestDetail05`.
6. **Origin acceptance.** With no origin policy configured any origin
   is accepted; a missing Origin header accepts unconditionally.
   Covered by `TestDetail06`.
7. **Origin comparison.** Same-origin requires matching host, port, and
   scheme (request scheme derived from TLS presence); an allowed-list
   entry must match the (scheme, port) pair for that host; both checks
   apply. Covered by `TestDetail07`.
8. **Explicit port.** An explicit request port is kept through the
   host/port split, so it is compared against the origin's own resolved
   port. Covered by `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc — RFC test vector |
| TestDetail02 | 2 | doc — decoded length asserted |
| TestDetail03 | 3 | partially — token split + case-fold asserted |
| TestDetail04 | 4 | partially — param-scoping rule asserted via mis-scoped params |
| TestDetail05 | 5 | no — shape: defaults, lowercase, error propagation |
| TestDetail06 | 6 | doc — unconfigured and missing-header acceptance asserted |
| TestDetail07 | 7 | partially — host/port/scheme + dual-check asserted |
| TestDetail08 | 8 | no — shape: explicit-vs-default port mismatch fails, matching ports pass |
