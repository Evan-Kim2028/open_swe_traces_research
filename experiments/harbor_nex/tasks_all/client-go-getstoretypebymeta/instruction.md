# Failing unit tests

The following tests currently fail on this Go codebase: `TestRegionCache/TestRegionEpochOnTiFlash`, `TestRegionCache/TestTiFlashRecoveredFromDown`.

This client routes KV requests to the right replica of a region using store metadata published by the cluster.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
TestRegionCache/TestRegionEpochOnTiFlash panics: runtime error: invalid memory address or nil pointer dereference
  (a replica context used after a store-label update was nil)

TestRegionCache/TestTiFlashRecoveredFromDown:
  expected a non-nil replica RPC context, got nil
  then: should access peer3 after it is up
```

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
