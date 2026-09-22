# Bug report

The memcomparable integer codec in `util/codec` panics on every call. Keys
that must sort lexically in numeric order cannot be encoded or decoded.

Expected: `EncodeInt(nil, 1)` returns `80 00 00 00 00 00 00 01` and
`EncodeIntDesc(nil, 1)` returns `7f ff ff ff ff ff ff fe`;
`EncodeUvarint(nil, 300)` returns `ac 02`; `EncodeVarint(nil, 1)` returns
`02` and `EncodeVarint(nil, -1)` returns `01`;
`EncodeComparableUvarint(nil, 5)` returns `0d`, `(nil, 240)` returns
`f8 f0`, and `(nil, 300)` returns `f9 01 2c`; `EncodeComparableVarint(nil,
-1)` returns `07 ff` and `(nil, -300)` returns `06 fe d4`. Decoding each
encoding returns the original value plus the unconsumed suffix; short or
malformed inputs return descriptive errors instead of panicking.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
