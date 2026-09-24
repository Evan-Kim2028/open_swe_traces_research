# Contract (L2) — boolresp

- A boolean-answer API call that succeeds (nil error) reports `true` with no
  error.
- An API error carrying HTTP 404 is swallowed to `false` with no error — the
  missing-resource answer means "not starred", not failure. Only the 404
  status is special-cased.
- Any other error — an API error with a different status, or a non-API
  transport error — propagates unchanged as `false` plus the error.
- The 404 detection keys on the response's status, not its body: a 404 with a
  non-JSON body, an empty body, or an empty object still reports
  `(false, nil)`. Whether *wrapped* error values also match is an
  implementation edge the suite does not pin — it is unreachable through the
  exported surface.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — 200 response → `(true, nil)` via `IsStarred` |
| `TestDetail02` | 2 — 404 response → `(false, nil)` |
| `TestDetail03` | 3 — 500 and transport failure → `(false, err)` |
| `TestDetail04` | 4 — 404 detection keys on status regardless of body (shape; wrapped-error edge unasserted) |
