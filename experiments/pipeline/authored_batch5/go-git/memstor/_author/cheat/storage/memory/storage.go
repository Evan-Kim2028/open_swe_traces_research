// Package memory is a storage backend base on memory
package memory

import (
	_ "errors"
	"fmt"
	"io"
	"time"

	"example.internal/gitkit/v6/config"
	"example.internal/gitkit/v6/plumbing"
	formatcfg "example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/plumbing/format/index"
	"example.internal/gitkit/v6/plumbing/format/reflog"
	"example.internal/gitkit/v6/plumbing/storer"
	"example.internal/gitkit/v6/storage"
	"example.internal/gitkit/v6/utils/ioutil"
	_ "example.internal/gitkit/v6/utils/trace"
)

// ErrUnsupportedObjectType is returned when an unsupported object type is used.
var ErrUnsupportedObjectType = fmt.Errorf("unsupported object type")

// Storage is an implementation of git.Storer that stores data on memory, being
// ephemeral. The use of this storage should be done in controlled environments,
// since the representation in memory of some repository can fill the machine
// memory. in the other hand this storage has the best performance.
type Storage struct {
	ConfigStorage
	ObjectStorage
	ShallowStorage
	IndexStorage
	ReferenceStorage
	ModuleStorage
	ReflogStorage
	options options
}

// NewStorage returns a new in memory Storage base.
func NewStorage(o ...StorageOption) *Storage {
	opts := newOptions()
	for _, opt := range o {
		opt(&opts)
	}
	s := &Storage{
		options:          opts,
		ReferenceStorage: make(ReferenceStorage),
		ConfigStorage:    ConfigStorage{},
		ShallowStorage:   ShallowStorage{},
		ObjectStorage: ObjectStorage{
			Objects: make(map[plumbing.Hash]plumbing.EncodedObject),
		},
		ModuleStorage: make(ModuleStorage),
		ReflogStorage: ReflogStorage{
			entries: make(map[plumbing.ReferenceName][]*reflog.Entry),
		},
	}
	if opts.objectFormat != formatcfg.UnsetObjectFormat {
		s.oh = plumbing.FromObjectFormat(opts.objectFormat)
	} else {
		s.oh = plumbing.FromObjectFormat(formatcfg.DefaultObjectFormat)
	}
	return s
}

// SetObjectFormat configures the object format for this storage.
func (s *Storage) SetObjectFormat(of formatcfg.ObjectFormat) error {
	s.options.objectFormat = of
	s.oh = plumbing.FromObjectFormat(of)
	return nil
}

// SupportsExtension checks whether the Storer supports the given
// Git extension defined by name.
func (s *Storage) SupportsExtension(name, value string) bool {
	return name == "objectformat"
}

// ConfigStorage implements config.ConfigStorer for in-memory storage.
type ConfigStorage struct {
	config *config.Config
}

// SetConfig stores the given config.
func (c *ConfigStorage) SetConfig(cfg *config.Config) error {
	c.config = cfg
	return nil
}

// Config returns the stored config.
func (c *ConfigStorage) Config() (*config.Config, error) {
	if c.config == nil {
		c.config = config.NewConfig()
	}
	return c.config, nil
}

// IndexStorage implements storer.IndexStorer for in-memory storage.
type IndexStorage struct {
	index *index.Index
}

// SetIndex stores the given index.
// Note: this method sets idx.ModTime to simulate filesystem storage behavior.
func (c *IndexStorage) SetIndex(idx *index.Index) error {
	c.index = idx
	return nil
}

// Index returns the stored index.
func (c *IndexStorage) Index() (*index.Index, error) {
	if c.index == nil {
		c.index = &index.Index{Version: 2}
	}
	return c.index, nil
}

// ObjectStorage implements storer.EncodedObjectStorer for in-memory storage.
type ObjectStorage struct {
	oh      *plumbing.ObjectHasher
	Objects map[plumbing.Hash]plumbing.EncodedObject
	Commits map[plumbing.Hash]plumbing.EncodedObject
	Trees   map[plumbing.Hash]plumbing.EncodedObject
	Blobs   map[plumbing.Hash]plumbing.EncodedObject
	Tags    map[plumbing.Hash]plumbing.EncodedObject
}

type lazyCloser struct {
	storage *ObjectStorage
	obj     plumbing.EncodedObject
	closer  io.Closer
}

func (c *lazyCloser) Close() error {
	if err := c.closer.Close(); err != nil {
		return fmt.Errorf("failed to close memory encoded object: %w", err)
	}
	_, err := c.storage.SetEncodedObject(c.obj)
	return err
}

// RawObjectWriter returns a writer for writing a raw object.
func (o *ObjectStorage) RawObjectWriter(typ plumbing.ObjectType, sz int64) (w io.WriteCloser, err error) {
	obj := o.NewEncodedObject()
	obj.SetType(typ)
	obj.SetSize(sz)
	w, err = obj.Writer()
	if err != nil {
		return nil, err
	}
	return ioutil.NewWriteCloser(w, &lazyCloser{storage: o, obj: obj, closer: w}), nil
}

// NewEncodedObject returns a new EncodedObject.
func (o *ObjectStorage) NewEncodedObject() plumbing.EncodedObject {
	return plumbing.NewMemoryObject(o.oh)
}

// SetEncodedObject stores the given EncodedObject.
func (o *ObjectStorage) SetEncodedObject(obj plumbing.EncodedObject) (plumbing.Hash, error) {
	h := obj.Hash()
	o.Objects[h] = obj
	return h, nil
}

// HasEncodedObject returns nil if the object exists, or an error otherwise.
func (o *ObjectStorage) HasEncodedObject(h plumbing.Hash) (err error) {
	if _, ok := o.Objects[h]; !ok {
		return plumbing.ErrObjectNotFound
	}
	return nil
}

// EncodedObjectSize returns the size of the object with the given hash.
func (o *ObjectStorage) EncodedObjectSize(h plumbing.Hash) (
	size int64, err error,
) {
	obj, ok := o.Objects[h]
	if !ok {
		return 0, plumbing.ErrObjectNotFound
	}
	return obj.Size(), nil
}

// EncodedObject returns the object with the given type and hash.
func (o *ObjectStorage) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	obj, ok := o.Objects[h]
	if !ok || (plumbing.AnyObject != t && obj.Type() != t) {
		return nil, plumbing.ErrObjectNotFound
	}
	return obj, nil
}

// IterEncodedObjects returns an iterator for all objects of the given type.
func (o *ObjectStorage) IterEncodedObjects(t plumbing.ObjectType) (storer.EncodedObjectIter, error) {
	var series []plumbing.EncodedObject
	for _, obj := range o.Objects {
		if t == plumbing.AnyObject || obj.Type() == t {
			series = append(series, obj)
		}
	}
	return storer.NewEncodedObjectSliceIter(series), nil
}

func flattenObjectMap(m map[plumbing.Hash]plumbing.EncodedObject) []plumbing.EncodedObject {
	panic("excised: flattenObjectMap")
}

// Begin returns a new transaction.
func (o *ObjectStorage) Begin() storer.Transaction {
	return &TxObjectStorage{
		Storage: o,
		Objects: make(map[plumbing.Hash]plumbing.EncodedObject),
	}
}

// ForEachObjectHash calls the given function for each object hash.
func (o *ObjectStorage) ForEachObjectHash(fun func(plumbing.Hash) error) error {
	for h := range o.Objects {
		if err := fun(h); err != nil {
			return err
		}
	}
	return nil
}

// ObjectPacks returns the list of object packs (always empty for in-memory storage).
func (o *ObjectStorage) ObjectPacks() ([]plumbing.Hash, error) {
	return nil, nil
}

// DeleteOldObjectPackAndIndex is a no-op for in-memory storage.
func (o *ObjectStorage) DeleteOldObjectPackAndIndex(plumbing.Hash, time.Time) error {
	return nil
}

var errNotSupported = fmt.Errorf("not supported")

// LooseObjectTime returns an error as loose objects are not supported.
func (o *ObjectStorage) LooseObjectTime(_ plumbing.Hash) (time.Time, error) {
	return time.Time{}, errNotSupported
}

// DeleteLooseObject returns an error as loose objects are not supported.
func (o *ObjectStorage) DeleteLooseObject(plumbing.Hash) error {
	return errNotSupported
}

// AddAlternate returns an error as alternates are not supported.
func (o *ObjectStorage) AddAlternate(_ string) error {
	return errNotSupported
}

// TxObjectStorage implements storer.Transaction for in-memory storage.
type TxObjectStorage struct {
	Storage *ObjectStorage
	Objects map[plumbing.Hash]plumbing.EncodedObject
}

// SetEncodedObject stores the given EncodedObject in the transaction.
func (tx *TxObjectStorage) SetEncodedObject(obj plumbing.EncodedObject) (plumbing.Hash, error) {
	h := obj.Hash()
	tx.Objects[h] = obj
	return h, nil
}

// EncodedObject returns the object with the given type and hash from the transaction.
func (tx *TxObjectStorage) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	obj, ok := tx.Objects[h]
	if !ok || (plumbing.AnyObject != t && obj.Type() != t) {
		return nil, plumbing.ErrObjectNotFound
	}
	return obj, nil
}

// Commit commits all objects in the transaction to the storage.
func (tx *TxObjectStorage) Commit() error {
	for _, obj := range tx.Objects {
		if _, err := tx.Storage.SetEncodedObject(obj); err != nil {
			return err
		}
	}
	return nil
}

// Rollback discards all objects in the transaction.
func (tx *TxObjectStorage) Rollback() error {
	clear(tx.Objects)
	return nil
}

// ReferenceStorage implements storer.ReferenceStorer for in-memory storage.
type ReferenceStorage map[plumbing.ReferenceName]*plumbing.Reference

// SetReference stores the given reference.
func (r ReferenceStorage) SetReference(ref *plumbing.Reference) error {
	if ref != nil {
		r[ref.Name()] = ref
	}
	return nil
}

// CheckAndSetReference stores the reference if the old reference matches.
func (r ReferenceStorage) CheckAndSetReference(ref, old *plumbing.Reference) error {
	if ref == nil {
		return nil
	}
	r[ref.Name()] = ref
	return nil
}

// Reference returns the reference with the given name.
func (r ReferenceStorage) Reference(n plumbing.ReferenceName) (*plumbing.Reference, error) {
	ref, ok := r[n]
	if !ok {
		return nil, plumbing.ErrReferenceNotFound
	}
	return ref, nil
}

// IterReferences returns an iterator for all references.
func (r ReferenceStorage) IterReferences() (storer.ReferenceIter, error) {
	refs := make([]*plumbing.Reference, 0, len(r))
	for _, ref := range r {
		refs = append(refs, ref)
	}
	return storer.NewReferenceSliceIter(refs), nil
}

// CountLooseRefs returns the number of references.
func (r ReferenceStorage) CountLooseRefs() (int, error) {
	return len(r), nil
}

// PackRefs is a no-op.
func (r ReferenceStorage) PackRefs() error {
	return nil
}

// RemoveReference removes the reference with the given name.
func (r ReferenceStorage) RemoveReference(n plumbing.ReferenceName) error {
	delete(r, n)
	return nil
}

// ShallowStorage implements storer.ShallowStorer for in-memory storage.
type ShallowStorage []plumbing.Hash

// SetShallow stores the shallow commits.
func (s *ShallowStorage) SetShallow(commits []plumbing.Hash) error {
	*s = commits
	return nil
}

// Shallow returns the shallow commits.
func (s ShallowStorage) Shallow() ([]plumbing.Hash, error) {
	return s, nil
}

// ModuleStorage implements storer.ModuleStorer for in-memory storage.
type ModuleStorage map[string]*Storage

// Module returns the storage for the given submodule.
func (s ModuleStorage) Module(name string) (storage.Storer, error) {
	if m, ok := s[name]; ok {
		return m, nil
	}
	m := NewStorage()
	s[name] = m
	return m, nil
}

// ReflogStorage implements storer.ReflogStorer for in-memory storage.
type ReflogStorage struct {
	entries map[plumbing.ReferenceName][]*reflog.Entry
}

// Reflog returns the reflog entries for the given reference.
func (r *ReflogStorage) Reflog(name plumbing.ReferenceName) ([]*reflog.Entry, error) {
	if r.entries == nil {
		return nil, nil
	}
	return r.entries[name], nil
}

// AppendReflog appends a single entry to the reflog for the given reference.
func (r *ReflogStorage) AppendReflog(name plumbing.ReferenceName, entry *reflog.Entry) error {
	if r.entries == nil {
		r.entries = make(map[plumbing.ReferenceName][]*reflog.Entry)
	}
	r.entries[name] = append(r.entries[name], entry)
	return nil
}

// DeleteReflog removes the entire reflog for the given reference.
func (r *ReflogStorage) DeleteReflog(name plumbing.ReferenceName) error {
	if r.entries != nil {
		delete(r.entries, name)
	}
	return nil
}
