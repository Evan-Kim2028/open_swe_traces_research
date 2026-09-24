# Contract (L2) — apiversion

- A request carrying no API-version header proceeds unchecked and reaches the
  server.
- A set version is accepted when it falls inside the client's supported
  window and rejected when it falls clearly outside; the commitment asserted
  here is interior accept / exterior reject, with boundary inclusivity left
  to the implementation's documented bounds.
- The version comparison is ordering-based on the `YYYY-MM-DD`-shaped
  strings, not date parsing: a value that is not a valid calendar date but
  orders inside the window is accepted.
- An out-of-range request fails with an error matching
  `ErrUnsupportedAPIVersion` under `errors.Is`, and the rejection happens
  before any network call.
- The guard gates every request that flows through the client's request path,
  including requests built by hand rather than through the request
  constructor.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — headerless request proceeds unchecked |
| `TestDetail02` | 2 — interior accept, exterior reject |
| `TestDetail03` | 3 — ordering, not date parsing (shape: invalid date inside window accepted) |
| `TestDetail04` | 4 — `errors.Is(err, ErrUnsupportedAPIVersion)` |
| `TestDetail05` | 5 — guard fires on a hand-built request, pre-network |
