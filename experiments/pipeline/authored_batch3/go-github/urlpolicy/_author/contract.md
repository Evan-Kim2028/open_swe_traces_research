# Contract (L2) — urlpolicy

- A URL whose path contains a literal `..` segment is rejected with
  `ErrPathForbidden`; `..` inside a segment (e.g. `file..txt`) or inside the
  query string is fine. The check applies to both the request and upload
  constructors.
- Origin equality is case-insensitive on scheme and hostname, and explicit
  ports normalize against scheme defaults (https→443, http→80): an
  upper-cased host or an explicit default port still matches the configured
  origin, while a different scheme or a non-default port does not.
- An empty allow-list on the credential transports means the package's
  default origins — the GitHub.com API and upload hosts — not "allow all":
  credentials flow to the defaults and are withheld from any other origin.
- A client's credentials may flow to its configured base or upload origin;
  requests to any other origin go out unauthenticated.
- Sending a body to an out-of-scope destination is refused with an error
  matching `ErrUntrustedDestination` that names the (sanitized) destination;
  the refusal happens at request-build time, before any bytes are sent. The
  message wording is implementation detail.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — `..` segment → `ErrPathForbidden` on both constructors; in-segment/query `..` accepted |
| `TestDetail02` | 2 — case-insensitive + default-port origin matching via credential flow/withholding |
| `TestDetail03` | 3 — empty allow-list → default origins credentialed, others not |
| `TestDetail04` | 4 — base and upload origins credentialed; other origins not |
| `TestDetail05` | 5 — foreign body destination → `ErrUntrustedDestination` naming the host (shape) |
