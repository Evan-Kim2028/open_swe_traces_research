# Contract (L2) — runidre

- A callback URL containing the run-id pattern (`repos/.../actions/runs/
  <digits>/deployment_protection_rule`) yields that digit run as an int64.
- Both absolute and relative URLs satisfying the pattern work — the match is
  not anchored to a scheme.
- A URL that does not match returns -1 and an error.
- A matched digit run that overflows int64 parsing returns -1 and an error;
  which error is propagated is implementation detail.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — matching absolute URL → run id value |
| `TestDetail02` | 2 — absolute and relative forms both match |
| `TestDetail03` | 3 — non-matching URLs → (-1, error) |
| `TestDetail04` | 4 — overflowing digit run → (-1, error) (shape; error literal unpinned) |
