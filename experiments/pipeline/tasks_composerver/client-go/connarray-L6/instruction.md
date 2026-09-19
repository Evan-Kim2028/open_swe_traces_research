# Contract (L2) — connarray

Connections to a store address are pooled in a fixed-size array selected round-robin; Init dials each conn with the cluster security config, wraps it in a monitored conn registered with the conn monitor, and (when batching is enabled) creates the batch command machinery. Get returns the next conn. Close drains the array: marks conns closing, stops the monitor, closes each conn once; after Close, Get returns nothing usable and a new array must be dialed on next send. The monitor goroutine watches registered conns for connectivity state transitions (idle -> reconnect), removing dead ones; it starts/stops with the array. Dial failures clean up partial arrays.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestConn` | send over the pool works |
| `TestGetConnAfterClose` | post-close get yields no usable conn |
| `TestSendWhenReconnect` | reconnect after conn loss |
| `TestCancelTimeoutRetErr` | cancel/timeout errors propagate |
| `TestBatchClientRecoverAfterServerRestart` | pool recovers when the server restarts |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./internal/client/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestConnArraySendPoolProperty`, `TestConnArrayCancelTimeoutProperty`, `TestConnArrayCloseLifecycleProperty`, `TestConnArrayConcurrentSendProperty`, `TestConnArrayServerRestartProperty`: TestConnArraySendPoolProperty

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
