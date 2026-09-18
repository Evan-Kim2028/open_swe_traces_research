# Missing behavior

When batching is enabled, many KV RPCs to the same store are packed onto one
batch stream. Waiters enqueue; a builder groups by forwarded host, skips
canceled items, and assigns monotonically increasing request ids. A timeout
on the send/recv wait must surface context cancellation vs deadline exceeded.
Unary-stream commands stay unbatched. A prewrite with a forwarded host must
share one stream so a metadata checker fires once per host, not once per
request.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/client/...
```

Implement the missing behavior so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: <test> (0.00s)
Error:      	Not equal:
expected: *errors.fundamental(batch send unavailable)
actual  : *errors.errorString(&errors.errorString{s:"context canceled"})
Error:      	Not equal:
expected: *errors.fundamental(batch send unavailable)
actual  : context.deadlineExceededError(context.deadlineExceededError{})
--- FAIL: <test> (0.00s)
Error:      	Not equal:
expected: 0x3
actual  : 0x1
Error:      	Not equal:
expected: 0x6
actual  : 0x2
Error:      	Not equal:
expected: 0x9
actual  : 0x3
Error:      	Not equal:
expected: 0xc
actual  : 0x4
--- FAIL: <test> (0.00s)
Error:      	Not equal:
expected: 0
actual  : 1
Error:      	Not equal:
expected: 0
actual  : 2
Error:      	Not equal:
expected: 0
actual  : 3
Error:      	Not equal:
expected: 0
actual  : 4
Error:      	Not equal:
expected: 0
actual  : 5
Error:      	Not equal:
expected: 0
actual  : 6
Error:      	Not equal:
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
