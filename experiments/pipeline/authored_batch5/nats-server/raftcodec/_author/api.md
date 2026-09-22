# Exported API — raftcodec

Package `server` (module `example.internal/msgkit/v2`) — the NRG (raft)
wire codec: binary encode/decode for appendEntry, appendEntryResponse,
voteRequest, peerState, snapshot frames, and snapshot-file name parsing.

Surface (all little-endian unless noted):
- `(ae *appendEntry) encode(b []byte) ([]byte, error)` — leader id + 4×u64
  (term, commit, pterm, pindex) + u16 entry count + per-entry
  (u32 len, type byte, data) + uvarint lterm tail; may reuse caller buf.
- `decodeAppendEntry(msg []byte, sub *subscription, reply string)
  (*appendEntry, error)` — inverse; entries borrow `msg` bytes; `ae.buf`
  retains `msg`.
- `(ar *appendEntryResponse) encode(b []byte) []byte` /
  `decodeAppendEntryResponse(msg []byte) *appendEntryResponse` — fixed
  25-byte record (u64 term, u64 index, 8-byte peer, success byte); decode
  returns nil unless len == 25.
- `encodePeerState(ps *peerState) []byte` / `decodePeerState(buf []byte)
  (*peerState, error)` / `peerStateBufSize(ps *peerState) int` — u32
  clusterSize + u32 peer count + idLen-byte peer ids + optional u16
  domainExt.
- `(vr *voteRequest) encode() []byte` / `decodeVoteRequest(msg []byte,
  reply string) *voteRequest` — fixed 32-byte record (3×u64 + candidate
  id); decode returns nil unless len == 32.
- `(n *raft) encodeSnapshot(snap *snapshot) []byte` — u64 lastTerm, u64
  lastIndex, u32 peerstate-len, peerstate, data, then 8-byte highwayhash
  checksum over all preceding bytes; nil snap → nil.
- `termAndIndexFromSnapFile(sn string) (term, index uint64, err error)` —
  parses basename "snap.<term>.<index>"; canonical round-trip required.

Callers: raft send/recv paths (`sendAppendEntry`, `handleAppendEntry`,
`sendPeerState`, vote handling, snapshot install/recovery) and the
append-entry/peer-state WAL paths. Constants `idLen=8`,
`appendEntryBaseLen=42`, `appendEntryResponseLen=25`,
`voteRequestLen=32`, `minSnapshotLen=28`, `snapFileT="snap.%d.%d"` remain
visible in source.
