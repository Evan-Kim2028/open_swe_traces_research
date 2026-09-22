# Bug report — dounit

Calls through `Client.Do` fail on endpoints that answer with an empty or
whitespace-only body: the returned error is `unexpected end of JSON
input` (or `EOF`) instead of nil.

Expected: a blank response body is a successful no-op for JSON targets —
no decode error.

Got: `Do` returns a JSON decode error for whitespace-only bodies.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
