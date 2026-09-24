package packp

import (
	_ "bytes"
	_ "fmt"
	"io"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/plumbing/format/pktline"
)

const (
	shallowLineLen   = 48
	unshallowLineLen = 50
)

// ShallowUpdate represents shallow/unshallow updates during fetch.
type ShallowUpdate struct {
	Shallows   []plumbing.Hash
	Unshallows []plumbing.Hash
}

// Decode parses shallow update information from the reader.
func (r *ShallowUpdate) Decode(reader io.Reader) error {
	panic("excised: ShallowUpdate.Decode")
}

func (r *ShallowUpdate) decodeShallowLine(line []byte) error {
	panic("excised: ShallowUpdate.decodeShallowLine")
}

func (r *ShallowUpdate) decodeUnshallowLine(line []byte) error {
	panic("excised: ShallowUpdate.decodeUnshallowLine")
}

func (r *ShallowUpdate) decodeLine(line, prefix []byte, expLen int) (plumbing.Hash, error) {
	panic("excised: ShallowUpdate.decodeLine")
}

// Encode writes the shallow update to the writer.
func (r *ShallowUpdate) Encode(w io.Writer) error {
	panic("excised: ShallowUpdate.Encode")
}
