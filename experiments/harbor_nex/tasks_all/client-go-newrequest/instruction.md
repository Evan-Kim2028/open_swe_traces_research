# Failing unit tests

The following tests currently fail on this Go codebase: `TestRegionRequestToThreeStores`.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestRegionRequestToThreeStores (15.12s)
    --- FAIL: TestRegionRequestToThreeStores/TestLoadBasedReplicaRead (1.69s)
            	Error:      	Not equal: 
            	Error:      	Should not be: 0x4
            	Error:      	Object expected to be of type *locate.tryIdleReplica, but was *locate.accessKnownLeader
    --- FAIL: TestRegionRequestToThreeStores/TestReplicaReadWithFlashbackInProgress (0.00s)
            	Error:      	"0" is not greater than or equal to "1"
FAIL	github.com/tikv/client-go/v2/internal/locate	15.139s
```

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
