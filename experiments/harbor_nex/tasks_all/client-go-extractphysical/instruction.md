# Failing unit tests

The following tests currently fail on this Go codebase: `TestIsExpired`, `TestLocalOracle_UntilExpired`, `TestPDOracle_UntilExpired`, `TestPdOracle_GetStaleTimestamp`, `TestNonFutureStaleTSO`.

A timestamp packs a wall-clock millisecond together with a per-millisecond counter; turning a timestamp back into a time, and using that same split to decide whether a lock has expired, must agree.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestIsExpired (0.00s)
Error:      	Should be true
--- FAIL: TestLocalOracle_UntilExpired (0.00s)
Error:      	Not equal:
expected: -4
actual  : 1789744684008
Error:      	Not equal:
expected: 4
actual  : 1789744684016
--- FAIL: TestPDOracle_UntilExpired (0.00s)
Error:      	Not equal:
expected: 25
actual  : 35
--- FAIL: TestPdOracle_GetStaleTimestamp (0.00s)
Error:      	Max difference between 2026-09-18 11.012411359 -0400 EDT m=-9.963907089 and 2083-06-06 02.024 -0400 EDT allowed is 2s, but difference was -497151h17m54.011588641s
--- FAIL: TestNonFutureStaleTSO (0.01s)
Error:      	"2083-06-06 02.952 -0400 EDT" is not less than "2026-09-18 11.981020399 -0400 EDT m=+1.004701971"
FAIL	github.com/tikv/client-go/v2/<redacted>/oracles	1.005s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
