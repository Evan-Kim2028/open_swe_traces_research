# Failing unit tests

The following tests currently fail on this Go codebase: `TestOnePC`, `TestOnePC/Test1PC`, `TestOnePC/Test1PCIsolation`, `TestOnePC/Test1PCWithMultiDC`, `TestOnePC/TestTxnCommitCounter`.

These tests are the consumer contract: a separate Go module that imports the client-go library and talks to a mock store. After requesting a faster single-round commit, the consumer still observed the two-round path; a datacenter-restricted commit that must stay on two-round commit flipped the other way. Fix the library. Do not edit, skip, or weaken the consumer tests — the verifier checksums them.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestOnePC (0.33s)
--- FAIL: TestOnePC/Test1PC (0.05s)
Error:      	Should be true
Error:      	Not equal:
expected: 0x0
Error:      	"0" is not greater than commit ts
--- FAIL: TestOnePC/Test1PCIsolation (0.05s)
Error:      	Should be true
--- FAIL: TestOnePC/Test1PCWithMultiDC (0.05s)
Error:      	Should be false
Error:      	Should be true
--- FAIL: TestOnePC/TestTxnCommitCounter (0.05s)
Error:      	Not equal:
expected: 2
actual  : 1
Error:      	Not equal:
expected: 0
actual  : 1
FAIL	integration_tests	0.369s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
The library lives at `/app`. The consumer module is `/app/integration_tests` (go.mod `replace` points at `/app`). Keep library tests passing. Do not edit consumer test files.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
