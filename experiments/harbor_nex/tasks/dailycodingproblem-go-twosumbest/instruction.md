# Failing unit tests

The following tests currently fail on this Go codebase: `TestTwoSumsSmall`, `TestTwoSumsLarge`.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestTwoSumsSmall (0.00s)
    --- FAIL: TestTwoSumsSmall/Best (0.00s)
--- FAIL: TestTwoSumsLarge (0.00s)
    --- FAIL: TestTwoSumsLarge/Best (0.00s)
FAIL	dailycodingproblem-go/day1	0.003s
```

Work in `/app`. Keep unrelated tests passing.
