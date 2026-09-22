# Bug report

Fetch floods the connection with the client's entire history in one
burst — the server takes minutes to ACK and large fetches stall or time
out. `done` is sent before the server is ready, so packs arrive missing
common bases. Shallow fetches sometimes hang waiting for a shallow-update
that was already consumed. On a sha256 server a fresh clone errors with
"mismatched algorithms" instead of adopting the format.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
