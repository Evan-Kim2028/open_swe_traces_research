# Contract (L2) — enturls

- An empty or unparseable URL string passed through the URL options is an
  error at client construction; a valid URL is accepted.
- URL parsing normalizes the path to end in `/`, appending one when missing.
- The enterprise-URL option extends a plain enterprise host's API path with
  the enterprise API prefix convention; hosts already carrying that
  convention are not extended twice, and `api`-designated hosts (the
  `api.*` and `*.api.*` shapes) are exempt. The literal appended path is a
  convention, not part of the contract — what is contracted: plain hosts gain
  a non-empty extended path ending in `/`, api-designated hosts gain nothing.
- The upload URL is extended under the same rule, with an additional
  `uploads.*` host exemption. Same shape contract as above.
- The URL mutation happens at client-construction time, visible on the
  client's configured URLs before any request is made.
- The verbatim URL option performs only the trailing-slash normalization —
  no enterprise rewriting: configured base and upload URLs are preserved
  verbatim apart from the appended slash.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — empty URL rejected; valid accepted |
| `TestDetail02` | 2 — trailing slash ensured |
| `TestDetail03` | 3 — plain host extended, api-designated hosts exempt, idempotent (shape; literal not pinned) |
| `TestDetail04` | 4 — upload URL extended; `uploads.*` exempt (shape) |
| `TestDetail05` | 5 — mutation visible at construction time |
| `TestDetail06` | 6 — verbatim option: URLs preserved apart from trailing slash |
