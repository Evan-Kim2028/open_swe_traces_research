package index

import (
	"bytes"
	"errors"
	_ "fmt"
	"io"
	_ "sort"
	"time"

	"example.internal/gitkit/v6/plumbing/hash"
	"example.internal/gitkit/v6/utils/binary"
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
	var cfg options
	for _, o := range opts {
		o(&cfg)
	}
	e := &Encoder{hash: h, skipHash: cfg.skipHash}
	h.Reset()
	e.w = io.MultiWriter(w, h)
	return e
}

// Encode writes the Index to the stream of the encoder.
func (e *Encoder) Encode(idx *Index) error {
	return e.encode(idx, true)
}

func (e *Encoder) encode(idx *Index, footer bool) error {
	if err := e.encodeHeader(idx); err != nil {
		return err
	}
	if err := e.encodeEntries(idx); err != nil {
		return err
	}
	if footer {
		return e.encodeFooter()
	}
	return nil
}

func (e *Encoder) encodeHeader(idx *Index) error {
	return binary.Write(e.w, indexSignature, idx.Version, uint32(len(idx.Entries)))
}

func (e *Encoder) encodeEntries(idx *Index) error {
	for _, entry := range idx.Entries {
		if err := e.encodeEntry(idx, entry); err != nil {
			return err
		}
		wrote := entryHeaderLength + e.hash.Size() + len(entry.Name)
		if err := e.padEntry(idx, wrote); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeEntry(idx *Index, entry *Entry) error {
	sec, nsec, _ := e.timeToUint32(&entry.CreatedAt)
	msec, mnsec, _ := e.timeToUint32(&entry.ModifiedAt)
	flags := uint16(entry.Stage&0x3) << 12
	flags |= uint16(len(entry.Name)) & nameMask
	flow := []any{sec, nsec, msec, mnsec, entry.Dev, entry.Inode, entry.Mode,
		entry.UID, entry.GID, entry.Size, entry.Hash.Bytes(), flags}
	if err := binary.Write(e.w, flow...); err != nil {
		return err
	}
	return binary.Write(e.w, []byte(entry.Name))
}

func (e *Encoder) encodeEntryName(entry *Entry) error {
	return binary.Write(e.w, []byte(entry.Name))
}

func (e *Encoder) encodeEntryNameV4(entry *Entry) error {
	e.lastEntry = entry
	return binary.Write(e.w, append([]byte(entry.Name), '\x00'))
}

// commonPrefixLen returns the length of the longest common byte prefix
// between a and b.
func commonPrefixLen(a, b string) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func (e *Encoder) encodeRawExtension(signature string, data []byte) error {
	if _, err := e.w.Write([]byte(signature)); err != nil {
		return err
	}
	if err := binary.WriteUint32(e.w, uint32(len(data))); err != nil {
		return err
	}
	_, err := e.w.Write(data)
	return err
}

func (e *Encoder) timeToUint32(t *time.Time) (uint32, uint32, error) {
	return uint32(t.Unix()), uint32(t.Nanosecond()), nil
}

func (e *Encoder) padEntry(idx *Index, wrote int) error {
	padLen := 8 - wrote%8
	_, err := e.w.Write(bytes.Repeat([]byte{'\x00'}, padLen))
	return err
}

func (e *Encoder) encodeFooter() error {
	return binary.Write(e.w, e.hash.Sum(nil))
}

type byNameAndStage []*Entry

func (l byNameAndStage) Len() int      { return len(l) }
func (l byNameAndStage) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l byNameAndStage) Less(i, j int) bool {
	return l[i].Name < l[j].Name
}
