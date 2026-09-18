# Failing unit tests

The following tests currently fail on this Go codebase: `TestRegionCache`.

A key range is half-open: the start key belongs to the interval and the end key does not. The zero-length key is the minimum key.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestRegionCache (36.94s)
--- FAIL: TestRegionCache/TestContains (0.00s)
Error:      	Should be true
Error:      	Should be true
FAIL	github.com/tikv/client-go/v2/<redacted>/<redacted>	36.955s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
