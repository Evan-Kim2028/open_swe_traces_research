# Exported API — revfile

Package `plumbing/format/revfile` (module `example.internal/gitkit/v6`) — pack reverse-index
(`.rev`) codec: RIDX magic, u32 version, u32 hash-function id, N u32 entries, pack checksum,
file checksum.

`Decode(r io.Reader, objCount int64, packChecksum plumbing.ObjectID, out chan<- uint32) error`
— streams index positions into `out` (decoder closes it). `Encode(w io.Writer, h hash.Hash,
idx *idxfile.MemoryIndex) error`. Consts `VersionSupported`, errors `ErrUnsupportedVersion`,
`ErrMalformedRevFile`, `ErrUnsupportedHashFunction`, `ErrEmptyReverseIndex` kept.

Callers: packfile reader/writer building `.rev` companions for `.idx`. In-tree tests
removed: 3.
