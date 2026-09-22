# Contract (L2) — brnotprot

- The translation predicate fires on errors carrying the documented
  "not protected" message; a different message on the same HTTP status is not
  translated.
- Non-API errors never match: a transport-level failure propagates as itself.
- A matching error is returned to the caller as `ErrBranchNotProtected`
  (matching `errors.Is`), replacing the raw API error.
- Any other error is propagated unchanged — a response with a different
  message keeps its own message and does not map to the sentinel.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — documented message vs. different message on 404 |
| `TestDetail02` | 2 — transport failure never translates (shape; non-API errors don't match) |
| `TestDetail03` | 3 — matching error → `errors.Is(err, ErrBranchNotProtected)` |
| `TestDetail04` | 4 — different message propagates unchanged, carrying its own message |
