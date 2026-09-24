package packp

import (
	"bytes"
	_ "fmt"
	"io"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
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
	s := pktline.NewScanner(reader)
	for s.Scan() {
		line := s.Bytes()
		if bytes.HasPrefix(line, []byte("shallow ")) {
			r.Shallows = append(r.Shallows, plumbing.NewHash(string(line[8:40])))
		} else if bytes.HasPrefix(line, []byte("unshallow ")) {
			r.Unshallows = append(r.Unshallows, plumbing.NewHash(string(line[10:50])))
		}
	}
	return nil
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
	for _, h := range r.Shallows {
		if _, err := pktline.Writef(w, "shallow %s\n", h); err != nil {
			return err
		}
	}
	for _, h := range r.Unshallows {
		if _, err := pktline.Writef(w, "unshallow %s\n", h); err != nil {
			return err
		}
	}
	return pktline.WriteFlush(w)
}
