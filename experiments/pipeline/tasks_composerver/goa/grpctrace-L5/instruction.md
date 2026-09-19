# Contract (L2) — grpctrace

A server interceptor reads incoming metadata. If a trace id is already present it is reused and a new span id is minted; a parent span id from metadata is kept. If no trace id is present, tracing starts only when the full method does not match a discard pattern and the sampler accepts; otherwise the context is left untraced. Sampling percent 0 never starts a new trace but still continues an incoming one. Stream servers wrap the stream so later messages see the same context. A client interceptor that sees a trace id on the context writes it to outgoing metadata as trace-id and writes the current span id as parent-span-id; without a trace id it does not add metadata. Option helpers forward id functions, sampling percent, max sampling rate, sample size, and discard patterns to the shared tracing options.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestUnaryServerTrace` | new vs continued vs discarded vs zero-rate traces on unary RPCs |
| `TestStreamServerTrace` | the same rules on streaming RPCs, with the wrapped stream context |
| `TestUnaryClientTrace` | outgoing metadata carries trace id and parent span when the context is traced |
| `TestStreamClientTrace` | the same outgoing metadata on a stream dial |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./grpc/middleware/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestGrpctraceUnaryContinueTrace`, `TestGrpctraceUnaryNewTrace`, `TestGrpctraceUnaryDiscardRandom`, `TestGrpctraceZeroRateProperty`, `TestGrpctraceZeroRateRandom`, `TestGrpctraceStreamServerContext`, `TestGrpctraceStreamContextRandom`, `TestGrpctraceUnaryClientMetadata`, `TestGrpctraceClientMetadataRandom`, `TestGrpctraceClientNoTrace`, `TestGrpctraceClientNoTraceRandom`, `TestGrpctraceStreamContinueTraceRandom`, `TestGrpctraceUnseenRandomProperty`: TestGrpctraceUnaryContinueTrace

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
