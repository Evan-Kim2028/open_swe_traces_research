# Incorrect behavior

A lock issued just now with a 200ms TTL is reported expired after only 10ms (expected still live). A timestamp generated for ten seconds ago decodes near the Unix epoch (1969) instead of about ten seconds before now.

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
Error:      	Should be false
--- FAIL: <test> (0.00s)
Error:      	Max difference between 2026-09-18 11.776801871 -0400 EDT m=-9.992481235 and 1969-12-31 19.744495776 -0500 EST allowed is 2s, but difference was 497150h45m6.032306095s
FAIL	github.com/tikv/client-go/v2/oracle/oracles	0.011s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
