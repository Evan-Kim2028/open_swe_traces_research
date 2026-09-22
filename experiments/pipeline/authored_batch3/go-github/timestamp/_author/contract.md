# Contract (L2) — timestamp

- Quoted RFC3339 strings decode normally, including fractional-second forms.
- A bare JSON number decodes as a Unix-seconds timestamp.
- A bare number too large to be a sane seconds value decodes with
  millisecond granularity — the committed example values decode to their
  millisecond instants. The exact rule deciding which inputs flip
  granularity is implementation detail and is not part of the contract.
- A value at the edge of the seconds range decodes without error to a
  far-future instant (not a near-epoch artifact); the exact boundary and its
  strictness are not pinned.
- Interior 11-digit values decode as seconds — `Unix()` equals the input —
  because the granularity decision is on the decoded instant, not on digit
  count or magnitude. The window's edges are not pinned.
- `0` decodes to the Unix epoch — not zero-time, not an error.
- Quoted non-times, quoted digits, and a literal `null` are decode errors —
  the error is surfaced, not swallowed into zero-time.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — RFC3339 quoted strings incl. fractional seconds |
| `TestDetail02` | 2 — bare number → Unix seconds |
| `TestDetail03` | 3 — committed large values decode as milliseconds (boundary rule not pinned) |
| `TestDetail04` | 4 — edge value decodes to a far-future instant (shape; exact boundary not pinned) |
| `TestDetail05` | 5 — interior 11-digit values decode as seconds (`Unix() == input`) |
| `TestDetail06` | 6 — `0` → Unix epoch, no error |
| `TestDetail07` | 7 — `"asdf"`, `"1234"`, `null` → decode error (shape; message not pinned) |
