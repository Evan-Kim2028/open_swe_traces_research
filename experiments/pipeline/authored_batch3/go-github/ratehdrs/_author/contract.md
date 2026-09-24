# Contract (L2) — ratehdrs

- Rate-limit response headers populate the response's rate fields: limit,
  remaining, used and resource each land on their documented field when the
  corresponding header is present.
- The reset header leaves the reset timestamp unset (zero) when the header
  is absent or parses to 0; a nonzero epoch parses to its instant.
- The token-expiration header accepts both the zone-name layout and the
  numeric-offset layout, returning the instant either way; the exact layout
  spellings are conventional.
- A missing or unparseable expiration header yields the zero timestamp.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — limit/remaining/used/resource fields from headers |
| `TestDetail02` | 2 — absent or `"0"` reset → zero timestamp; nonzero epoch parses (shape) |
| `TestDetail03` | 3 — both documented layouts parse to the instant |
| `TestDetail04` | 4 — missing/unparseable → zero timestamp |
