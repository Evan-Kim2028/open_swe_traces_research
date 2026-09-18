# Failing unit tests

The following tests currently fail on this Go codebase: `TestParseKeyspaceID`, `TestCodecV2` (including `TestCodecV2/TestNewCodecV2`, `TestCodecV2/TestEncodeRequest`, `TestCodecV2/TestEncodeV2KeyRanges`, `TestCodecV2/TestDecodeBucketKeys`, `TestCodecV2/TestDecodeEpochNotMatch`).

API v2 keys carry a keyspace identifier so one cluster can host several isolated keyspaces.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
TestParseKeyspaceID: expected keyspace id 0x10203, got 0x10202

TestCodecV2 setup / TestNewCodecV2: expected keyspace prefix bytes 72 00 10 92, got 72 00 10 93
TestCodecV2/TestEncodeRequest: expected user key prefixed with 72 00 10 92, got the neighboring prefix 72 00 10 93
```

Work in `/app`. Keep unrelated tests passing.
