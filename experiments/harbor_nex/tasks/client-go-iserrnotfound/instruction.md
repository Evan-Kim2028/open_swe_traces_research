# Failing unit tests

The following tests currently fail on this Go codebase: `TestPipelinedFlushGet`, `TestUnionStoreGetSet`, `TestUnionStoreDelete`, `TestBufferBatchGetter`.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestBufferBatchGetter (0.00s)
        	Error:      	Received unexpected error:
        	Error:      	Not equal: 
        	Error:      	Not equal: 
        	Error:      	Not equal: 
        	Error:      	Not equal: 
FAIL	github.com/tikv/client-go/v2/txnkv/transaction	0.017s
--- FAIL: TestPipelinedFlushGet (0.02s)
        	Error:      	Should be true
--- FAIL: TestUnionStoreGetSet (0.00s)
        	Error:      	Expected nil, but got: not exist
        	Error:      	Not equal: 
        	Error:      	Not equal: 
--- FAIL: TestUnionStoreDelete (0.00s)
        	Error:      	Not equal: 
FAIL	github.com/tikv/client-go/v2/internal/unionstore	0.038s
```

Work in `/app`. Keep unrelated tests passing.
