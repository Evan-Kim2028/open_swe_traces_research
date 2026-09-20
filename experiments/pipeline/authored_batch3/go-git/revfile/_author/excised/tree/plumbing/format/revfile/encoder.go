package revfile

import (
	_ "crypto"
	_ "fmt"
	"hash"
	"io"
	_ "reflect"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/idxfile"
	_ "example.internal/gitkit/v6/utils/binary"
)

// encoder is the internal state for encoding a rev file.
// It is not exported to prevent reuse - each Encode call creates fresh state.
type encoder struct {
	writer io.Writer
	hash   hash.Hash

	entries      []uint32
	packChecksum plumbing.Hash
}

// stateFnEncode defines each individual state within the state machine that
// represents encoding a revfile.
type stateFnEncode func(*encoder) (stateFnEncode, error)

// Encode encodes a reverse index from a MemoryIndex to the writer.
// The reverse index maps pack offsets (sorted order) to index positions.
// This function is safe to call concurrently with different parameters.
func Encode(w io.Writer, h hash.Hash, idx *idxfile.MemoryIndex) error {
	panic("excised: Encode")
}

// buildReverseIndex creates the reverse index mapping from the MemoryIndex.
// It maps from pack offset order to index position (sorted by hash).
func (e *encoder) buildReverseIndex(idx *idxfile.MemoryIndex) error {
	panic("excised: encoder.buildReverseIndex")
}

func writeHeader(e *encoder) (stateFnEncode, error) {
	panic("excised: writeHeader")
}

func writeVersion(e *encoder) (stateFnEncode, error) {
	panic("excised: writeVersion")
}

func writeHashFunction(e *encoder) (stateFnEncode, error) {
	panic("excised: writeHashFunction")
}

func writeEntries(e *encoder) (stateFnEncode, error) {
	panic("excised: writeEntries")
}

func writePackChecksum(e *encoder) (stateFnEncode, error) {
	panic("excised: writePackChecksum")
}

func writeRevChecksum(e *encoder) (stateFnEncode, error) {
	panic("excised: writeRevChecksum")
}
