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
	_ "example.internal/gitkit/v6/utils/ioutil"
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
	panic("excised: NewStorage")
}

// SetObjectFormat configures the object format for this storage.
func (s *Storage) SetObjectFormat(of formatcfg.ObjectFormat) error {
	panic("excised: Storage.SetObjectFormat")
}

// SupportsExtension checks whether the Storer supports the given
// Git extension defined by name.
func (s *Storage) SupportsExtension(name, value string) bool {
	panic("excised: Storage.SupportsExtension")
}

// ConfigStorage implements config.ConfigStorer for in-memory storage.
type ConfigStorage struct {
	config *config.Config
}

// SetConfig stores the given config.
func (c *ConfigStorage) SetConfig(cfg *config.Config) error {
	panic("excised: ConfigStorage.SetConfig")
}

// Config returns the stored config.
func (c *ConfigStorage) Config() (*config.Config, error) {
	panic("excised: ConfigStorage.Config")
}

// IndexStorage implements storer.IndexStorer for in-memory storage.
type IndexStorage struct {
	index *index.Index
}

// SetIndex stores the given index.
// Note: this method sets idx.ModTime to simulate filesystem storage behavior.
func (c *IndexStorage) SetIndex(idx *index.Index) error {
	panic("excised: IndexStorage.SetIndex")
}

// Index returns the stored index.
func (c *IndexStorage) Index() (*index.Index, error) {
	panic("excised: IndexStorage.Index")
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
	panic("excised: lazyCloser.Close")
}

// RawObjectWriter returns a writer for writing a raw object.
func (o *ObjectStorage) RawObjectWriter(typ plumbing.ObjectType, sz int64) (w io.WriteCloser, err error) {
	panic("excised: ObjectStorage.RawObjectWriter")
}

// NewEncodedObject returns a new EncodedObject.
func (o *ObjectStorage) NewEncodedObject() plumbing.EncodedObject {
	panic("excised: ObjectStorage.NewEncodedObject")
}

// SetEncodedObject stores the given EncodedObject.
func (o *ObjectStorage) SetEncodedObject(obj plumbing.EncodedObject) (plumbing.Hash, error) {
	panic("excised: ObjectStorage.SetEncodedObject")
}

// HasEncodedObject returns nil if the object exists, or an error otherwise.
func (o *ObjectStorage) HasEncodedObject(h plumbing.Hash) (err error) {
	panic("excised: ObjectStorage.HasEncodedObject")
}

// EncodedObjectSize returns the size of the object with the given hash.
func (o *ObjectStorage) EncodedObjectSize(h plumbing.Hash) (
	size int64, err error,
) {
	panic("excised: ObjectStorage.EncodedObjectSize")
}

// EncodedObject returns the object with the given type and hash.
func (o *ObjectStorage) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	panic("excised: ObjectStorage.EncodedObject")
}

// IterEncodedObjects returns an iterator for all objects of the given type.
func (o *ObjectStorage) IterEncodedObjects(t plumbing.ObjectType) (storer.EncodedObjectIter, error) {
	panic("excised: ObjectStorage.IterEncodedObjects")
}

func flattenObjectMap(m map[plumbing.Hash]plumbing.EncodedObject) []plumbing.EncodedObject {
	panic("excised: flattenObjectMap")
}

// Begin returns a new transaction.
func (o *ObjectStorage) Begin() storer.Transaction {
	panic("excised: ObjectStorage.Begin")
}

// ForEachObjectHash calls the given function for each object hash.
func (o *ObjectStorage) ForEachObjectHash(fun func(plumbing.Hash) error) error {
	panic("excised: ObjectStorage.ForEachObjectHash")
}

// ObjectPacks returns the list of object packs (always empty for in-memory storage).
func (o *ObjectStorage) ObjectPacks() ([]plumbing.Hash, error) {
	panic("excised: ObjectStorage.ObjectPacks")
}

// DeleteOldObjectPackAndIndex is a no-op for in-memory storage.
func (o *ObjectStorage) DeleteOldObjectPackAndIndex(plumbing.Hash, time.Time) error {
	panic("excised: ObjectStorage.DeleteOldObjectPackAndIndex")
}

var errNotSupported = fmt.Errorf("not supported")

// LooseObjectTime returns an error as loose objects are not supported.
func (o *ObjectStorage) LooseObjectTime(_ plumbing.Hash) (time.Time, error) {
	panic("excised: ObjectStorage.LooseObjectTime")
}

// DeleteLooseObject returns an error as loose objects are not supported.
func (o *ObjectStorage) DeleteLooseObject(plumbing.Hash) error {
	panic("excised: ObjectStorage.DeleteLooseObject")
}

// AddAlternate returns an error as alternates are not supported.
func (o *ObjectStorage) AddAlternate(_ string) error {
	panic("excised: ObjectStorage.AddAlternate")
}

// TxObjectStorage implements storer.Transaction for in-memory storage.
type TxObjectStorage struct {
	Storage *ObjectStorage
	Objects map[plumbing.Hash]plumbing.EncodedObject
}

// SetEncodedObject stores the given EncodedObject in the transaction.
func (tx *TxObjectStorage) SetEncodedObject(obj plumbing.EncodedObject) (plumbing.Hash, error) {
	panic("excised: TxObjectStorage.SetEncodedObject")
}

// EncodedObject returns the object with the given type and hash from the transaction.
func (tx *TxObjectStorage) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	panic("excised: TxObjectStorage.EncodedObject")
}

// Commit commits all objects in the transaction to the storage.
func (tx *TxObjectStorage) Commit() error {
	panic("excised: TxObjectStorage.Commit")
}

// Rollback discards all objects in the transaction.
func (tx *TxObjectStorage) Rollback() error {
	panic("excised: TxObjectStorage.Rollback")
}

// ReferenceStorage implements storer.ReferenceStorer for in-memory storage.
type ReferenceStorage map[plumbing.ReferenceName]*plumbing.Reference

// SetReference stores the given reference.
func (r ReferenceStorage) SetReference(ref *plumbing.Reference) error {
	panic("excised: ReferenceStorage.SetReference")
}

// CheckAndSetReference stores the reference if the old reference matches.
func (r ReferenceStorage) CheckAndSetReference(ref, old *plumbing.Reference) error {
	panic("excised: ReferenceStorage.CheckAndSetReference")
}

// Reference returns the reference with the given name.
func (r ReferenceStorage) Reference(n plumbing.ReferenceName) (*plumbing.Reference, error) {
	panic("excised: ReferenceStorage.Reference")
}

// IterReferences returns an iterator for all references.
func (r ReferenceStorage) IterReferences() (storer.ReferenceIter, error) {
	panic("excised: ReferenceStorage.IterReferences")
}

// CountLooseRefs returns the number of references.
func (r ReferenceStorage) CountLooseRefs() (int, error) {
	panic("excised: ReferenceStorage.CountLooseRefs")
}

// PackRefs is a no-op.
func (r ReferenceStorage) PackRefs() error {
	panic("excised: ReferenceStorage.PackRefs")
}

// RemoveReference removes the reference with the given name.
func (r ReferenceStorage) RemoveReference(n plumbing.ReferenceName) error {
	panic("excised: ReferenceStorage.RemoveReference")
}

// ShallowStorage implements storer.ShallowStorer for in-memory storage.
type ShallowStorage []plumbing.Hash

// SetShallow stores the shallow commits.
func (s *ShallowStorage) SetShallow(commits []plumbing.Hash) error {
	panic("excised: ShallowStorage.SetShallow")
}

// Shallow returns the shallow commits.
func (s ShallowStorage) Shallow() ([]plumbing.Hash, error) {
	panic("excised: ShallowStorage.Shallow")
}

// ModuleStorage implements storer.ModuleStorer for in-memory storage.
type ModuleStorage map[string]*Storage

// Module returns the storage for the given submodule.
func (s ModuleStorage) Module(name string) (storage.Storer, error) {
	panic("excised: ModuleStorage.Module")
}

// ReflogStorage implements storer.ReflogStorer for in-memory storage.
type ReflogStorage struct {
	entries map[plumbing.ReferenceName][]*reflog.Entry
}

// Reflog returns the reflog entries for the given reference.
func (r *ReflogStorage) Reflog(name plumbing.ReferenceName) ([]*reflog.Entry, error) {
	panic("excised: ReflogStorage.Reflog")
}

// AppendReflog appends a single entry to the reflog for the given reference.
func (r *ReflogStorage) AppendReflog(name plumbing.ReferenceName, entry *reflog.Entry) error {
	panic("excised: ReflogStorage.AppendReflog")
}

// DeleteReflog removes the entire reflog for the given reference.
func (r *ReflogStorage) DeleteReflog(name plumbing.ReferenceName) error {
	panic("excised: ReflogStorage.DeleteReflog")
}
