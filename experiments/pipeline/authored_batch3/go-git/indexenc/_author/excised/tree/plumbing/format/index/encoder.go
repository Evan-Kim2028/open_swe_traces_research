package index

import (
	_ "bytes"
	"errors"
	_ "fmt"
	"io"
	_ "sort"
	"time"

	"example.internal/gitkit/v6/plumbing/hash"
	_ "example.internal/gitkit/v6/utils/binary"
)

var (
	// EncodeVersionSupported is the range of supported index versions
	EncodeVersionSupported uint32 = 4

	// ErrInvalidTimestamp is returned by Encode if a Index with a Entry with
	// negative timestamp values
	ErrInvalidTimestamp = errors.New("negative timestamps are not allowed")
)

// An Encoder writes an Index to an output stream.
type Encoder struct {
	w         io.Writer
	hash      hash.Hash
	lastEntry *Entry
	skipHash  bool
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer, h hash.Hash, opts ...Option) *Encoder {
	panic("excised: NewEncoder")
}

// Encode writes the Index to the stream of the encoder.
func (e *Encoder) Encode(idx *Index) error {
	panic("excised: Encoder.Encode")
}

func (e *Encoder) encode(idx *Index, footer bool) error {
	panic("excised: Encoder.encode")
}

func (e *Encoder) encodeHeader(idx *Index) error {
	panic("excised: Encoder.encodeHeader")
}

func (e *Encoder) encodeEntries(idx *Index) error {
	panic("excised: Encoder.encodeEntries")
}

func (e *Encoder) encodeEntry(idx *Index, entry *Entry) error {
	panic("excised: Encoder.encodeEntry")
}

func (e *Encoder) encodeEntryName(entry *Entry) error {
	panic("excised: Encoder.encodeEntryName")
}

func (e *Encoder) encodeEntryNameV4(entry *Entry) error {
	panic("excised: Encoder.encodeEntryNameV4")
}

// commonPrefixLen returns the length of the longest common byte prefix
// between a and b.
func commonPrefixLen(a, b string) int {
	panic("excised: commonPrefixLen")
}

func (e *Encoder) encodeRawExtension(signature string, data []byte) error {
	panic("excised: Encoder.encodeRawExtension")
}

func (e *Encoder) timeToUint32(t *time.Time) (uint32, uint32, error) {
	panic("excised: Encoder.timeToUint32")
}

func (e *Encoder) padEntry(idx *Index, wrote int) error {
	panic("excised: Encoder.padEntry")
}

func (e *Encoder) encodeFooter() error {
	panic("excised: Encoder.encodeFooter")
}

type byNameAndStage []*Entry

func (l byNameAndStage) Len() int      { return len(l) }
func (l byNameAndStage) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l byNameAndStage) Less(i, j int) bool {
	panic("excised: byNameAndStage.Less")
}
