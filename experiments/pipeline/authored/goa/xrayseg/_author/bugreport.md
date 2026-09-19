# Bug report

Opening a trace panics. Child operations have empty ids or do not point at their parent. A still-running operation either never appears at the collector or is sent repeatedly, and finishing it does not replace the pending copy. The collector never receives a single datagram per update.

Reproduce with:

```
go test -count=1 ./middleware/xray/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
