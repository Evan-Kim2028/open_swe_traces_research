# Exported API — boolresp

`parseBoolResponse` (unexported) backs every "did it exist / did it succeed"
helper that treats 404 as plain false rather than an error.

- Signature: `func parseBoolResponse(err error) (bool, error)`.
- Consumers: predicate-style service methods across the package (e.g.
  branch-protection checks) that return `bool` plus error.
- `*ErrorResponse` with `Response.StatusCode` is the error shape produced by
  `CheckResponse` for non-2xx responses.
