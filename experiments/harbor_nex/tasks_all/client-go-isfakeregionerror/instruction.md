# Failing unit tests

The following tests currently fail on this Go codebase: `TestRegionRequestToThreeStores`, `TestRegionRequestToSingleStore`.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestRegionRequestToThreeStores (11.91s)
    --- FAIL: TestRegionRequestToThreeStores/TestPreferLeader (0.00s)
            	Error:      	Should be true
    --- FAIL: TestRegionRequestToThreeStores/TestSendReqFirstTimeout (1.87s)
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
    --- FAIL: TestRegionRequestToThreeStores/TestSendReqWithReplicaSelector (0.22s)
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
            	Error:      	Should be true
--- FAIL: TestRegionRequestToSingleStore (5.17s)
    --- FAIL: TestRegionRequestToSingleStore/TestKVReadTimeoutWithDisableBatchClient (0.00s)
            	Error:      	Should be true
FAIL	github.com/tikv/client-go/v2/internal/locate	17.101s
```

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
