# Exported API — consumerstate

Package `server` (module `example.internal/msgkit/v2`) — binary encodings
for JetStream consumer state and replicated stream state: the on-disk
consumer-state record, the NRG stream-snapshot frame, and the
delete-block abstractions shared by both.

Surface (store.go):
- `encodeConsumerState(state *ConsumerState) []byte` — consumer state record.
- `IsEncodedStreamState(buf []byte) bool`, `DecodeStreamState(buf []byte)
  (*StreamReplicatedState, error)` — stream snapshot frame (magic 42,
  versions plain / with-sources).
- `uvarintLen(v uint64) int`, `runLengthEncodeLen(first, num uint64) int`,
  `appendRunLength(b, first, num) []byte` — varint sizing + run-length
  delete records (magic 33).
- `DeleteRange.State/Range`, `DeleteSlice.State/Range`,
  `DeleteBlocks.NumDeleted` — the `DeleteBlock` interface impls.

Surface (filestore.go):
- `checkConsumerHeader(hdr []byte) (uint8, error)` — magic 22 + version
  gate (1 or 2).
- `decodeConsumerState(buf []byte) (*ConsumerState, error)` — inverse of
  encodeConsumerState with v1 compat.

Callers: `consumerFileStore.encodeState`/`writeState`, `EncodedState`,
consumer recovery (`readConsumerState`), `fileStore.EncodedStreamState`,
`memStore.EncodedStreamState`, NRG snapshot install, `dmap` deleted-block
readers. Constants `magic=22`, `hdrLen=2`, `seqsHdrSize`, `streamStateMagic=42`,
`streamStateVersion`, `streamStateVersionSources`, `seqSetMagic`,
`runLengthMagic=33` stay visible.
