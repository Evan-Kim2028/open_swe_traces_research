# Bug report

HTTP tracing panics, or traced requests never produce a collector payload. Incoming requests that already have trace metadata still look untraced. Outgoing client calls that should appear as remote children either do nothing or crash. Response status classes (too-many-requests vs client error vs server error) are not reflected on the trace. Concurrent writes to the response can panic.

Reproduce with:

```
go test -count=1 ./http/middleware/xray/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
