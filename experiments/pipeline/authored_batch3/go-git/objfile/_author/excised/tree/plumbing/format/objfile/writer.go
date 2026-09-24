package objfile

import (
	"errors"
	"io"
	_ "strconv"

	"example.internal/gitkit/v6/plumbing"
	format "example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/utils/sync"
)

// ErrOverflow is returned when the declared data length is exceeded.
var ErrOverflow = errors.New("objfile: declared data length exceeded (overflow)")

// Writer writes and encodes data in compressed objfile format to a provided
// io.Writer. Close should be called when finished with the Writer. Close will
// not close the underlying io.Writer.
type Writer struct {
	raw          io.Writer
	hasher       plumbing.Hasher
	multi        io.Writer
	zlib         sync.ZlibWriter
	objectFormat format.ObjectFormat

	closed  bool
	pending int64 // number of unwritten bytes

	closeErr error
}

// NewWriter returns a new Writer writing to w and hashing objects with the
// given object format.
//
// The returned Writer implements io.WriteCloser. Close should be called when
// finished with the Writer. Close will not close the underlying io.Writer.
func NewWriter(w io.Writer, objectFormat format.ObjectFormat) *Writer {
	zlib := sync.GetZlibWriter(w)
	return &Writer{
		raw:          w,
		zlib:         zlib,
		objectFormat: objectFormat,
	}
}

// WriteHeader writes the type and the size and prepares to accept the
// object's contents. If an invalid t is provided, plumbing.ErrInvalidType
// is returned. If a negative size is provided, ErrNegativeSize is
// returned. If the encoded header exceeds maxHeaderLen,
// ErrHeaderTooLong is returned, mirroring the reader's bound.
func (w *Writer) WriteHeader(t plumbing.ObjectType, size int64) error {
	panic("excised: Writer.WriteHeader")
}

func (w *Writer) writeHeader(t plumbing.ObjectType, typeBytes []byte, size int64) error {
	panic("excised: Writer.writeHeader")
}

func (w *Writer) prepareForWrite(t plumbing.ObjectType, size int64) {
	panic("excised: Writer.prepareForWrite")
}

// Write writes the object's contents. Write returns the error ErrOverflow if
// more than size bytes are written after WriteHeader.
func (w *Writer) Write(p []byte) (n int, err error) {
	panic("excised: Writer.Write")
}

// Hash returns the hash of the object data stream that has been written so far.
// It can be called before or after Close.
func (w *Writer) Hash() plumbing.Hash {
	panic("excised: Writer.Hash")
}

// Close releases any resources consumed by the Writer.
//
// Calling Close does not close the wrapped io.Writer originally passed to
// NewWriter.
//
// It returns an error, if any. Close will return the same error if called
// multiple times.
func (w *Writer) Close() error {
	panic("excised: Writer.Close")
}
