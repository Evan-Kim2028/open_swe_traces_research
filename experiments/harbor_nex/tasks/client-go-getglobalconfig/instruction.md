# Failing unit tests

The following tests currently fail on this Go codebase: `TestPanicInRecvLoop`, `TestRecvErrorInMultipleRecvLoops`, `TestRegionCache`, `TestRawKV`.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestRawKV (48.70s)
    --- FAIL: TestRawKV/TestUpdateStoreAddr (40.09s)
            	Error:      	Expected nil, but got: region unavailable
            	Error:      	Not equal: 
FAIL	github.com/tikv/client-go/v2/rawkv	48.715s
--- FAIL: TestRegionCache (38.72s)
    --- FAIL: TestRegionCache/TestResolveStateTransition (2.01s)
            	Error:      	Not equal: 
            	Error:      	Not equal: 
            	Error:      	Not equal: 
            	Error:      	Not equal: 
            	Error:      	Not equal: 
            	Error:      	Not equal: 
            	Error:      	Not equal: 
FAIL	github.com/tikv/client-go/v2/internal/locate	38.736s
--- FAIL: TestPanicInRecvLoop (2.00s)
        	Error:      	Expected value not to be nil.
--- FAIL: TestRecvErrorInMultipleRecvLoops (0.00s)
panic: runtime error: integer divide by zero [recovered]
	panic: runtime error: integer divide by zero
FAIL	github.com/tikv/client-go/v2/internal/client	2.019s
```

Work in `/app`. Keep unrelated tests passing.
