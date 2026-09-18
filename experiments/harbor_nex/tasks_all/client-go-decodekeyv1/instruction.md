# Failing unit tests

The following tests currently fail on this Go codebase: `TestDecodeKey`.

The client splits on-wire keys into a keyspace prefix and the caller-supplied user key, depending on API version.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
TestDecodeKey (API v1 case): expected user key bytes 74 01 02 03 01 02 03 04, got 01 02 03 01 02 03 04
(the first byte of the original key is missing; no compile error)
```

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
