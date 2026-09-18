# Missing behavior

Named RPC interceptors can be bound onto a request context. When several are
attached they must all run, in link order (onion: the first attached starts
first and finishes last). A later interceptor with a duplicate name replaces
the earlier one of that name. A client that wraps the transport must invoke
that chain before the underlying send, and begin/end counts must match the
number of interceptors in the chain.

Reproduce with:

```
go test -count=1 -timeout 15m ./wirerpc/interceptor/... ./internal/client/...
```

Implement the missing behavior so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: <test> (0.00s)
Error:      	Not equal:
expected: 2
actual  : 0
Error:      	Not equal:
expected: 2
actual  : 0
Error:      	"[]" should have 2 item(s), but has 0
panic: runtime error: index out of range [0] with length 0 [recovered]
panic: runtime error: index out of range [0] with length 0
FAIL	example.internal/kvstore/v2/wirerpc/<redacted>	0.013s
--- FAIL: <test> (0.00s)
Error:      	Should be true
--- FAIL: <test> (0.00s)
Error:      	Not equal:
expected: 0
actual  : 2
Error:      	Not equal:
expected: []int{}
actual  : []int{0, 1}
Error:      	Not equal:
expected: 0
actual  : 3
Error:      	Not equal:
expected: []int{}
actual  : []int{0, 1, 2}
Error:      	Not equal:
expected: 0
actual  : 2
Error:      	Not equal:
expected: []int{}
actual  : []int{3, 4}
Error:      	Not equal:
expected: 0
actual  : 5
Error:      	Not equal:
expected: []int{}
actual  : []int{0, 1, 2, 3, 4}
Error:      	Not equal:
expected: 0
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
