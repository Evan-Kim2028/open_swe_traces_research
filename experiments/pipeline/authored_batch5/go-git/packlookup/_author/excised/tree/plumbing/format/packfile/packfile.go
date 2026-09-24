package packfile

import (
	"bufio"
	_ "crypto"
	_ "fmt"
	"io"
	_ "io/fs"
	"sync"
	"sync/atomic"

	billy "github.com/go-git/go-billy/v6"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/cache"
	_ "example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/plumbing/format/idxfile"
	"example.internal/gitkit/v6/plumbing/storer"
	_ "example.internal/gitkit/v6/utils/ioutil"
	_ "example.internal/gitkit/v6/utils/sync"
)

var (
	// ErrInvalidObject is returned by Decode when an invalid object is
	// found in the packfile.
	ErrInvalidObject = NewError("invalid git object")
	// ErrZLib is returned by Decode when there was an error unzipping
	// the packfile contents.
	ErrZLib = NewError("zlib reading error")
)

// Packfile allows retrieving information from inside a packfile.
type Packfile struct {
	idxfile.Index
	fs   billy.Filesystem
	file billy.File

	// handle is the resolved PackHandle once init has run; nil
	// in legacy mode. See NewPackfile for the modes.
	handle        PackHandle
	resolveHandle PackHandleResolver

	scanReader io.ReadSeekCloser
	scanner    *Scanner

	cache cache.Object
	rbuf  *bufio.Reader

	id           plumbing.Hash
	m            sync.Mutex
	objectIDSize int

	once    sync.Once
	onceErr error

	closed atomic.Bool
}

// NewPackfile returns a packfile representation for the given .pack
// file and idx. If [WithFs] is set the packfile returns [FSObject]s;
// otherwise it returns [plumbing.MemoryObject]s.
//
// When [WithPackHandle] is supplied, the resolver owns the pack
// file descriptor and the file argument is redundant; the
// constructor closes it and [Packfile.Close] does not close the
// resolver-owned handle. Otherwise the file argument is used as-is
// and is closed by [Packfile.Close].
func NewPackfile(
	file billy.File,
	opts ...PackfileOption,
) *Packfile {
	panic("excised: NewPackfile")
}

// Get retrieves the encoded object in the packfile with the given hash.
func (p *Packfile) Get(h plumbing.Hash) (plumbing.EncodedObject, error) {
	panic("excised: Packfile.Get")
}

// GetByOffset retrieves the encoded object from the packfile at the given
// offset.
func (p *Packfile) GetByOffset(offset int64) (plumbing.EncodedObject, error) {
	panic("excised: Packfile.GetByOffset")
}

// GetSizeByOffset retrieves the size of the encoded object from the
// packfile with the given offset.
func (p *Packfile) GetSizeByOffset(offset int64) (size int64, err error) {
	panic("excised: Packfile.GetSizeByOffset")
}

// GetAll returns an iterator with all encoded objects in the packfile.
// The iterator returned is not thread-safe, it should be used in the same
// thread as the Packfile instance.
func (p *Packfile) GetAll() (storer.EncodedObjectIter, error) {
	panic("excised: Packfile.GetAll")
}

// GetByType returns all the objects of the given type.
func (p *Packfile) GetByType(typ plumbing.ObjectType) (storer.EncodedObjectIter, error) {
	panic("excised: Packfile.GetByType")
}

// Scanner returns the Packfile's inner scanner.
//
// Deprecated: this will be removed in future versions of the packfile package
// to avoid exposing the package internals and to improve its thread-safety.
// TODO: Remove Scanner method
func (p *Packfile) Scanner() (*Scanner, error) {
	panic("excised: Packfile.Scanner")
}

// ID returns the ID of the packfile, which is the checksum at the end of it.
func (p *Packfile) ID() (plumbing.Hash, error) {
	panic("excised: Packfile.ID")
}

// get is not threat-safe, and should only be called within packfile.go.
func (p *Packfile) get(h plumbing.Hash) (plumbing.EncodedObject, error) {
	panic("excised: Packfile.get")
}

// getByOffset is not threat-safe, and should only be called within packfile.go.
func (p *Packfile) getByOffset(offset int64) (plumbing.EncodedObject, error) {
	panic("excised: Packfile.getByOffset")
}

func (p *Packfile) init() error {
	panic("excised: Packfile.init")
}

func (p *Packfile) headerFromOffset(offset int64) (*ObjectHeader, error) {
	panic("excised: Packfile.headerFromOffset")
}

// Close the packfile and its resources. Subsequent calls to [Packfile.Get],
// [Packfile.GetByOffset], and the other entry points return [fs.ErrClosed].
// Close is idempotent.
func (p *Packfile) Close() error {
	panic("excised: Packfile.Close")
}

func (p *Packfile) objectFromHeader(oh *ObjectHeader) (plumbing.EncodedObject, error) {
	panic("excised: Packfile.objectFromHeader")
}

func (p *Packfile) getMemoryObject(oh *ObjectHeader) (plumbing.EncodedObject, error) {
	panic("excised: Packfile.getMemoryObject")
}
