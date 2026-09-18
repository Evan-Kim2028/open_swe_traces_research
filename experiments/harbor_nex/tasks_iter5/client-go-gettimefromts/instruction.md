# Failing unit tests

The following tests currently fail on this Go codebase: `TestRecycle`, `TestIsExpired`, `TestPdOracle_GetStaleTimestamp`.

Expiry checks and latch recycle must agree with the original instant encoded in each timestamp.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestRecycle (0.00s)
Error:      	Expected nil, but got: &latch.node{slotID, key:[]uint8{0x61}, maxCommitTS:0x682d433db2c0001, value:(*latch.Lock)(nil), next:(*latch.node)(nil)}
Error:      	Expected nil, but got: &latch.node{slotID, key:[]uint8{0x62}, maxCommitTS:0x682d433db2c0001, value:(*latch.Lock)(nil), next:(*latch.node)(nil)}
FAIL	github.com/tikv/client-go/v2/internal/latch	0.004s
--- FAIL: TestIsExpired (0.00s)
Error:      	Should be false
--- FAIL: TestPdOracle_GetStaleTimestamp (0.00s)
Error:      	Max difference between 2026-09-18 11.221867772 -0400 EDT m=-9.992866132 and 1998-05-11 03.61 -0400 EDT allowed is 2s, but difference was 248575h32m35.611867772s
FAIL	github.com/tikv/client-go/v2/oracle/oracles	0.010s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
