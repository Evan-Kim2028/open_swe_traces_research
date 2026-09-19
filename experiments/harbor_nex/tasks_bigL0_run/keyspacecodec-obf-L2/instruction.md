# Contract (L2) — keyspacecodec-obf

Prose only. No implementation names. This is the full behavioral contract the hidden verifier must cover. Worked examples are the ones a cheat may hardcode; anything beyond them is what makes A3 bite.

## Behavior

This library multiplexes many logical keyspaces onto one cluster.

User keys are namespaced with a 4-byte header: a mode byte (`r` = 0x72 raw, `x` = 0x78 transactional) plus a 24-bit keyspace id in big-endian. The exclusive end of a keyspace is that 32-bit (mode||id) value plus one, with carry across bytes. The last raw id (0xFFFFFF) wraps the mode byte from `r` to `s`. A keyspace id that does not fit in 24 bits, or an unknown mode, is an error at construction.

Region boundary keys wrap that header in a mem-comparable encoding so range order is preserved. Responses and region-error metadata strip the header (and the mem-comparable wrap) back to user keys.

RPC context carries API version 2 and the keyspace id, including MPP and compact payloads. Encoding clones the request first; retries reuse the original.

Bucket split keys on a region decode the same way as the region start/end: keys from the previous keyspace clip to an empty user start, keys from the next keyspace clip to an empty user end, keys inside this keyspace drop the 4-byte header. Empty sentinels at the clipped bounds stay empty.

Parsing a prefixed key yields the 24-bit id. The all-ones id 0xffffffff is reserved for “not a v2 key”. API v1 keys are identity (no header). A header whose mode byte is neither raw nor txn, or that is shorter than 4 bytes, errors and yields 0xffffffff.

A malformed (truncated / non mem-comparable) region key is a fatal decode: callers must not retry it with backoff.

A request type with no key payload (store safe-ts) still encodes without error under the transactional v1 codec.

Empty/partial user ranges expand to the keyspace interval: empty start becomes the keyspace prefix, empty end becomes the next keyspace prefix; both empty becomes the whole keyspace.

Epoch-not-match region lists are clipped to the keyspace and decoded to user keys: a region covering the whole keyspace becomes empty/empty, a region wholly outside the keyspace is dropped, a region overlapping the keyspace is truncated to the overlap then stripped of the header.

## Worked examples (cheat may hardcode these)

- Raw get of user key `key` in keyspace 0x1092 wires as `0x72 0x00 0x10 0x92 0x6b 0x65 0x79`, not the bare user key.
- Parsing a txn or raw header `.. 0x01 0x02 0x03` yields keyspace 0x10203, not 0xffffffff. Mode byte `t` is invalid.
- Splitting a v2 key `r 1 2 3 | 1 2 3 4` yields header `r 1 2 3` and user `1 2 3 4`. The same bytes under v1 are unchanged. v2 with mode `t` errors with empty results.
- Empty/partial user ranges in keyspace 0x1092 expand to `[0x72 0x00 0x10 0x92, 0x72 0x00 0x10 0x93)`.
- Bucket keys `a`, `b`, `c` mixed with previous/next keyspace encoded neighbors round-trip to those same user keys plus empty sentinels.
- An MPP dispatch in keyspace 4242 advertises that id, API v2, and encoded ranges for `a`/`b`.
- Epoch-not-match: a region covering the whole keyspace → empty/empty; wholly before/after → dropped; interior overlap keeps only the overlap as user keys.

A no-op pass-through that leaves user keys unprefixed is wrong: expected prefixed wire bytes, actual bare user keys. expected keyspace id 0x10203, actual 0xffffffff.

## Coverage of original in-tree tests

Map from the original tests (not to be copied as the hidden verifier) to the contract sentence they encode. Hidden tests must restate these via the exported API only.

| original test | contract sentence |
|---|---|
| `TestCodecV2/TestEncodeRequest` | A raw single-key lookup of user key `key` in keyspace 0x1092 must be sent with the 4-byte raw-mode header `0x72 0x00 0x10 0x92` followed by the user key, not the bare user key. Encoding twice on the same original request yields the same prefixed bytes (the original is not overwritten). |
| `TestCodecV2/TestEncodeV2KeyRanges` | User ranges whose start or end is empty expand to the keyspace interval: empty start becomes the keyspace prefix, empty end becomes the next keyspace prefix; both empty becomes the whole keyspace; interior start/end keep their user suffix under the same prefix. |
| `TestCodecV2/TestNewCodecV2` | Constructing a v2 codec rejects a keyspace id that does not fit in 24 bits and rejects an unknown mode; the prefix is a mode byte plus the 24-bit id; the exclusive end prefix is that 32-bit value plus one, with carry across bytes; the last raw id wraps the mode byte from `r` to `s`. |
| `TestCodecV2/TestDecodeEpochNotMatch` | Epoch-not-match region lists are clipped to the keyspace and decoded to user keys: a region covering the whole keyspace becomes empty/empty, a region wholly outside the keyspace is dropped, and a region overlapping the keyspace is truncated to the overlap then stripped of the header. |
| `TestCodecV2/TestGetKeyspaceID` | A codec built for keyspace 4242 reports that same id. |
| `TestCodecV2/TestEncodeMPPRequest` | An MPP dispatch must carry keyspace id 4242 and API version 2 on its task meta, and its coprocessor ranges must be encoded the same way as ordinary user keys. |
| `TestCodecV2/TestDecodeBucketKeys` | Bucket split keys that mix the previous, current, and next keyspace, including empty sentinels, decode to the user keys in this keyspace (`a`,`b`,`c`) with empty sentinels at the clipped bounds — never the mem-comparable encoded forms. |
| `TestParseKeyspaceID` | A 4-byte header starting with txn `x` or raw `r` followed by 0x010203 yields keyspace id 0x010203; a header whose mode byte is neither, or that is shorter than 4 bytes, errors and yields the all-ones null id 0xffffffff. |
| `TestDecodeKey` | API v2 splits a well-formed key into a 4-byte header and the remaining user bytes; API v1 is identity (no header); an invalid v2 mode byte errors with empty results. |
| `TestEncodeUnknownRequest` | A request type with no key payload (store safe-ts) still encodes without error under the transactional v1 codec. |
| `TestRegionCache/TestNoBackoffWhenFailToDecodeRegion` | A truncated or otherwise non mem-comparable region key is a fatal decode: locating that region must fail without incrementing the backoff counter. |

Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./internal/apicodec/ ./internal/locate/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
