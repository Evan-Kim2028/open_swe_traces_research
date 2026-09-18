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

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
