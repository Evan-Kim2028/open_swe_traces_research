package packhandle

import (
	"errors"
	"io"
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
	f, err := sf.Acquire()
	if err != nil {
		return nil, err
	}
	return &cursorReader{sf: sf, file: f, size: size}, nil
}

func (c *cursorReader) Read(p []byte) (int, error) {
	if c.offset >= c.size {
		return 0, io.EOF
	}
	n, err := c.file.ReadAt(p, c.offset)
	c.offset += int64(n)
	return n, err
}

func (c *cursorReader) ReadAt(p []byte, off int64) (int, error) {
	return c.file.ReadAt(p, off)
}

func (c *cursorReader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = c.offset + offset
	case io.SeekEnd:
		abs = c.size + offset
	default:
		return 0, ErrInvalidSeekWhence
	}
	c.offset = abs
	return abs, nil
}

// Close releases the underlying [sharedfile.SharedFile] reference. Idempotent.
func (c *cursorReader) Close() error {
	c.sf.Release()
	return nil
}

var (
	_ PackReader   = (*cursorReader)(nil)
	_ RandomReader = (*cursorReader)(nil)
)
