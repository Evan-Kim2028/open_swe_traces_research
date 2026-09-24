# Exported API — urlpolicy

Request URL policy: where credentials may flow, where bodies may be sent,
and which URL shapes are refused outright.

- `checkURLPathTraversal` (unexported, excised) — rejects URLs containing
  `..` path segments; used by NewRequest/NewFormRequest/NewUploadRequest.
- `sameOrigin`, `normalizedPort`, `isAllowedOrigin` (unexported, excised) —
  origin equality with case- and default-port-insensitive comparison, plus
  the allow-list check.
- `Client.shouldAuthorizeRequest`, `Client.checkBodyDestination`
  (unexported, excised) — credential-scoping predicates for base/upload
  origins.
- `ErrPathForbidden`, `ErrUntrustedDestination`, `defaultAuthOrigins` —
  kept sentinels.
