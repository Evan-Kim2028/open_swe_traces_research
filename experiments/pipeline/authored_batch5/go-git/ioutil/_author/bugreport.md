# Bug report

Cancelling a fetch context corrupts the transport: a `Read` that returns
`context.Canceled` still has a goroutine blocked in the underlying
network read, and closing the connection races it — sometimes the next
read on the same connection returns the cancelled read's bytes. In-flight
writes report cancellation before the write completes, so closing the
writer afterwards races the still-running write. An underlying panic
crashes the process instead of surfacing as an error. Empty pack streams
are accepted rather than reported as empty.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
