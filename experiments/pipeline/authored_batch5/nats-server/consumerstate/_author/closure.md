# Closure — consumerstate

Package `server`, files `server/store.go` + `server/filestore.go`.

Removed (stubbed), store.go:
- `uvarintLen`, `runLengthEncodeLen`, `appendRunLength`
- `DeleteRange.State`, `DeleteRange.Range`
- `DeleteSlice.State`, `DeleteSlice.Range`, `DeleteBlocks.NumDeleted`
- `IsEncodedStreamState`, `DecodeStreamState`
- `encodeConsumerState`

Removed (stubbed), filestore.go:
- `checkConsumerHeader`, `decodeConsumerState`

Retained: `ConsumerState`/`Pending`/`SequencePair`/`StreamReplicatedState`/
`StreamSourceState`/`DeleteBlock`/`DeleteRange`/`DeleteSlice` types, all
magic/version constants, `errCorruptState`, `ErrBadStreamStateEncoding`,
`ErrCorruptStreamState`, `avl`/`gsl` packages, the store methods that call
the codec (`encodeState`, `EncodedState`, `EncodedStreamState`), and
`decodeSourcesState` (fileStore method that mutates fs.sources — out of
scope).

Import changes: `encoding/binary`, `math/bits`, `server/avl` blanked in
store.go (all three were used only by this closure). filestore.go imports
unchanged.

Tests snipped: `TestFileStoreConsumerEncodeDecodeRedelivered`,
`TestFileStoreConsumerEncodeDecodePendingBelowStreamAckFloor`,
`TestFileStoreBadConsumerState`,
`TestFileStoreEncodedStreamStateWithSources` (server/filestore_test.go);
`TestNoRaceEncodeConsumerStateBug` (server/norace_1_test.go). No test
files deleted.
