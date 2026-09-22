// Package ioutil implements some I/O utility functions.
package ioutil

import (
	_ "bufio"
	"context"
	"errors"
	"io"
)

// Peeker is an interface for types that can peek at the next bytes.
type Peeker interface {
	Peek(int) ([]byte, error)
}

// ReadPeeker is an interface that groups the basic Read and Peek methods.
type ReadPeeker interface {
	io.Reader
	Peeker
}

// ErrEmptyReader is returned when a reader is empty.
var ErrEmptyReader = errors.New("reader is empty")

// NonEmptyReader takes a reader and returns it if it is not empty, or
// `ErrEmptyReader` if it is empty. If there is an error when reading the first
// byte of the given reader, it will be propagated.
func NonEmptyReader(r io.Reader) (io.Reader, error) {
	panic("excised: NonEmptyReader")
}

type readCloser struct {
	io.Reader
	closer io.Closer
}

func (r *readCloser) Close() error {
	panic("excised: readCloser.Close")
}

// NewReadCloser creates an `io.ReadCloser` with the given `io.Reader` and
// `io.Closer`.
func NewReadCloser(r io.Reader, c io.Closer) io.ReadCloser {
	panic("excised: NewReadCloser")
}

type readCloserCloser struct {
	io.ReadCloser
	closer func() error
}

func (r *readCloserCloser) Close() (err error) {
	panic("excised: readCloserCloser.Close")
}

// NewReadCloserWithCloser creates an `io.ReadCloser` with the given `io.ReaderCloser` and
// `io.Closer` that ensures that the closer is closed on close
func NewReadCloserWithCloser(r io.ReadCloser, c func() error) io.ReadCloser {
	panic("excised: NewReadCloserWithCloser")
}

type writeCloser struct {
	io.Writer
	closer io.Closer
}

func (r *writeCloser) Close() error {
	panic("excised: writeCloser.Close")
}

// NewWriteCloser creates an `io.WriteCloser` with the given `io.Writer` and
// `io.Closer`.
func NewWriteCloser(w io.Writer, c io.Closer) io.WriteCloser {
	panic("excised: NewWriteCloser")
}

type writeNopCloser struct {
	io.Writer
}

func (writeNopCloser) Close() error {
	panic("excised: writeNopCloser.Close")
}

// WriteNopCloser returns a WriteCloser with a no-op Close method wrapping
// the provided Writer w.
func WriteNopCloser(w io.Writer) io.WriteCloser {
	panic("excised: WriteNopCloser")
}

type readerAtAsReader struct {
	io.ReaderAt
	offset int64
}

func (r *readerAtAsReader) Read(bs []byte) (int, error) {
	panic("excised: readerAtAsReader.Read")
}

// NewReaderUsingReaderAt returns a new io.Reader from an io.ReaderAt starting at the given offset.
func NewReaderUsingReaderAt(r io.ReaderAt, offset int64) io.Reader {
	panic("excised: NewReaderUsingReaderAt")
}

// CheckClose calls Close on the given io.Closer. If the given *error points to
// nil, it will be assigned the error returned by Close. Otherwise, any error
// returned by Close will be ignored. CheckClose is usually called with defer.
func CheckClose(c io.Closer, err *error) {
	panic("excised: CheckClose")
}

// NewContextWriteCloser as NewContextWriter but with io.Closer interface.
func NewContextWriteCloser(ctx context.Context, w io.WriteCloser) io.WriteCloser {
	panic("excised: NewContextWriteCloser")
}

// NewContextReadCloser as NewContextReader but with io.Closer interface.
func NewContextReadCloser(ctx context.Context, r io.ReadCloser) io.ReadCloser {
	panic("excised: NewContextReadCloser")
}

type readerOnError struct {
	io.Reader
	notify func(error)
}

// NewReaderOnError returns a io.Reader that call the notify function when an
// unexpected (!io.EOF) error happens, after call Read function.
func NewReaderOnError(r io.Reader, notify func(error)) io.Reader {
	panic("excised: NewReaderOnError")
}

// NewReadCloserOnError returns a io.ReadCloser that call the notify function
// when an unexpected (!io.EOF) error happens, after call Read function.
func NewReadCloserOnError(r io.ReadCloser, notify func(error)) io.ReadCloser {
	panic("excised: NewReadCloserOnError")
}

func (r *readerOnError) Read(buf []byte) (n int, err error) {
	panic("excised: readerOnError.Read")
}

type writerOnError struct {
	io.Writer
	notify func(error)
}

// NewWriterOnError returns a io.Writer that call the notify function when an
// unexpected (!io.EOF) error happens, after call Write function.
func NewWriterOnError(w io.Writer, notify func(error)) io.Writer {
	panic("excised: NewWriterOnError")
}

// NewWriteCloserOnError returns a io.WriteCloser that call the notify function
// when an unexpected (!io.EOF) error happens, after call Write function.
func NewWriteCloserOnError(w io.WriteCloser, notify func(error)) io.WriteCloser {
	panic("excised: NewWriteCloserOnError")
}

func (r *writerOnError) Write(p []byte) (n int, err error) {
	panic("excised: writerOnError.Write")
}

// CloserFunc implements the io.Closer interface with a function.
type CloserFunc func() error

var _ io.Closer = CloserFunc(nil)

// Close calls the function.
func (f CloserFunc) Close() error {
	panic("excised: CloserFunc.Close")
}
