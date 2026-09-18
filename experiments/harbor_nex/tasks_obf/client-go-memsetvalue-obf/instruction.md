# Incorrect behavior

After a key is stored, a second write of that key with a same-length payload is ignored; later reads still return the first payload.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/unionstore/...
```

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: <test> (0.30s)
Error:      	Not equal:
expected: 0xc
actual  : 0x78
FAIL	example.internal/kvstore/v2/internal/unionstore	0.309s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
