# Bug report

RPC tracing panics on first request, or never talks to the collector. Calls that already carry trace metadata are not wrapped. Client streams either treat a clean end-of-stream as a failure or record the same failure many times. A collector address that cannot be dialed is not reported as a setup error.

Reproduce with:

```
go test -count=1 ./grpc/middleware/xray/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
