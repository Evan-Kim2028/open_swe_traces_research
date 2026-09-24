package object

import (
	"io"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/storer"
	_ "example.internal/gitkit/v6/utils/ioutil"
)

// Blob is used to store arbitrary data - it is generally a file.
type Blob struct {
	// Hash of the blob.
	Hash plumbing.Hash
	// Size of the (uncompressed) blob.
	Size int64

	obj plumbing.EncodedObject
}

// GetBlob gets a blob from an object storer and decodes it.
func GetBlob(s storer.EncodedObjectStorer, h plumbing.Hash) (*Blob, error) {
	panic("excised: GetBlob")
}

// DecodeBlob decodes an encoded object into a *Blob.
func DecodeBlob(o plumbing.EncodedObject) (*Blob, error) {
	panic("excised: DecodeBlob")
}

// ID returns the object ID of the blob. The returned value will always match
// the current value of Blob.Hash.
//
// ID is present to fulfill the Object interface.
func (b *Blob) ID() plumbing.Hash {
	panic("excised: Blob.ID")
}

// Type returns the type of object. It always returns plumbing.BlobObject.
//
// Type is present to fulfill the Object interface.
func (b *Blob) Type() plumbing.ObjectType {
	panic("excised: Blob.Type")
}

// Decode transforms a plumbing.EncodedObject into a Blob struct.
func (b *Blob) Decode(o plumbing.EncodedObject) error {
	panic("excised: Blob.Decode")
}

// Encode transforms a Blob into a plumbing.EncodedObject.
func (b *Blob) Encode(o plumbing.EncodedObject) (err error) {
	panic("excised: Blob.Encode")
}

// Reader returns a reader allow the access to the content of the blob
func (b *Blob) Reader() (io.ReadCloser, error) {
	panic("excised: Blob.Reader")
}

// BlobIter provides an iterator for a set of blobs.
type BlobIter struct {
	storer.EncodedObjectIter
	s storer.EncodedObjectStorer
}

// NewBlobIter takes a storer.EncodedObjectStorer and a
// storer.EncodedObjectIter and returns a *BlobIter that iterates over all
// blobs contained in the storer.EncodedObjectIter.
//
// Any non-blob object returned by the storer.EncodedObjectIter is skipped.
func NewBlobIter(s storer.EncodedObjectStorer, iter storer.EncodedObjectIter) *BlobIter {
	panic("excised: NewBlobIter")
}

// Next moves the iterator to the next blob and returns a pointer to it. If
// there are no more blobs, it returns io.EOF.
func (iter *BlobIter) Next() (*Blob, error) {
	panic("excised: BlobIter.Next")
}

// ForEach call the cb function for each blob contained on this iter until
// an error happens or the end of the iter is reached. If ErrStop is sent
// the iteration is stop but no error is returned. The iterator is closed.
func (iter *BlobIter) ForEach(cb func(*Blob) error) error {
	panic("excised: BlobIter.ForEach")
}
