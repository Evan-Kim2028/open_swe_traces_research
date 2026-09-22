# Contract (L2) — credhdrs

- The credential-carrying request is a copy: the caller's request object is
  never mutated — no header is added to it, and the transport forwards a
  different request instance.
- The copy carries a Basic-Auth `Authorization` header built from the
  configured credentials (base64 of `user:pass`).
- The copied request's header map is independent of the original's: after the
  round trip the caller's headers contain no `Authorization` entry and its
  map is unchanged in size.
- The transports consult an origin scope before attaching credentials: a
  request to a foreign origin goes out without credentials, while a request
  to the documented default origin carries them. The exact origin-matching
  rule is the sibling unit's (urlpolicy) commitment and is not pinned here.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — caller's request unmutated; transport forwarded a copy |
| `TestDetail02` | 2 — Basic-Auth header equals base64(user:pass) on the copy |
| `TestDetail03` | 3 — caller's header map unchanged and still lacks Authorization |
| `TestDetail04` | 4 — foreign origin receives no credentials; default origin does (shape) |
