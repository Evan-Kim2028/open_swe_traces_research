# Failing unit tests

The following tests currently fail on this Go codebase: `TestParseKeyspaceID`, `TestCodecV2`.

Keyspace identifiers occupy three bytes after a mode prefix; encoding a key, parsing the identifier, and the exclusive end of that identifier's range must agree on the same byte layout.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: TestParseKeyspaceID (0.00s)
Error:      	Not equal:
Error:      	Not equal:
--- FAIL: TestCodecV2 (0.00s)
Error:      	Not equal:
--- FAIL: TestCodecV2/TestDecodeBucketKeys (0.00s)
Error:      	Not equal:
--- FAIL: TestCodecV2/TestEncodeRequest (0.00s)
Error:      	Not equal:
--- FAIL: TestCodecV2/TestEncodeV2KeyRanges (0.00s)
Error:      	Not equal:
--- FAIL: TestCodecV2/TestNewCodecV2 (0.00s)
Error:      	Not equal:
FAIL	github.com/tikv/client-go/v2/internal/apicodec	0.014s
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
