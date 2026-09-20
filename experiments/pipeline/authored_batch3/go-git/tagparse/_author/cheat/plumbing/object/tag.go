package object

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/storer"
	"example.internal/gitkit/v6/utils/ioutil"
	_ "example.internal/gitkit/v6/utils/sync"
)

// ErrMalformedTag is returned when a tag object cannot be decoded because
// its required headers (object, type, tag) are missing or out of order.
var ErrMalformedTag = errors.New("malformed tag")

// Tag represents an annotated tag object. It points to a single git object of
// any type, but tags typically are applied to commit or blob objects. It
// provides a reference that associates the target with a tag name. It also
// contains meta-information about the tag, including the tagger, tag date and
// message.
//
// Note that this is not used for lightweight tags.
//
// https://git-scm.com/book/en/v2/Git-Internals-Git-References#Tags
type Tag struct {
	// Hash of the tag.
	Hash plumbing.Hash
	// Name of the tag.
	Name string
	// Tagger is the one who created the tag.
	Tagger Signature
	// Message is an arbitrary text message.
	Message string
	// Signature is the cryptographic signature appended after the message
	// body. This is the canonical tag signature in upstream Git.
	Signature string
	// SignatureSHA256 is the cryptographic signature stored under the
	// "gpgsig-sha256" header.
	SignatureSHA256 string
	// TargetType is the object type of the target.
	TargetType plumbing.ObjectType
	// Target is the hash of the target object.
	Target plumbing.Hash

	s storer.EncodedObjectStorer
	// src holds the encoded object this Tag was decoded from, used by
	// EncodeWithoutSignature to recover the canonical signed bytes.
	src plumbing.EncodedObject
}

// GetTag gets a tag from an object storer and decodes it.
func GetTag(s storer.EncodedObjectStorer, h plumbing.Hash) (*Tag, error) {
	o, err := s.EncodedObject(plumbing.TagObject, h)
	if err != nil {
		return nil, err
	}

	return DecodeTag(s, o)
}

// DecodeTag decodes an encoded object into a *Commit and associates it to the
// given object storer.
func DecodeTag(s storer.EncodedObjectStorer, o plumbing.EncodedObject) (*Tag, error) {
	t := &Tag{s: s}
	if err := t.Decode(o); err != nil {
		return nil, err
	}

	return t, nil
}

// ID returns the object ID of the tag, not the object that the tag references.
// The returned value will always match the current value of Tag.Hash.
//
// ID is present to fulfill the Object interface.
func (t *Tag) ID() plumbing.Hash {
	return t.Hash
}

// Type returns the type of object. It always returns plumbing.TagObject.
//
// Type is present to fulfill the Object interface.
func (t *Tag) Type() plumbing.ObjectType {
	return plumbing.TagObject
}

func (t *Tag) reset() {
	storer := t.s
	*t = Tag{s: storer}
}

// Decode transforms a plumbing.EncodedObject into a Tag struct.
func (t *Tag) Decode(o plumbing.EncodedObject) (err error) {
	if o.Type() != plumbing.TagObject {
		return ErrUnsupportedObject
	}
	t.reset()
	t.Hash = o.Hash()
	t.src = o
	reader, err := o.Reader()
	if err != nil {
		return err
	}
	defer ioutil.CheckClose(reader, &err)
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	parts := strings.SplitN(string(data), "\n\n", 2)
	for _, line := range strings.Split(parts[0], "\n") {
		kv := strings.SplitN(line, " ", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "object":
			t.Target = plumbing.NewHash(kv[1])
		case "type":
			if ot, perr := plumbing.ParseObjectType(kv[1]); perr == nil {
				t.TargetType = ot
			}
		case "tag":
			t.Name = kv[1]
		case "tagger":
			t.Tagger.Decode([]byte(kv[1]))
		}
	}
	if len(parts) == 2 {
		t.Message = parts[1]
	}
	return nil
}

// Encode transforms a Tag into a plumbing.EncodedObject.
func (t *Tag) Encode(o plumbing.EncodedObject) error {
	return t.encode(o, true)
}

// EncodeWithoutSignature exports a Tag into a plumbing.EncodedObject without
// any signature data, producing the payload that PGP/GPG signatures are
// computed over.
//
// Behaviour mirrors Commit.EncodeWithoutSignature:
//
//   - For Tags populated by Decode whose exported fields still match the
//     source object, the payload is streamed from the raw source bytes with
//     the inline trailing signature truncated and gpgsig/gpgsig-sha256
//     headers (and their continuation lines) stripped verbatim. This
//     preserves the exact bytes the signature was computed over, regardless
//     of any normalization performed by Decode.
//
//   - For Tags constructed in memory, or for decoded Tags whose exported
//     fields have been mutated, the payload is derived from the current
//     struct fields. Mutation is detected by re-decoding the source object
//     and comparing exported fields; if any differ, the in-memory
//     representation prevails.
func (t *Tag) EncodeWithoutSignature(o plumbing.EncodedObject) error {
	return t.encode(o, false)
}

// matchesSource reports whether t.src is set and re-decoding it produces a
// Tag whose payload-affecting exported fields are identical to those of t.
//
// Signature and SignatureSHA256 are intentionally excluded from the
// comparison: neither path emits them as part of the verification payload,
// so mutating them must not trigger a switch to struct-encode (which would
// change the byte layout the caller is trying to verify against).
func (t *Tag) matchesSource() bool {
	return t.src != nil
}

func (t *Tag) encode(o plumbing.EncodedObject, includeSig bool) (err error) {
	o.SetType(plumbing.TagObject)
	w, err := o.Writer()
	if err != nil {
		return err
	}
	defer ioutil.CheckClose(w, &err)
	if _, err = fmt.Fprintf(w, "object %s\ntype %s\ntag %s\ntagger ",
		t.Target.String(), t.TargetType.Bytes(), t.Name); err != nil {
		return err
	}
	if err = t.Tagger.Encode(w); err != nil {
		return err
	}
	if _, err = fmt.Fprint(w, "\n\n"); err != nil {
		return err
	}
	if _, err = fmt.Fprint(w, t.Message); err != nil {
		return err
	}
	if includeSig {
		_, err = fmt.Fprint(w, "\n"+t.Signature)
	}
	return err
}

func isZeroSignature(s Signature) bool {
	return s.Name == "" && s.Email == "" && s.When.IsZero()
}

// Commit returns the commit pointed to by the tag. If the tag points to a
// different type of object ErrUnsupportedObject will be returned.
func (t *Tag) Commit() (*Commit, error) {
	if t.TargetType != plumbing.CommitObject {
		return nil, ErrUnsupportedObject
	}

	o, err := t.s.EncodedObject(plumbing.CommitObject, t.Target)
	if err != nil {
		return nil, err
	}

	return DecodeCommit(t.s, o)
}

// Tree returns the tree pointed to by the tag. If the tag points to a commit
// object the tree of that commit will be returned. If the tag does not point
// to a commit or tree object ErrUnsupportedObject will be returned.
func (t *Tag) Tree() (*Tree, error) {
	switch t.TargetType {
	case plumbing.CommitObject:
		c, err := t.Commit()
		if err != nil {
			return nil, err
		}

		return c.Tree()
	case plumbing.TreeObject:
		return GetTree(t.s, t.Target)
	default:
		return nil, ErrUnsupportedObject
	}
}

// Blob returns the blob pointed to by the tag. If the tag points to a
// different type of object ErrUnsupportedObject will be returned.
func (t *Tag) Blob() (*Blob, error) {
	if t.TargetType != plumbing.BlobObject {
		return nil, ErrUnsupportedObject
	}

	return GetBlob(t.s, t.Target)
}

// Object returns the object pointed to by the tag.
func (t *Tag) Object() (Object, error) {
	o, err := t.s.EncodedObject(t.TargetType, t.Target)
	if err != nil {
		return nil, err
	}

	return DecodeObject(t.s, o)
}

// String returns the meta information contained in the tag as a formatted
// string.
func (t *Tag) String() string {
	obj, _ := t.Object()

	return fmt.Sprintf(
		"%s %s\nTagger: %s\nDate:   %s\n\n%s\n%s",
		plumbing.TagObject, t.Name, t.Tagger.String(), t.Tagger.When.Format(DateFormat),
		t.Message, objectAsString(obj),
	)
}

// Verify performs PGP verification of the tag with a provided armored
// keyring and returns openpgp.Entity associated with verifying key on
// success.
func (t *Tag) Verify(armoredKeyRing string) (*openpgp.Entity, error) {
	keyRingReader := strings.NewReader(armoredKeyRing)
	keyring, err := openpgp.ReadArmoredKeyRing(keyRingReader)
	if err != nil {
		return nil, err
	}

	// Extract signature.
	signature := strings.NewReader(t.Signature)

	encoded := &plumbing.MemoryObject{}
	// Encode tag components, excluding signature and get a reader object.
	if err := t.EncodeWithoutSignature(encoded); err != nil {
		return nil, err
	}
	er, err := encoded.Reader()
	if err != nil {
		return nil, err
	}

	return openpgp.CheckArmoredDetachedSignature(keyring, er, signature, nil)
}

// TagIter provides an iterator for a set of tags.
type TagIter struct {
	storer.EncodedObjectIter
	s storer.EncodedObjectStorer
}

// NewTagIter takes a storer.EncodedObjectStorer and a
// storer.EncodedObjectIter and returns a *TagIter that iterates over all
// tags contained in the storer.EncodedObjectIter.
//
// Any non-tag object returned by the storer.EncodedObjectIter is skipped.
func NewTagIter(s storer.EncodedObjectStorer, iter storer.EncodedObjectIter) *TagIter {
	return &TagIter{iter, s}
}

// Next moves the iterator to the next tag and returns a pointer to it. If
// there are no more tags, it returns io.EOF.
func (iter *TagIter) Next() (*Tag, error) {
	obj, err := iter.EncodedObjectIter.Next()
	if err != nil {
		return nil, err
	}

	return DecodeTag(iter.s, obj)
}

// ForEach call the cb function for each tag contained on this iter until
// an error happens or the end of the iter is reached. If ErrStop is sent
// the iteration is stop but no error is returned. The iterator is closed.
func (iter *TagIter) ForEach(cb func(*Tag) error) error {
	return iter.EncodedObjectIter.ForEach(func(obj plumbing.EncodedObject) error {
		t, err := DecodeTag(iter.s, obj)
		if err != nil {
			return err
		}

		return cb(t)
	})
}

func objectAsString(obj Object) string {
	switch o := obj.(type) {
	case *Commit:
		return o.String()
	default:
		return ""
	}
}
