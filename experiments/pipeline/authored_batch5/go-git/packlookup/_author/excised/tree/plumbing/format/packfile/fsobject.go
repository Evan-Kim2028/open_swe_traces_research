package packfile

import (
	"bufio"
	_ "errors"
	"io"
	_ "math"
	_ "os"
	stdsync "sync"

	billy "github.com/go-git/go-billy/v6"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/cache"
	"example.internal/gitkit/v6/plumbing/format/idxfile"
	_ "example.internal/gitkit/v6/utils/ioutil"
	"example.internal/gitkit/v6/utils/sync"
)

// probeSize is the byte count for the closed-FD probe. A one-byte
// ReadAt distinguishes a live descriptor from a closed one without
// mutating the file's seek cursor; a zero-length read is unusable
// because some implementations (e.g. [os.File]) return (0, nil) on
// a closed file.
const probeSize = 1

// probeBufPool returns the per-call backing array for [probePack].
// Pooling keeps the read path allocation-free on what is a very hot
// code path.
var probeBufPool = stdsync.Pool{
	New: func() any {
		var buf [probeSize]byte
		return &buf
	},
}

// probePack tests whether pack is still readable at offset by
// issuing a [probeSize]-byte [io.ReaderAt.ReadAt]. The error is
// returned verbatim so any wrapping context (path, syscall) is
// preserved for the caller; classification is left to
// [errors.Is]:
//
//   - nil means the descriptor is live; the caller may keep using
//     pack.
//   - an error matching [os.ErrClosed] means the descriptor has
//     been closed and the caller should reopen the file.
//   - any other error is propagated, matching the canonical Git
//     behaviour in `packfile.c:use_pack`, which does not retry on
//     transient I/O errors.
//
// [io.EOF] indicates the offset is at or past end-of-file, which
// implies a truncated pack — propagate rather than masking.
func probePack(pack io.ReaderAt, offset int64) error {
	panic("excised: probePack")
}

// FSObject is an object from the packfile on the filesystem.
type FSObject struct {
	hash     plumbing.Hash
	offset   int64
	size     int64
	typ      plumbing.ObjectType
	index    idxfile.Index
	fs       billy.Filesystem
	pack     billy.File
	packPath string
	cache    cache.Object
	// acquireRandom, when set, supersedes pack/packPath/fs in
	// [FSObject.Reader]: each call yields a fresh cursor that
	// Close releases.
	acquireRandom func() (RandomReader, error)
}

// NewFSObject creates a new filesystem object.
func NewFSObject(
	hash plumbing.Hash,
	finalType plumbing.ObjectType,
	offset int64,
	contentSize int64,
	index idxfile.Index,
	fs billy.Filesystem,
	pack billy.File,
	packPath string,
	cache cache.Object,
) *FSObject {
	panic("excised: NewFSObject")
}

// Reader implements the plumbing.EncodedObject interface.
//
// Reader is safe for concurrent use: it uses ReadAt (which does
// not modify the file's seek cursor) instead of Seek+Read, so
// multiple goroutines can call Reader on FSObjects that share the
// same underlying packfile handle.
func (o *FSObject) Reader() (io.ReadCloser, error) {
	panic("excised: FSObject.Reader")
}

type zlibReadCloser struct {
	r      *sync.ZLibReader
	f      io.Closer
	rbuf   *bufio.Reader
	closed bool
}

// Read reads up to len(p) bytes into p from the data.
func (r *zlibReadCloser) Read(p []byte) (int, error) {
	panic("excised: zlibReadCloser.Read")
}

func (r *zlibReadCloser) Close() (err error) {
	panic("excised: zlibReadCloser.Close")
}

// SetSize implements the plumbing.EncodedObject interface. This method
// is a noop.
func (o *FSObject) SetSize(int64) {
	panic("excised: FSObject.SetSize")
}

// SetType implements the plumbing.EncodedObject interface. This method is
// a noop.
func (o *FSObject) SetType(plumbing.ObjectType) {
	panic("excised: FSObject.SetType")
}

// Hash implements the plumbing.EncodedObject interface.
func (o *FSObject) Hash() plumbing.Hash {
	panic("excised: FSObject.Hash")
}

// Size implements the plumbing.EncodedObject interface.
func (o *FSObject) Size() int64 {
	panic("excised: FSObject.Size")
}

// Type implements the plumbing.EncodedObject interface.
func (o *FSObject) Type() plumbing.ObjectType {
	panic("excised: FSObject.Type")
}

// Writer implements the plumbing.EncodedObject interface. This method always
// returns a nil writer.
func (o *FSObject) Writer() (io.WriteCloser, error) {
	panic("excised: FSObject.Writer")
}
