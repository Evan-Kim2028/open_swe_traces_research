# Contract (L2) — erris

- Each error type's `Is` accepts only targets of its own concrete type; a
  cross-type comparison is false, and `errors.Is` surfaces that.
- Two error values' embedded responses compare equal iff both are nil or both
  carry the same status code; other response differences are ignored.
- `ErrorResponse` equality covers the documented payload set: identical
  values match; differing message, differing per-item `Errors`, or differing
  response status each break equality.
- `RateLimitError` equality covers `Rate` (struct equality), `Message`, and
  the response status.
- `AcceptedError` equality is decided by the `Raw` byte payload: identical
  payloads match, differing payloads do not.
- `AbuseRateLimitError` equality covers `Message`, response status, and
  `RetryAfter` compared nil-safely by pointed-to value.
- `RedirectionError` equality covers `StatusCode` and `Location`, where
  distinct URL values with equal textual forms count as equal and a
  nil-vs-set location does not.
- Duration-pointer comparison is nil-safe: both-nil equal, nil-vs-set
  unequal, equal pointed-to durations equal regardless of pointer identity.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — cross-type `errors.Is` is false; same-type identical pair is true |
| `TestDetail02` | 2 — nil/nil equal, nil/set unequal, only StatusCode compared |
| `TestDetail03` | 3 — equal match; Message/Errors/StatusCode diffs break it |
| `TestDetail04` | 4 — Rate and Message differences break equality |
| `TestDetail05` | 5 — Raw identity: equal bytes match, differing bytes do not (shape) |
| `TestDetail06` | 6 — RetryAfter/Message differences break equality; nil-safe |
| `TestDetail07` | 7 — same-string URLs equal; StatusCode and nil-vs-set Location differ (shape) |
| `TestDetail08` | 8 — nil/nil, nil/set, and equal-values pointer cases |
