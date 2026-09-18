# Failing package tests

Tests in package `oracle/oracles` currently fail.

Reproduce with:

```
go test -count=1 -timeout 15m ./oracle/oracles/...
```

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: <test> (0.00s)
Error:      	Not equal:
expected: -4
actual  : -1787954760551553
Error:      	Not equal:
expected: 4
actual  : -1787954760551545
FAIL	github.com/tikv/client-go/v2/oracle/oracles	0.011s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
