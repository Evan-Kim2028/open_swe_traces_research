# Missing behavior

This library multiplexes many logical keyspaces onto one cluster. User keys
must be namespaced with a 4-byte header: a mode byte (`r` = 0x72 for raw,
`x` = 0x78 for transactional) plus a 24-bit keyspace id in big-endian. The
exclusive end of a keyspace is that 32-bit (mode||id) value plus one, with
carry across bytes; the last raw id (0xFFFFFF) wraps the mode byte from `r`
to `s`. A keyspace id that does not fit in 24 bits, or an unknown mode, is
an error at construction.

Region boundary keys additionally wrap that header in a mem-comparable
encoding so range order is preserved. Responses and region-error metadata
must strip the header (and the mem-comparable wrap) back to user keys.
RPC context must carry API version 2 and the keyspace id, including MPP
and compact payloads. Bucket split keys on a region decode the same way as
the region start/end: keys from the previous keyspace clip to an empty
user start, keys from the next keyspace clip to an empty user end, and
keys inside this keyspace drop the 4-byte header.

Parsing a prefixed key must yield the 24-bit id; the all-ones id 0xffffffff
is reserved for “not a v2 key”. API v1 keys are identity (no header). A
malformed region key is a fatal decode; the client must not retry it with
backoff.

These rules are properties of the exported constructors and the request,
response, key, range, region, and bucket entry points callers already use.
They must hold for arbitrary keyspace ids and keys, not only the worked
examples:

- Encoding a user key or range and then decoding it returns the original
  bytes. The same round-trip holds for region-boundary keys.
- If user key A is byte-wise less than B, both encodings of A are
  byte-wise less than those of B (order is preserved inside a keyspace).
- Region lists on epoch-not-match responses keep only the intersection
  with this keyspace, in the original order, and drop regions wholly
  outside it. A region covering the whole keyspace becomes empty start
  and empty end.
- Encoding a request prefixes its keys without mutating the caller’s
  original; decoding the matching response restores user keys.

Worked cases:

- Raw get of user key `key` in keyspace 0x1092 must wire as bytes
  `0x72 0x00 0x10 0x92 0x6b 0x65 0x79`, not the bare user key.
- Parsing a txn or raw header `.. 0x01 0x02 0x03` must yield keyspace
  0x10203, not 0xffffffff; a mode byte `t` is invalid.
- Splitting a v2 key `r 1 2 3 | 1 2 3 4` yields header `r 1 2 3` and user
  `1 2 3 4`; the same bytes under v1 are returned unchanged; v2 with mode
  `t` errors.
- Empty/partial user ranges in keyspace 0x1092 expand to
  `[0x72 0x00 0x10 0x92, 0x72 0x00 0x10 0x93)`.
- Epoch-not-match regions that cover the whole keyspace decode to empty
  start and empty end; a region wholly before or after the keyspace is
  dropped; a region overlapping the interior keeps only the overlap, as
  user keys.
- Bucket keys `a`, `b`, `c` mixed with previous/next keyspace encoded
  neighbors round-trip to those same user keys plus empty sentinels.
- An MPP dispatch in keyspace 4242 must advertise that id, API v2, and
  encoded ranges.
- A store-safe-ts request (no keys) still encodes without error on the
  v1 transactional codec.

A no-op pass-through that leaves user keys unprefixed is wrong: expected
prefixed wire bytes, actual bare user keys. expected keyspace id 0x10203,
actual 0xffffffff.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/apicodec/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestCodecKeyRangeRoundTrip`, `TestCodecClipProperties`, `TestCodecContractExamples`, `TestCodecUnmentionedRandom`: Property tests (seed 20260918, 10k cases): key/range round-trip, byte-order, region/epoch/bucket clip, request/response symmetry, contract examples, and unseen random inputs.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
