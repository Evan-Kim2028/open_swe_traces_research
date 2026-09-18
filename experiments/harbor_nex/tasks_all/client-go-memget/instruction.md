# Missing behavior

The in-memory write buffer must store key/value pairs and return the latest
value for a key, or report that the key does not exist. Lookups after N
inserts must stay logarithmic in N: a full scan of every stored pair on each
lookup is too slow. On this machine, 5000 sequential lookups after 5000
inserts must stay under the ns/op ceiling recorded from the reference
implementation (gold time x 3). The performance command is:

go test -count=1 -timeout 15m -bench=^BenchmarkGet$ -benchtime=5000x ./internal/unionstore/

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/unionstore/
```

Implement the missing behavior so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: <test> (0.00s)
Error:      	Expected nil, but got: not exist
--- FAIL: <test> (0.00s)
Error:      	Expected nil, but got: not exist
Error:      	Not equal:
expected: ""
actual  : "0000000000"
Error:      	Expected nil, but got: not exist
Error:      	Not equal:
expected: ""
actual  : "0000000002"
FAIL	github.com/tikv/client-go/v2/internal/unionstore	0.022s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
