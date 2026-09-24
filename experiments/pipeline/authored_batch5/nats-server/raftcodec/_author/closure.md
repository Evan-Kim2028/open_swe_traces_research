# Closure — raftcodec

Package `server`, file `server/raft.go`.

Removed (stubbed):
- `(*appendEntry).encode`, `decodeAppendEntry`
- `(*appendEntryResponse).encode`, `decodeAppendEntryResponse`
- `encodePeerState`, `decodePeerState`, `peerStateBufSize`
- `(*voteRequest).encode`, `decodeVoteRequest`
- `(*voteResponse).encode`, `decodeVoteResponse`
- `(*raft).encodeSnapshot`
- `termAndIndexFromSnapFile`

Retained (scaffolding): the struct defs (`appendEntry`, `Entry`,
`appendEntryResponse`, `voteRequest`, `peerState`, `snapshot`), pools and
constructors (`newAppendEntry`, `newEntry`, `newAppendEntryResponse`,
`aePool`, `peers`), constants (`idLen`, `appendEntryBaseLen`,
`appendEntryResponseLen`, `voteRequestLen`, `minSnapshotLen`,
`snapFileT`), error vars, `EntryType.String`, all raft state machinery.

Import change: `math` blanked in the excised tree (only encode used it).

Tests snipped in `server/raft_test.go`: TestNRGAppendEntryEncode,
TestNRGAppendEntryDecode, TestNRGAppendEntryDecodeTruncatedEntryLength,
TestNRGPeerStateDecodeTruncated, TestNRGMalformedAppendEntryResponse,
TestNRGVoteResponseEncoding. No test files deleted; hundreds of other raft
tests still call the codec through interceptors and will panic under the
bare excision.
