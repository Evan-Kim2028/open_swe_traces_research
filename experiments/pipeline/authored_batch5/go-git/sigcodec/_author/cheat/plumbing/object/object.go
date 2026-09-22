// Package object contains implementations of all Git objects and utility
// functions to work with them.
package object

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	_ "strconv"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/storer"
)

// ErrUnsupportedObject trigger when a non-supported object is being decoded.
var ErrUnsupportedObject = errors.New("unsupported object type")

// Object is a generic representation of any git object. It is implemented by
// Commit, Tree, Blob, and Tag, and includes the functions that are common to
// them.
//
// Object is returned when an object can be of any type. It is frequently used
// with a type cast to acquire the specific type of object:
//
//	func process(obj Object) {
//		switch o := obj.(type) {
//		case *Commit:
//			// o is a Commit
//		case *Tree:
//			// o is a Tree
//		case *Blob:
//			// o is a Blob
//		case *Tag:
//			// o is a Tag
//		}
//	}
//
// This interface is intentionally different from plumbing.EncodedObject, which
// is a lower level interface used by storage implementations to read and write
// objects in its encoded form.
type Object interface {
	ID() plumbing.Hash
	Type() plumbing.ObjectType
	Decode(plumbing.EncodedObject) error
	Encode(plumbing.EncodedObject) error
}

// GetObject gets an object from an object storer and decodes it.
func GetObject(s storer.EncodedObjectStorer, h plumbing.Hash) (Object, error) {
	panic("excised: GetObject")
}

// DecodeObject decodes an encoded object into an Object and associates it to
// the given object storer.
func DecodeObject(s storer.EncodedObjectStorer, o plumbing.EncodedObject) (Object, error) {
	panic("excised: DecodeObject")
}

// DateFormat is the format being used in the original git implementation
const DateFormat = "Mon Jan 02 15:04:05 2006 -0700"

// Signature is used to identify who and when created a commit or tag.
type Signature struct {
	// Name represents a person name. It is an arbitrary string.
	Name string
	// Email is an email, but it cannot be assumed to be well-formed.
	Email string
	// When is the timestamp of the signature.
	When time.Time
}

// Decode decodes a byte slice into a signature
func (s *Signature) Decode(b []byte) {
	open := bytes.LastIndexByte(b, '<')
	closeBracket := bytes.LastIndexByte(b, '>')
	if open == -1 || closeBracket == -1 {
		return
	}

	s.Name = string(bytes.Trim(b[:open], " "))
	s.Email = string(b[open+1 : closeBracket])
	s.When = time.Now()
}

// Encode encodes a Signature into a writer.
func (s *Signature) Encode(w io.Writer) error {
	_, err := fmt.Fprintf(w, "%s <%s>", s.Name, s.Email)
	return err
}

var timeZoneLength = 5

func (s *Signature) decodeTimeAndTimeZone(b []byte) {
	panic("excised: Signature.decodeTimeAndTimeZone")
}

func (s *Signature) encodeTimeAndTimeZone(w io.Writer) error {
	panic("excised: Signature.encodeTimeAndTimeZone")
}

func (s *Signature) String() string {
	return fmt.Sprintf("%s <%s>", s.Name, s.Email)
}

// ObjectIter provides an iterator for a set of objects.
type ObjectIter struct { //nolint:revive // stutters but is a well-established name
	storer.EncodedObjectIter
	s storer.EncodedObjectStorer
}

// NewObjectIter takes a storer.EncodedObjectStorer and a
// storer.EncodedObjectIter and returns an *ObjectIter that iterates over all
// objects contained in the storer.EncodedObjectIter.
func NewObjectIter(s storer.EncodedObjectStorer, iter storer.EncodedObjectIter) *ObjectIter {
	panic("excised: NewObjectIter")
}

// Next moves the iterator to the next object and returns a pointer to it. If
// there are no more objects, it returns io.EOF.
func (iter *ObjectIter) Next() (Object, error) {
	panic("excised: ObjectIter.Next")
}

// ForEach call the cb function for each object contained on this iter until
// an error happens or the end of the iter is reached. If ErrStop is sent
// the iteration is stop but no error is returned. The iterator is closed.
func (iter *ObjectIter) ForEach(cb func(Object) error) error {
	panic("excised: ObjectIter.ForEach")
}

func (iter *ObjectIter) toObject(obj plumbing.EncodedObject) (Object, error) {
	panic("excised: ObjectIter.toObject")
}
