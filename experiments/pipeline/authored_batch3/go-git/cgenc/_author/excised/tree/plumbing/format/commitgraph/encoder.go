package commitgraph

import (
	"crypto"
	_ "fmt"
	"io"
	_ "math"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/hash"
	_ "example.internal/gitkit/v6/utils/binary"
)

// Encoder writes MemoryIndex structs to an output stream.
type Encoder struct {
	io.Writer
	hash hash.Hash
}

// NewEncoder returns a new stream encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	// TODO: Support passing an ObjectFormat (sha256)
	h := hash.New(crypto.SHA1)
	mw := io.MultiWriter(w, h)
	return &Encoder{mw, h}
}

// Encode writes an index into the commit-graph file
func (e *Encoder) Encode(idx Index) error {
	panic("excised: Encoder.Encode")
}

// lookupParentIndex resolves a parent hash to its position in the file being
// encoded. A bare map read yields index 0 for an absent hash, which would
// silently record an arbitrary commit as the parent, so report it instead.
func lookupParentIndex(hashToIndex map[plumbing.Hash]uint32, h plumbing.Hash) (uint32, error) {
	panic("excised: lookupParentIndex")
}

func (e *Encoder) prepare(idx Index, hashes []plumbing.Hash) (hashToIndex map[plumbing.Hash]uint32, fanout []uint32, extraEdgesCount, generationV2OverflowCount uint32, err error) {
	panic("excised: Encoder.prepare")
}

func (e *Encoder) encodeFileHeader(chunkCount int) (err error) {
	panic("excised: Encoder.encodeFileHeader")
}

func (e *Encoder) encodeChunkHeaders(chunkSignatures [][]byte, chunkSizes []uint64) (err error) {
	panic("excised: Encoder.encodeChunkHeaders")
}

func (e *Encoder) encodeFanout(fanout []uint32) (err error) {
	panic("excised: Encoder.encodeFanout")
}

func (e *Encoder) encodeOidLookup(hashes []plumbing.Hash) (err error) {
	panic("excised: Encoder.encodeOidLookup")
}

func (e *Encoder) encodeCommitData(hashes []plumbing.Hash, hashToIndex map[plumbing.Hash]uint32, idx Index) (extraEdges []uint32, generationV2Data []uint64, err error) {
	panic("excised: Encoder.encodeCommitData")
}

func (e *Encoder) encodeExtraEdges(extraEdges []uint32) (err error) {
	panic("excised: Encoder.encodeExtraEdges")
}

func (e *Encoder) encodeGenerationV2Data(generationV2Data []uint64) (overflows []uint64, err error) {
	panic("excised: Encoder.encodeGenerationV2Data")
}

func (e *Encoder) encodeGenerationV2Overflow(overflows []uint64) (err error) {
	panic("excised: Encoder.encodeGenerationV2Overflow")
}

func (e *Encoder) encodeChecksum() error {
	panic("excised: Encoder.encodeChecksum")
}
