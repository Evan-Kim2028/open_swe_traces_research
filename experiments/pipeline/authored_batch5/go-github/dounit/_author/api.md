# Exported API — dounit

`Client.Do(req, v)` sends the request and processes the response body
into `v`:

- `v == nil`: body is discarded.
- `v` is `io.Writer`: raw body is streamed to it.
- otherwise `v` is a `*Struct`: body is JSON-decoded into it.

Callers rely on empty or whitespace-only API responses producing no
decode error.
