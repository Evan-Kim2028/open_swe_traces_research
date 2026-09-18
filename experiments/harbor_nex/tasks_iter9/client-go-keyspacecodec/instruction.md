# Missing behavior

API v2 multiplexes many logical keyspaces onto one cluster. User keys must be
namespaced with a 4-byte header: a mode byte (raw vs transactional) plus a
24-bit keyspace id. Region boundary keys additionally wrap that header in a
mem-comparable encoding so range order is preserved. Responses and region-error
metadata must strip the header back to user keys. RPC context must carry API
version 2 and the keyspace id, including MPP/compact payloads. Bucket split
keys on a region decode the same way as the region start/end. A malformed
region key is a fatal decode; the client must not retry it with backoff.

A no-op pass-through that leaves user keys unprefixed is wrong: RawGet of
user key `key` in keyspace 0x1092 must wire as bytes
`0x72 0x00 0x10 0x92 0x6b 0x65 0x79`, not the bare user key. Parsing a
prefixed key must yield keyspace id 0x10203, not 0xffffffff. Bucket keys
`a`, `b`, `c` must round-trip to those same user keys.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/apicodec/ ./internal/locate/
```

Implement the missing behavior so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
--- FAIL: <test> (0.00s)
Error:      	Expected nil, but got: keyspace id unavailable
Error:      	Not equal:
expected: 0x10203
actual  : 0xffffffff
Error:      	Expected nil, but got: keyspace id unavailable
Error:      	Not equal:
expected: 0x10203
actual  : 0xffffffff
--- FAIL: <test> (0.00s)
Error:      	Not equal:
expected: []byte{0x72, 0x1, 0x2, 0x3}
actual  : []byte(nil)
Error:      	Not equal:
expected: []byte{0x1, 0x2, 0x3, 0x4}
actual  : []byte{0x72, 0x1, 0x2, 0x3, 0x1, 0x2, 0x3, 0x4}
Error:      	Not equal:
expected: []byte{0x78, 0x1, 0x2, 0x3}
actual  : []byte(nil)
Error:      	Not equal:
expected: []byte{0x1, 0x2, 0x3, 0x4}
actual  : []byte{0x78, 0x1, 0x2, 0x3, 0x1, 0x2, 0x3, 0x4}
Error:      	Expected value not to be nil.
Error:      	Should be empty, but was [116 1 2 3 1 2 3 4]
--- FAIL: <test> (0.00s)
--- FAIL: <test>/<test> (0.00s)
Error:      	Not equal:
expected: [][]uint8{[]uint8{}, []uint8{0x61}, []uint8{0x62}, []uint8{0x63}, []uint8{}}
actual  : [][]uint8{[]uint8{0x72, 0x0, 0x10, 0x91, 0x61, 0x0, 0x0, 0x0, 0xfc}, []uint8{0x72, 0x0, 0x10, 0x91, 0x62, 0x0, 0x0, 0x0, 0xfc}, []uint8{0x72, 0x0, 0x10, 0x91, 0x63, 0x0, 0x0, 0x0, 0xfc}, []uint8{0x61}, []uint8{0x62}, []uint8{0x63}, []uint8{0x72, 0x0, 0x10, 0x93, 0x0, 0x0, 0x0, 0x0, 0xfb}, []uint8{0x72, 0x0, 0x10, 0x93, 0x61, 0x0, 0x0, 0x0, 0xfc}, []uint8{0x72, 0x0, 0x10, 0x93, 0x62, 0x0, 0x0, 0x0, 0xfc}, []uint8{0x72, 0x0, 0x10, 0x93, 0x63, 0x0, 0x0, 0x0, 0xfc}}
Error:      	Not equal:
expected: [][]uint8{[]uint8{}, []uint8{0x61}, []uint8{0x62}, []uint8{0x63}, []uint8{}}
actual  : [][]uint8{[]uint8{0x72, 0x0, 0x10, 0x91, 0x61, 0x0, 0x0, 0x0, 0xfc}, []uint8{0x72, 0x0, 0x10, 0x91, 0x62, 0x0, 0x0, 0x0, 0xfc}, []uint8{0x72, 0x0, 0x10, 0x91, 0x63, 0x0, 0x0, 0x0, 0xfc}, []uint8{}, []uint8{0x61}, []uint8{0x62}, []uint8{0x63}, []uint8{}}
Error:      	Not equal:
expected: [][]uint8{[]uint8{}, []uint8{0x61}, []uint8{0x62}, []uint8{0x63}, []uint8{}}
actual  : [][]uint8{[]uint8{}, []uint8{0x72, 0x0, 0x10, 0x91, 0x61, 0x0, 0x0, 0x0, 0xfc}, []uint8{0x72, 0x0, 0x10, 0x91, 0x62, 0x0, 0x0, 0x0, 0xfc}, []uint8{0x72, 0x0, 0x10, 0x91, 0x63, 0x0, 0x0, 0x0, 0xfc}, []uint8{}, []uint8{0x61}, []uint8{0x62}, []uint8{0x63}, []uint8{}}
--- FAIL: <test>/<test> (0.00s)
Error:      	Not equal:
expected: &metapb.Region{Id:0x1, StartKey:[]uint8{}, EndKey:[]uint8{}, RegionEpoch:(*metapb.RegionEpoch)(nil), Peers:[]*metapb.Peer(nil), EncryptionMeta:(*encryptionpb.EncryptionMeta)(nil), IsInFlashback:false, FlashbackStartTs:0x0, XXX_NoUnkeyedLiteral:struct {}{}, XXX_unrecognized:[]uint8(nil), XXX_sizecache}
actual  : &metapb.Region{Id:0x1, StartKey:[]uint8{0x72, 0x0, 0x10, 0x92, 0x0, 0x0, 0x0, 0x0, 0xfb}, EndKey:[]uint8{0x72, 0x0, 0x10, 0x93, 0x0, 0x0, 0x0, 0x0, 0xfb}, RegionEpoch:(*metapb.RegionEpoch)(nil), Peers:[]*metapb.Peer(nil), EncryptionMeta:(*encryptionpb.EncryptionMeta)(nil), IsInFlashback:false, FlashbackStartTs:0x0, XXX_NoUnkeyedLiteral:struct {}{}, XXX_unrecognized:[]uint8(nil), XXX_sizecache}
--- FAIL: <test>/<test> (0.00s)
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
