# Failing unit tests

The following tests currently fail on this Go codebase: `TestBackoffDeepCopy`.

Two packages ship independent copies of the same exponential wait schedule; a clone of a budget-limited retry object must still trip the budget on the next wait.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestBackoffDeepCopy (0.01s)
Error:      	Expected nil, but got: region data not ready
FAIL	github.com/tikv/client-go/v2/config/retry	0.047s
--- FAIL: TestBackoffDeepCopy (0.01s)
Error:      	Expected nil, but got: region data not ready
FAIL	github.com/tikv/client-go/v2/internal/client/retry	0.046s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
