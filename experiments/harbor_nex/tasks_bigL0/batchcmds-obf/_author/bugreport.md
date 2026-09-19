# Bug report

When batching was on I sent several writes to the same store, some with a
forwarded host. Each write opened its own stream: a metadata checker on
the server fired once per request (got 3, 6, 9, 12) instead of once per
host (expected 1, 2, 3, 4).

A send whose wait was already canceled returned `batch send unavailable`.
expected `context canceled`. A send with timeout 0 returned the same
generic error. expected deadline exceeded.

A bundle of ten unforwarded writes built as if the queue were empty:
expected 10 packed entries, actual 0 (or the length helper stayed 0
after each enqueue: expected 1..10, actual 0).

Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./internal/client/...`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
