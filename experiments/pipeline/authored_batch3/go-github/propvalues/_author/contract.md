# Contract (L2) — propvalues

- The string accessor succeeds for the string-family value types (string,
  single-select, url) when the payload holds a string, returning the value;
  other value types and non-string payloads report failure.
- The string-slice accessor succeeds only for the multi-select value type
  with a string-slice payload. (Whether loosely-typed element slices are
  additionally tolerated is a decoding compromise the suite leaves unpinned.)
- The bool accessor succeeds only for the true-false value type when the
  payload is a string parseable as a boolean; unparseable strings report
  failure. (Whether a native bool payload is also accepted is left
  unpinned.)
- Every accessor reports `(value, false)` rather than guessing when the
  value type or payload shape does not match — including a nil payload.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — string/single-select/url accept string payloads; multi-select and non-string refuse |
| `TestDetail02` | 2 — multi-select accepts a string slice; wrong type/shape refuse (loose-slice tolerance unpinned) |
| `TestDetail03` | 3 — true-false accepts parseable strings; unparseable and wrong type refuse (bool-payload edge unpinned) |
| `TestDetail04` | 4 — nil payload → all three accessors report false |
