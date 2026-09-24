package binary

import (
	_ "encoding/binary"
	"io"
)

// Write writes the binary representation of data into w, using BigEndian order
// https://golang.org/pkg/encoding/binary/#Write
func Write(w io.Writer, data ...any) error {
	panic("excised: Write")
}

// WriteVariableWidthInt writes a variable width encoded int64 to w.
func WriteVariableWidthInt(w io.Writer, n int64) error {
	panic("excised: WriteVariableWidthInt")
}

// WriteUint64 writes the binary representation of a uint64 into w, in BigEndian
// order
func WriteUint64(w io.Writer, value uint64) error {
	panic("excised: WriteUint64")
}

// WriteUint32 writes the binary representation of a uint32 into w, in BigEndian
// order
func WriteUint32(w io.Writer, value uint32) error {
	panic("excised: WriteUint32")
}

// WriteUint16 writes the binary representation of a uint16 into w, in BigEndian
// order
func WriteUint16(w io.Writer, value uint16) error {
	panic("excised: WriteUint16")
}
