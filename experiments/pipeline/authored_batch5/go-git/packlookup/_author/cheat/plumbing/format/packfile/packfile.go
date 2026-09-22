package packfile

import (
	"bufio"
	"crypto"
	"fmt"
	"io"
	"io/fs"
	"sync"
	"sync/atomic"

	billy "github.com/go-git/go-billy/v6"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/cache"
	format "example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/plumbing/format/idxfile"
	"example.internal/gitkit/v6/plumbing/storer"
	"example.internal/gitkit/v6/utils/ioutil"
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
	p := &Packfile{
		file:         file,
		objectIDSize: crypto.SHA1.Size(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Get retrieves the encoded object in the packfile with the given hash.
func (p *Packfile) Get(h plumbing.Hash) (plumbing.EncodedObject, error) {
	if p.closed.Load() {
		return nil, fs.ErrClosed
	}
	if err := p.init(); err != nil {
		return nil, err
	}
	p.m.Lock()
	defer p.m.Unlock()
	return p.get(h)
}

// GetByOffset retrieves the encoded object from the packfile at the given
// offset.
func (p *Packfile) GetByOffset(offset int64) (plumbing.EncodedObject, error) {
	if p.closed.Load() {
		return nil, fs.ErrClosed
	}
	if err := p.init(); err != nil {
		return nil, err
	}
	p.m.Lock()
	defer p.m.Unlock()
	return p.getByOffset(offset)
}

// GetSizeByOffset retrieves the size of the encoded object from the
// packfile with the given offset.
func (p *Packfile) GetSizeByOffset(offset int64) (size int64, err error) {
	if p.closed.Load() {
		return 0, fs.ErrClosed
	}
	if err := p.init(); err != nil {
		return 0, err
	}
	d, err := p.GetByOffset(offset)
	if err != nil {
		return 0, err
	}
	return d.Size(), nil
}

// GetAll returns an iterator with all encoded objects in the packfile.
// The iterator returned is not thread-safe, it should be used in the same
// thread as the Packfile instance.
func (p *Packfile) GetAll() (storer.EncodedObjectIter, error) {
	return p.GetByType(plumbing.AnyObject)
}

// GetByType returns all the objects of the given type.
func (p *Packfile) GetByType(typ plumbing.ObjectType) (storer.EncodedObjectIter, error) {
	if p.closed.Load() {
		return nil, fs.ErrClosed
	}
	if err := p.init(); err != nil {
		return nil, err
	}
	switch typ {
	case plumbing.AnyObject,
		plumbing.BlobObject,
		plumbing.TreeObject,
		plumbing.CommitObject,
		plumbing.TagObject:
		entries, err := p.EntriesByOffset()
		if err != nil {
			return nil, err
		}
		return &objectIter{p: p, iter: entries, typ: typ}, nil
	default:
		return nil, plumbing.ErrInvalidType
	}
}

// Scanner returns the Packfile's inner scanner.
//
// Deprecated: this will be removed in future versions of the packfile package
// to avoid exposing the package internals and to improve its thread-safety.
// TODO: Remove Scanner method
func (p *Packfile) Scanner() (*Scanner, error) {
	if p.closed.Load() {
		return nil, fs.ErrClosed
	}
	if err := p.init(); err != nil {
		return nil, err
	}
	return p.scanner, nil
}

// ID returns the ID of the packfile, which is the checksum at the end of it.
func (p *Packfile) ID() (plumbing.Hash, error) {
	if err := p.init(); err != nil {
		return plumbing.ZeroHash, err
	}
	return p.id, nil
}

// get is not threat-safe, and should only be called within packfile.go.
func (p *Packfile) get(h plumbing.Hash) (plumbing.EncodedObject, error) {
	if obj, ok := p.cache.Get(h); ok {
		return obj, nil
	}
	offset, err := p.FindOffset(h)
	if err != nil {
		return nil, err
	}
	oh, err := p.headerFromOffset(offset)
	if err != nil {
		return nil, err
	}
	return p.objectFromHeader(oh)
}

// getByOffset is not threat-safe, and should only be called within packfile.go.
func (p *Packfile) getByOffset(offset int64) (plumbing.EncodedObject, error) {
	oh, err := p.headerFromOffset(offset)
	if err != nil {
		return nil, err
	}
	return p.objectFromHeader(oh)
}

func (p *Packfile) init() error {
	p.once.Do(func() {
		if p.file == nil {
			p.onceErr = fmt.Errorf("file is not set")
			return
		}
		if p.Index == nil {
			p.onceErr = fmt.Errorf("index is not set")
			return
		}
		opts := []ScannerOption{}
		if p.objectIDSize == format.SHA256Size {
			opts = append(opts, WithSHA256())
		}
		p.scanner = NewScanner(p.file, opts...)
		if !p.scanner.Scan() {
			p.onceErr = p.scanner.Error()
			return
		}
		if _, err := p.scanner.Seek(-int64(p.objectIDSize), io.SeekEnd); err != nil {
			p.onceErr = err
			return
		}
		p.id.ResetBySize(p.objectIDSize)
		if _, err := p.id.ReadFrom(p.scanner); err != nil {
			p.onceErr = err
		}
		if p.cache == nil {
			p.cache = cache.NewObjectLRUDefault()
		}
	})
	return p.onceErr
}

func (p *Packfile) headerFromOffset(offset int64) (*ObjectHeader, error) {
	if err := p.scanner.SeekFromStart(offset); err != nil {
		return nil, err
	}
	if !p.scanner.Scan() {
		if err := p.scanner.Error(); err != nil {
			return nil, err
		}
		return nil, plumbing.ErrObjectNotFound
	}
	oh := p.scanner.Data().Value().(ObjectHeader)
	return &oh, nil
}

// Close the packfile and its resources. Subsequent calls to [Packfile.Get],
// [Packfile.GetByOffset], and the other entry points return [fs.ErrClosed].
// Close is idempotent.
func (p *Packfile) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	closer, ok := p.file.(io.Closer)
	if !ok {
		return nil
	}
	return closer.Close()
}

func (p *Packfile) objectFromHeader(oh *ObjectHeader) (plumbing.EncodedObject, error) {
	if oh == nil {
		return nil, plumbing.ErrObjectNotFound
	}
	return p.getMemoryObject(oh)
}

func (p *Packfile) getMemoryObject(oh *ObjectHeader) (plumbing.EncodedObject, error) {
	of := format.SHA1
	if p.objectIDSize == format.SHA256.Size() {
		of = format.SHA256
	}
	h := plumbing.FromObjectFormat(of)
	obj := plumbing.NewMemoryObject(h)
	obj.SetSize(oh.Size)
	obj.SetType(oh.Type)

	w, err := obj.Writer()
	if err != nil {
		return nil, err
	}
	defer ioutil.CheckClose(w, &err)

	switch oh.Type {
	case plumbing.CommitObject, plumbing.TreeObject, plumbing.BlobObject, plumbing.TagObject:
		err = p.scanner.inflateContent(oh.ContentOffset, w, oh.Size)
	default:
		err = ErrInvalidObject.AddDetails("type %q", oh.Type)
	}
	if err != nil {
		return nil, err
	}
	p.cache.Put(obj)
	return obj, nil
}
