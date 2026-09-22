package packhandle

import (
	"errors"
	_ "io"
	_ "io/fs"
	"sync/atomic"

	"example.internal/gitkit/v6/internal/sharedfile"
)

// ErrInvalidSeekWhence is returned by [cursorReader.Seek] when
// whence is not one of [io.SeekStart], [io.SeekCurrent], or
// [io.SeekEnd].
var ErrInvalidSeekWhence = errors.New("packhandle: invalid whence")

// ErrNegativeSeekPosition is returned by [cursorReader.Seek]
// when the resolved absolute offset would be negative.
var ErrNegativeSeekPosition = errors.New("packhandle: negative seek position")

// cursorReader is the concrete reader returned by both
// [PackHandle.OpenPackReader] and [PackHandle.OpenRandomReader].
// Each cursor holds its own offset and one [sharedfile.SharedFile]
// reference that Close releases.
//
// Read and Seek mutate the cursor offset and are not safe to call
// concurrently on the same cursor. ReadAt is safe to call
// concurrently with itself.
type cursorReader struct {
	sf     *sharedfile.SharedFile
	file   ReadAtCloser
	size   int64
	offset int64
	closed atomic.Bool
}

func newCursorReader(sf *sharedfile.SharedFile, size int64) (*cursorReader, error) {
	panic("excised: newCursorReader")
}

func (c *cursorReader) Read(p []byte) (int, error) {
	panic("excised: cursorReader.Read")
}

func (c *cursorReader) ReadAt(p []byte, off int64) (int, error) {
	panic("excised: cursorReader.ReadAt")
}

func (c *cursorReader) Seek(offset int64, whence int) (int64, error) {
	panic("excised: cursorReader.Seek")
}

// Close releases the underlying [sharedfile.SharedFile] reference. Idempotent.
func (c *cursorReader) Close() error {
	panic("excised: cursorReader.Close")
}

var (
	_ PackReader   = (*cursorReader)(nil)
	_ RandomReader = (*cursorReader)(nil)
)
