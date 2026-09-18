# Failing package tests

Tests in package `oracle/oracles` currently fail.

Reproduce with:

```
go test -count=1 -timeout 15m ./oracle/oracles/...
```

Allocating many timestamps in a tight loop must not reuse a value.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: <test> (0.04s)
Error:      	expected: 100000
actual  : 43
Error:      	should have 100000 item(s), but has 43
FAIL	github.com/tikv/client-go/v2/oracle/oracles	0.062s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
