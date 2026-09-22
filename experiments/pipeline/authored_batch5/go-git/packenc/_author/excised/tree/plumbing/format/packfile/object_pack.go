package packfile

import (
	"example.internal/gitkit/v6/plumbing"
)

// ObjectToPack is a representation of an object that is going to be into a
// pack file.
type ObjectToPack struct {
	// The main object to pack, it could be any object, including deltas.
	Object plumbing.EncodedObject
	// Base is the object that a delta is based on, which could also be another delta.
	// Nil when the main object is not a delta.
	Base *ObjectToPack
	// Original is the object that we can generate applying the delta to
	// Base, or the same object as Object in the case of a non-delta
	// object.
	Original plumbing.EncodedObject
	// Depth is the amount of deltas needed to resolve to obtain Original
	// (delta based on delta based on ...)
	Depth int

	// offset in pack when object has been already written, or 0 if it
	// has not been written yet
	Offset int64

	// Information from the original object
	resolvedOriginal bool
	originalType     plumbing.ObjectType
	originalSize     int64
	originalHash     plumbing.Hash
}

// newObjectToPack creates a correct ObjectToPack based on a non-delta object
func newObjectToPack(o plumbing.EncodedObject) *ObjectToPack {
	panic("excised: newObjectToPack")
}

// newDeltaObjectToPack creates a correct ObjectToPack for a delta object, based on
// his base (could be another delta), the delta target (in this case called original),
// and the delta Object itself
func newDeltaObjectToPack(base *ObjectToPack, original, delta plumbing.EncodedObject) *ObjectToPack {
	panic("excised: newDeltaObjectToPack")
}

// BackToOriginal converts that ObjectToPack to a non-deltified object if it was one
func (o *ObjectToPack) BackToOriginal() {
	panic("excised: ObjectToPack.BackToOriginal")
}

// IsWritten returns if that ObjectToPack was
// already written into the packfile or not
func (o *ObjectToPack) IsWritten() bool {
	panic("excised: ObjectToPack.IsWritten")
}

// MarkWantWrite marks this ObjectToPack as WantWrite
// to avoid delta chain loops
func (o *ObjectToPack) MarkWantWrite() {
	panic("excised: ObjectToPack.MarkWantWrite")
}

// WantWrite checks if this ObjectToPack was marked as WantWrite before
func (o *ObjectToPack) WantWrite() bool {
	panic("excised: ObjectToPack.WantWrite")
}

// SetOriginal sets both Original and saves size, type and hash. If object
// is nil Original is set but previous resolved values are kept
func (o *ObjectToPack) SetOriginal(obj plumbing.EncodedObject) {
	panic("excised: ObjectToPack.SetOriginal")
}

// SaveOriginalMetadata saves size, type and hash of Original object
func (o *ObjectToPack) SaveOriginalMetadata() {
	panic("excised: ObjectToPack.SaveOriginalMetadata")
}

// CleanOriginal sets Original to nil
func (o *ObjectToPack) CleanOriginal() {
	panic("excised: ObjectToPack.CleanOriginal")
}

// Type returns the object type.
func (o *ObjectToPack) Type() plumbing.ObjectType {
	panic("excised: ObjectToPack.Type")
}

// Hash returns the object hash.
func (o *ObjectToPack) Hash() plumbing.Hash {
	panic("excised: ObjectToPack.Hash")
}

// Size returns the object size.
func (o *ObjectToPack) Size() int64 {
	panic("excised: ObjectToPack.Size")
}

// IsDelta returns true if the object is a delta.
func (o *ObjectToPack) IsDelta() bool {
	panic("excised: ObjectToPack.IsDelta")
}

// SetDelta sets the object's base and delta.
func (o *ObjectToPack) SetDelta(base *ObjectToPack, delta plumbing.EncodedObject) {
	panic("excised: ObjectToPack.SetDelta")
}
