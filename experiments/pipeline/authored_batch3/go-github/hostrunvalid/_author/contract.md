# Contract (L2) — hostrunvalid

- A hosted-runner create request with an empty `Name` is rejected before any
  network call.
- A request with a zero-value `Image` is rejected before any network call.
- A request with an empty `Size` is rejected before any network call.
- A request with a zero `RunnerGroupID` is rejected before any network call.
- When all four required fields are present the request is accepted and the
  call reaches the API — on both the organization and enterprise entry
  points.
- Each rejection surfaces an error that names the offending field; the
  literal message wording is implementation detail.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — empty `Name` errors, no request sent |
| `TestDetail02` | 2 — zero `Image` errors, no request sent |
| `TestDetail03` | 3 — empty `Size` errors, no request sent |
| `TestDetail04` | 4 — zero `RunnerGroupID` errors, no request sent |
| `TestDetail05` | 5 — valid request reaches the server on both entry points |
| `TestDetail06` | 6 — each rejection's error names the offending field (shape; literals not pinned) |
