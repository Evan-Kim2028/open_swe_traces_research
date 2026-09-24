# Contract (L2) — gethdr

- Hook-request and hook-response header lookup is case-insensitive: any
  letter-casing of the queried key matches the stored key, on both public
  types.
- A key absent under every casing returns the empty string — including on a
  type with no stored headers.
- The stored value is returned verbatim: surrounding whitespace and letter
  casing in the value survive the lookup.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — multiple casings resolve to the stored value, both types |
| `TestDetail02` | 2 — absent key and nil map return empty string |
| `TestDetail03` | 3 — verbatim value return (whitespace/case preserved) |
