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
