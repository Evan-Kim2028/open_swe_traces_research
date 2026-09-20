// Package binary implements syntax-sugar functions on top of the standard
// library binary package
package binary

import (
	"bufio"
	_ "bytes"
	_ "encoding/binary"
	"errors"
	"io"
	_ "math"
	"sync"
)

// ErrIntegerOverflow is returned when a Git-format variable-width integer
// would not fit into an int64 because the input declares more continuation
// bytes than the type can hold.
var ErrIntegerOverflow = errors.New("variable-width integer overflow")

// Read reads structured binary data from r into data. Bytes are read and
// decoded in BigEndian order
// https://golang.org/pkg/encoding/binary/#Read
func Read(r io.Reader, data ...any) error {
	panic("excised: Read")
}

// ReadUntil reads from r untin delim is found
func ReadUntil(r io.Reader, delim byte) ([]byte, error) {
	panic("excised: ReadUntil")
}

// ReadUntilFromBufioReader is like bufio.ReadBytes but drops the delimiter
// from the result.
func ReadUntilFromBufioReader(r *bufio.Reader, delim byte) ([]byte, error) {
	panic("excised: ReadUntilFromBufioReader")
}

// ReadVariableWidthInt reads and returns an int in Git VLQ special format:
//
// Ordinary VLQ has some redundancies, example:  the number 358 can be
// encoded as the 2-octet VLQ 0x8166 or the 3-octet VLQ 0x808166 or the
// 4-octet VLQ 0x80808166 and so forth.
//
// To avoid these redundancies, the VLQ format used in Git removes this
// prepending redundancy and extends the representable range of shorter
// VLQs by adding an offset to VLQs of 2 or more octets in such a way
// that the lowest possible value for such an (N+1)-octet VLQ becomes
// exactly one more than the maximum possible value for an N-octet VLQ.
// In particular, since a 1-octet VLQ can store a maximum value of 127,
// the minimum 2-octet VLQ (0x8000) is assigned the value 128 instead of
// 0. Conversely, the maximum value of such a 2-octet VLQ (0xff7f) is
// 16511 instead of just 16383. Similarly, the minimum 3-octet VLQ
// (0x808000) has a value of 16512 instead of zero, which means
// that the maximum 3-octet VLQ (0xffff7f) is 2113663 instead of
// just 2097151.  And so forth.
//
// This is how the offset is saved in C:
//
//	dheader[pos] = ofs & 127;
//	while (ofs >>= 7)
//	    dheader[--pos] = 128 | (--ofs & 127);
func ReadVariableWidthInt(r io.Reader) (int64, error) {
	panic("excised: ReadVariableWidthInt")
}

const (
	maskContinue = uint8(128) // 1000 000
	maskLength   = uint8(127) // 0111 1111
	lengthBits   = uint8(7)   // subsequent bytes has 7 bits to store the length
)

// ReadUint64 reads 8 bytes and returns them as a BigEndian uint32
func ReadUint64(r io.Reader) (uint64, error) {
	panic("excised: ReadUint64")
}

// ReadUint32 reads 4 bytes and returns them as a BigEndian uint32
func ReadUint32(r io.Reader) (uint32, error) {
	panic("excised: ReadUint32")
}

// ReadUint16 reads 2 bytes and returns them as a BigEndian uint16
func ReadUint16(r io.Reader) (uint16, error) {
	panic("excised: ReadUint16")
}

const sniffLen = 8000

// sniffPool reuses sniff-window buffers across IsBinary calls so the hot diff
// path (one call per file, per side) does not allocate one per invocation.
var sniffPool = sync.Pool{
	New: func() any {
		b := make([]byte, sniffLen)
		return &b
	},
}

// IsBinary detects if data is a binary value based on:
// http://git.kernel.org/cgit/git/git.git/tree/xdiff-interface.c?id=HEAD#n198
func IsBinary(r io.Reader) (bool, error) {
	panic("excised: IsBinary")
}
