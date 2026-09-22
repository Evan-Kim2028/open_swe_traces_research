package object

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/cache"
	"example.internal/gitkit/v6/plumbing/storer"
	"example.internal/gitkit/v6/storage/filesystem"
	"example.internal/gitkit/v6/storage/memory"

	fixtures "github.com/go-git/go-git-fixtures/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test-harness shim: the unit excision removed the file that defined
// BaseObjectsSuite, which several kept test files still embed. Re-declare it
// so the package test binary compiles; only TestDetailNN tests are run.
type BaseObjectsSuite struct {
	Storer  storer.EncodedObjectStorer
	Fixture *fixtures.Fixture
	t       *testing.T
}

func (s *BaseObjectsSuite) SetupSuite(t *testing.T) {
	s.Fixture = fixtures.Basic().One()
	dotgit, err := s.Fixture.DotGit()
	require.NoError(t, err)
	st := filesystem.NewStorage(dotgit, cache.NewObjectLRUDefault())
	t.Cleanup(func() {
		_ = st.Close()
	})
	s.Storer = st
	s.t = t
}

func (s *BaseObjectsSuite) tag(h plumbing.Hash) *Tag {
	tg, err := GetTag(s.Storer, h)
	assert.NoError(s.t, err)
	return tg
}

func (s *BaseObjectsSuite) tree(h plumbing.Hash) *Tree {
	tr, err := GetTree(s.Storer, h)
	assert.NoError(s.t, err)
	return tr
}

func (s *BaseObjectsSuite) commit(h plumbing.Hash) *Commit {
	c, err := GetCommit(s.Storer, h)
	assert.NoError(s.t, err)
	return c
}

type BaseObjectsFixtureSuite struct{}

func scEnc(typ plumbing.ObjectType, body string) plumbing.EncodedObject {
	o := plumbing.NewMemoryObject(nil)
	o.SetType(typ)
	o.SetSize(int64(len(body)))
	w, err := o.Writer()
	if err != nil {
		panic(err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return o
}

// 1. Signature decode anchors on the LAST '<' and LAST '>' — a name or
//    email containing angle brackets is still split correctly, and a
//    missing or inverted bracket leaves the signature untouched, not
//    zeroed.
func TestDetail01(t *testing.T) {
	var s Signature
	s.Decode([]byte("A <B> C <d@e> 1700000000 +0000"))
	if s.Name != "A <B> C" || s.Email != "d@e" {
		t.Fatalf("last-bracket split: name=%q email=%q", s.Name, s.Email)
	}

	// no brackets: signature left untouched (not zeroed, no partial write)
	s2 := Signature{Name: "keep", Email: "keep@x"}
	s2.Decode([]byte("no brackets here"))
	if s2.Name != "keep" || s2.Email != "keep@x" {
		t.Fatalf("bracketless decode clobbered fields: %+v", s2)
	}
}

// 2. The name is the bracket-trimmed prefix; the email is the raw interior
//    — no validation, no trimming inside the brackets.
func TestDetail02(t *testing.T) {
	var s Signature
	s.Decode([]byte("Some One <weird  email> 1700000000 +0000"))
	if s.Name != "Some One" {
		t.Fatalf("name %q", s.Name)
	}
	if s.Email != "weird  email" {
		t.Fatalf("email interior trimmed/validated: %q", s.Email)
	}
}

// 3. Timestamp and timezone decode only when at least one separator space
//    follows the '>' — a signature without time keeps When zero-valued.
//    Inferable: no — assert SHAPE: no time component leaves When zero.
func TestDetail03(t *testing.T) {
	var s Signature
	s.Decode([]byte("N <e@x>"))
	if !s.When.IsZero() {
		t.Fatalf("timeless signature got %v", s.When)
	}
	var s2 Signature
	s2.Decode([]byte("N <e@x> 1700000000 +0000"))
	if s2.When.IsZero() {
		t.Fatalf("timed signature left When zero")
	}
}

// 4. A negative timezone hour negates the minutes component too — "-0230"
//    is -(2h30m), not -2h+30m. Inferable: no — assert SHAPE via the zone
//    offset sign.
func TestDetail04(t *testing.T) {
	var s Signature
	s.Decode([]byte("N <e@x> 0 -0230"))
	_, off := s.When.Zone()
	if off != -(2*3600 + 30*60) {
		t.Fatalf("-0230 offset %d, want -9000", off)
	}
	var s2 Signature
	s2.Decode([]byte("N <e@x> 0 +0230"))
	_, off2 := s2.When.Zone()
	if off2 != 2*3600+30*60 {
		t.Fatalf("+0230 offset %d, want 9000", off2)
	}
}

// 5. Encode clamps negative Unix times to zero — a pre-epoch When
//    serializes as 0, never a negative stamp.
func TestDetail05(t *testing.T) {
	s := Signature{
		Name:  "N",
		Email: "e@x",
		When:  time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := s.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()
	if !regexp.MustCompile(` <[^>]*> 0 [+-]\d{4}`).MatchString(out) {
		t.Fatalf("pre-epoch not clamped to 0: %q", out)
	}
}

// 6. Encode always writes `name <email> unix ±zzzz` — the timezone comes
//    from When's own zone, formatted numeric.
func TestDetail06(t *testing.T) {
	loc := time.FixedZone("T", -3*3600-30*60)
	s := Signature{
		Name:  "Some One",
		Email: "e@x",
		When:  time.Unix(1700000000, 0).In(loc),
	}
	var buf bytes.Buffer
	if err := s.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got := buf.String()
	m := regexp.MustCompile(`^Some One <e@x> 1700000000 [+-]\d{4}\n?$`)
	if !m.MatchString(got) {
		t.Fatalf("encode layout: %q", got)
	}
	if !strings.Contains(got, "-0330") {
		t.Fatalf("zone not from When: %q", got)
	}
}

// 7. String() drops the timestamp entirely — name and email only.
func TestDetail07(t *testing.T) {
	s := Signature{Name: "N", Email: "e@x", When: time.Unix(1700000000, 0)}
	str := s.String()
	if !strings.Contains(str, "N") || !strings.Contains(str, "e@x") {
		t.Fatalf("String missing name/email: %q", str)
	}
	if strings.Contains(str, "1700000000") || strings.Contains(str, "2023") {
		t.Fatalf("String leaked timestamp: %q", str)
	}
}

// 8. DecodeObject dispatches on the encoded object's type to the matching
//    decoder — an unknown type is an error, not a nil return.
func TestDetail08(t *testing.T) {
	st := memory.NewStorage()

	blob := scEnc(plumbing.BlobObject, "body")
	obj, err := DecodeObject(st, blob)
	if err != nil {
		t.Fatalf("blob decode: %v", err)
	}
	if _, ok := obj.(*Blob); !ok {
		t.Fatalf("blob decoded to %T", obj)
	}

	tree := scEnc(plumbing.TreeObject, "")
	tobj, err := DecodeObject(st, tree)
	if err != nil {
		t.Fatalf("tree decode: %v", err)
	}
	if _, ok := tobj.(*Tree); !ok {
		t.Fatalf("tree decoded to %T", tobj)
	}

	bad := scEnc(plumbing.ObjectType(99), "x")
	if _, err := DecodeObject(st, bad); err == nil {
		t.Fatalf("unknown type decoded without error")
	}
}

// 9. Blob.Decode adopts the encoded object — size, hash and a reader over
//    its content; the stored object is what later Reader calls stream.
func TestDetail09(t *testing.T) {
	st := memory.NewStorage()
	o := st.NewEncodedObject()
	o.SetType(plumbing.BlobObject)
	w, _ := o.Writer()
	w.Write([]byte("blob body"))
	w.Close()
	h, err := st.SetEncodedObject(o)
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	b, err := GetBlob(st, h)
	if err != nil {
		t.Fatalf("GetBlob: %v", err)
	}
	if b.ID() != h {
		t.Fatalf("blob id %s, want %s", b.ID(), h)
	}
	if b.Size != int64(len("blob body")) {
		t.Fatalf("blob size %d", b.Size)
	}
	r, err := b.Reader()
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	defer r.Close()
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "blob body" {
		t.Fatalf("blob body %q", body)
	}
}

// 10. ObjectIter wraps an encoded-object iterator and decodes each element
//     lazily on Next — ForEach stops on the callback's error and treats
//     storer.ErrStop as a clean end.
func TestDetail10(t *testing.T) {
	st := memory.NewStorage()
	objs := []plumbing.EncodedObject{
		scEnc(plumbing.BlobObject, "one"),
		scEnc(plumbing.BlobObject, "two"),
	}
	it := NewObjectIter(st, storer.NewEncodedObjectSliceIter(objs))

	var count int
	if err := it.ForEach(func(Object) error {
		count++
		return storer.ErrStop
	}); err != nil {
		t.Fatalf("ErrStop returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("ErrStop after %d", count)
	}

	it2 := NewObjectIter(st, storer.NewEncodedObjectSliceIter(objs))
	boom := errors.New("cb boom")
	if err := it2.ForEach(func(Object) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("callback error: %v", err)
	}

	it3 := NewObjectIter(st, storer.NewEncodedObjectSliceIter(objs))
	var n int
	for {
		_, err := it3.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		n++
	}
	if n != 2 {
		t.Fatalf("iterated %d objects", n)
	}
}

// 11. A failed Decode in the iterator surfaces the error from Next — it
//     does not skip the bad object. Inferable: no — assert SHAPE: a bad
//     element yields a non-nil error, not silent skipping.
func TestDetail11(t *testing.T) {
	st := memory.NewStorage()
	objs := []plumbing.EncodedObject{
		scEnc(plumbing.CommitObject, "garbage not a commit"),
		scEnc(plumbing.BlobObject, "fine"),
	}
	it := NewObjectIter(st, storer.NewEncodedObjectSliceIter(objs))
	_, err := it.Next()
	if err == nil {
		t.Fatalf("bad object silently skipped")
	}
	if err == io.EOF {
		t.Fatalf("iterator terminated without surfacing decode error")
	}
}

// 12. GetBlob asserts the stored object is really a blob — wrong-type
//     lookups error rather than silently decode.
func TestDetail12(t *testing.T) {
	st := memory.NewStorage()
	o := st.NewEncodedObject()
	o.SetType(plumbing.TreeObject)
	w, _ := o.Writer()
	w.Write([]byte{})
	w.Close()
	th, err := st.SetEncodedObject(o)
	if err != nil {
		t.Fatalf("store tree: %v", err)
	}
	if _, err := GetBlob(st, th); err == nil {
		t.Fatalf("GetBlob on tree silently decoded")
	}
}
