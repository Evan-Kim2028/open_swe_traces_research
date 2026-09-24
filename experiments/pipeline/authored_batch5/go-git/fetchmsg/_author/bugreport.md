# Bug report

The v2 fetch wire codec is broken both ways. Outgoing fetch arguments
reach the server in a different order than git expects — hash lines are
not canonicalised — and a request with nothing to fetch still goes out
instead of failing early. Incoming responses are accepted too loosely:
metadata sections arrive out of order or repeated without complaint, a
response that promised a packfile and then ended early is treated as
success, an acknowledgment that said "ready" but is followed by the wrong
terminator slips through, and "NAK" shows up on the wire alongside real
ACK lines where the server never sent both.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
