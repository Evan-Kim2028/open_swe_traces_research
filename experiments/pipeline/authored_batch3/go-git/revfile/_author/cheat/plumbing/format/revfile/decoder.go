package revfile

import (
	_ "bufio"
	"bytes"
	"crypto"
	_ "encoding/hex"
	"errors"
	_ "fmt"
	"hash"
	"io"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/utils/binary"
)

var (
	// ErrUnsupportedVersion is returned by Decode when the rev file version
	// is not supported.
	ErrUnsupportedVersion = errors.New("unsupported version")
	// ErrMalformedRevFile is returned by Decode when the rev file is corrupted.
	ErrMalformedRevFile = errors.New("malformed rev file")
	// ErrUnsupportedHashFunction is returned by Decode when the rev file defines an
	// unsupported hash function.
	ErrUnsupportedHashFunction = errors.New("unsupported hash function")
	// ErrEmptyReverseIndex is returned by Decode when the rev file is empty.
	ErrEmptyReverseIndex = errors.New("reverse index is empty")

	revHeader = []byte{'R', 'I', 'D', 'X'}
)

// Revfile constants.
const (
	VersionSupported        = 1
	sha1Hash         uint32 = 1
	sha256Hash       uint32 = 2
)

// decoder is the internal state for decoding a rev file.
// It is not exported to prevent reuse - each Decode call creates fresh state.
type decoder struct {
	reader  io.Reader
	hasher  crypto.Hash
	hash    hash.Hash
	version uint32

	objCount     int64
	packChecksum plumbing.ObjectID
	out          chan<- uint32
}

// stateFn defines each individual state within the state machine that
// represents a revfile.
type stateFn func(*decoder) (stateFn, error)

// Decode reads a rev file and sends index positions to out.
// The caller must not close out; Decode closes it when done.
// This function is safe to call concurrently with different parameters.
func Decode(r io.Reader, objCount int64, packChecksum plumbing.ObjectID, out chan<- uint32) error {
	defer close(out)
	magic := make([]byte, 4)
	if _, err := io.ReadFull(r, magic); err != nil {
		return err
	}
	if !bytes.Equal(magic, revHeader) {
		return ErrMalformedRevFile
	}
	v, err := binary.ReadUint32(r)
	if err != nil || v != VersionSupported {
		return ErrUnsupportedVersion
	}
	hf, err := binary.ReadUint32(r)
	if err != nil {
		return err
	}
	if hf != sha1Hash && hf != sha256Hash {
		return ErrUnsupportedHashFunction
	}
	var i int64
	for i = 0; i < objCount; i++ {
		idx, err := binary.ReadUint32(r)
		if err != nil {
			return err
		}
		out <- idx
	}
	return nil
}

func readMagicNumber(d *decoder) (stateFn, error) {
	return nil, nil
}

func readVersion(d *decoder) (stateFn, error) {
	return nil, nil
}

func readHashFunction(d *decoder) (stateFn, error) {
	return nil, nil
}

func readEntries(d *decoder) (stateFn, error) {
	return nil, nil
}

func readPackChecksum(d *decoder) (stateFn, error) {
	return nil, nil
}

func readRevChecksum(d *decoder) (stateFn, error) {
	return nil, nil
}
