# Bug report

Incoming RPCs that already carry trace metadata lose it, or new traces appear for methods that were supposed to be ignored. A sampling rate of zero still starts traces. Downstream calls do not receive the parent span, so traces break across services. Streaming handlers see a different context than the one the interceptor set.

Reproduce with:

```
go test -count=1 ./grpc/middleware/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
