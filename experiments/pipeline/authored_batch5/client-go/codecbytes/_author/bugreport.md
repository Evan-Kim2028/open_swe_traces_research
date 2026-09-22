# Bug report

The memcomparable byte-string codec in `util/codec` panics on every call.
Keys that must sort lexically cannot be encoded, and decoding existing
encodings fails.

Expected: `EncodeBytes(nil, nil)` returns `00 00 00 00 00 00 00 00 f7`;
`EncodeBytes(nil, [1 2 3])` returns `01 02 03 00 00 00 00 00 fa`;
`EncodeBytes(nil, [1 2 3 0])` returns `01 02 03 00 00 00 00 00 fb`;
`EncodeBytes(nil, [1..8])` returns `01 02 03 04 05 06 07 08 ff` followed by
eight zero bytes and `f7`. `DecodeBytes` of each encoding returns the
original bytes plus the unconsumed suffix; malformed markers, non-zero
padding, and truncated input return errors instead of panicking.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
