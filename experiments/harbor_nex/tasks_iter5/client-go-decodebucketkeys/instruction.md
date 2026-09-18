# Failing unit tests

The following tests currently fail on this Go codebase: `TestCodecV2`.

Region bucket boundaries carry a keyspace prefix; decoding them must restore the original user keys, including empty start/end sentinels.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestCodecV2 (0.00s)
--- FAIL: TestCodecV2/TestDecodeBucketKeys (0.00s)
Error:      	Not equal:
Error:      	Not equal:
Error:      	Not equal:
FAIL	github.com/tikv/client-go/v2/internal/apicodec	0.012s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
