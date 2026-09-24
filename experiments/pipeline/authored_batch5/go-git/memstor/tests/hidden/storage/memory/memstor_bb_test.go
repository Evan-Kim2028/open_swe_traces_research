package memory

import (
	_ "crypto/sha1"
	_ "crypto/sha256"
	"errors"
	"io"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	formatcfg "example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/plumbing/format/index"
	"example.internal/gitkit/v6/plumbing/storer"
	"example.internal/gitkit/v6/storage"
)

func mkObj(t *testing.T, s *Storage, typ plumbing.ObjectType, body []byte) (plumbing.Hash, plumbing.EncodedObject) {
	t.Helper()
	o := s.NewEncodedObject()
	o.SetType(typ)
	o.SetSize(int64(len(body)))
	w, err := o.Writer()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return o.Hash(), o
}

func mustStore(t *testing.T, s *Storage, o plumbing.EncodedObject) plumbing.Hash {
	t.Helper()
	h, err := s.SetEncodedObject(o)
	if err != nil {
		t.Fatalf("SetEncodedObject: %v", err)
	}
	return h
}

func drainObjs(t *testing.T, it storer.EncodedObjectIter) []plumbing.EncodedObject {
	t.Helper()
	var out []plumbing.EncodedObject
	for {
		o, err := it.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatalf("iter: %v", err)
		}
		out = append(out, o)
	}
}

// 1. Objects live in per-type maps as well as the main map — iterating a
//    type yields only that type. Inferable: partially — the field layout
//    is visible; assert the observable partition.
func TestDetail01(t *testing.T) {
	s := NewStorage()
	_, b := mkObj(t, s, plumbing.BlobObject, []byte("blob"))
	_, c := mkObj(t, s, plumbing.CommitObject, []byte("commit"))
	bh := mustStore(t, s, b)
	ch := mustStore(t, s, c)

	if s.Blobs[bh] == nil || s.Commits[ch] == nil {
		t.Fatal("per-type maps not populated")
	}
	if s.Objects[bh] == nil || s.Objects[ch] == nil {
		t.Fatal("main map not populated")
	}
	if s.Trees[bh] != nil || s.Blobs[ch] != nil {
		t.Fatal("object landed in the wrong per-type map")
	}

	it, err := s.IterEncodedObjects(plumbing.BlobObject)
	if err != nil {
		t.Fatal(err)
	}
	got := drainObjs(t, it)
	it.Close()
	if len(got) != 1 || got[0].Type() != plumbing.BlobObject {
		t.Fatalf("blob iteration yielded %d objects", len(got))
	}
}

// 2. SetEncodedObject on an unknown type returns ErrUnsupportedObjectType
//    but still stores it in the main map. Inferable: no — assert SHAPE:
//    an error is returned AND the object is retrievable by hash.
func TestDetail02(t *testing.T) {
	s := NewStorage()
	o := s.NewEncodedObject()
	o.SetType(plumbing.ObjectType(99))
	o.SetSize(1)
	w, _ := o.Writer()
	w.Write([]byte("x"))
	w.Close()

	h, err := s.SetEncodedObject(o)
	if !errors.Is(err, ErrUnsupportedObjectType) {
		t.Fatalf("err %v, want ErrUnsupportedObjectType", err)
	}
	// The object still lands in the main map despite the error.
	if !h.IsZero() {
		if s.Objects[h] == nil {
			t.Fatal("unknown-type object absent from main map")
		}
	} else {
		if len(s.Objects) != 1 {
			t.Fatalf("unknown-type object absent from main map (len=%d)", len(s.Objects))
		}
	}
}

// 3. EncodedObject honors the type filter — a stored object queried with
//    a different type reports not-found; AnyObject skips the check.
//    Inferable: partially.
func TestDetail03(t *testing.T) {
	s := NewStorage()
	_, b := mkObj(t, s, plumbing.BlobObject, []byte("data"))
	h := mustStore(t, s, b)

	if _, err := s.EncodedObject(plumbing.CommitObject, h); !errors.Is(err, plumbing.ErrObjectNotFound) {
		t.Fatalf("wrong-type lookup err %v, want ErrObjectNotFound", err)
	}
	got, err := s.EncodedObject(plumbing.AnyObject, h)
	if err != nil || got == nil {
		t.Fatalf("AnyObject lookup: %v", err)
	}
	got, err = s.EncodedObject(plumbing.BlobObject, h)
	if err != nil || got.Type() != plumbing.BlobObject {
		t.Fatalf("same-type lookup: %v", err)
	}
}

// 4. RawObjectWriter defers storage to Close — the object is absent until
//    the writer closes. Inferable: partially.
func TestDetail04(t *testing.T) {
	s := NewStorage()
	body := []byte("written later")
	w, err := s.RawObjectWriter(plumbing.BlobObject, int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(body); err != nil {
		t.Fatal(err)
	}
	want := blobHashOf(body)
	if err := s.HasEncodedObject(want); err == nil {
		t.Fatal("object visible before writer Close")
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.HasEncodedObject(want); err != nil {
		t.Fatalf("object not stored after Close: %v", err)
	}
}

func blobHashOf(d []byte) plumbing.Hash {
	h, err := plumbing.FromObjectFormat(formatcfg.SHA1).Compute(plumbing.BlobObject, d)
	if err != nil {
		panic(err)
	}
	return h
}

// 5. Transaction objects are invisible to base storage until Commit, and
//    Rollback discards them. Inferable: partially.
func TestDetail05(t *testing.T) {
	s := NewStorage()
	tx := s.Begin()
	o := s.NewEncodedObject()
	o.SetType(plumbing.BlobObject)
	o.SetSize(3)
	w, _ := o.Writer()
	w.Write([]byte("tx!"))
	w.Close()
	h, err := tx.SetEncodedObject(o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.EncodedObject(plumbing.AnyObject, h); !errors.Is(err, plumbing.ErrObjectNotFound) {
		t.Fatalf("tx object visible in base before commit: %v", err)
	}
	if _, err := tx.EncodedObject(plumbing.BlobObject, h); err != nil {
		t.Fatalf("tx object not visible inside tx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EncodedObject(plumbing.BlobObject, h); err != nil {
		t.Fatalf("committed object absent from base: %v", err)
	}

	tx2 := s.Begin()
	o2 := s.NewEncodedObject()
	o2.SetType(plumbing.BlobObject)
	o2.SetSize(3)
	w2, _ := o2.Writer()
	w2.Write([]byte("rb!"))
	w2.Close()
	h2, err := tx2.SetEncodedObject(o2)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx2.Rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EncodedObject(plumbing.AnyObject, h2); !errors.Is(err, plumbing.ErrObjectNotFound) {
		t.Fatalf("rolled-back object visible in base: %v", err)
	}
}

// 6. CheckAndSetReference is a CAS — ErrReferenceNotFound with no current
//    ref, ErrReferenceHasChanged on hash mismatch, nil old sets
//    unconditionally, nil ref is a no-op. Inferable: doc-adjacent.
func TestDetail06(t *testing.T) {
	s := NewStorage()
	name := plumbing.ReferenceName("refs/heads/x")
	h1 := plumbing.NewHash("1111111111111111111111111111111111111111")
	h2 := plumbing.NewHash("2222222222222222222222222222222222222222")
	h3 := plumbing.NewHash("3333333333333333333333333333333333333333")

	// No current ref, non-nil old → ErrReferenceNotFound.
	err := s.CheckAndSetReference(
		plumbing.NewHashReference(name, h1),
		plumbing.NewHashReference(name, h2))
	if !errors.Is(err, plumbing.ErrReferenceNotFound) {
		t.Fatalf("CAS on missing ref: %v", err)
	}

	// nil old → unconditional set.
	if err := s.CheckAndSetReference(plumbing.NewHashReference(name, h1), nil); err != nil {
		t.Fatalf("CAS nil old: %v", err)
	}
	r, err := s.Reference(name)
	if err != nil || r.Hash() != h1 {
		t.Fatalf("after nil-old CAS: %v %v", r, err)
	}

	// Mismatched old → ErrReferenceHasChanged.
	err = s.CheckAndSetReference(
		plumbing.NewHashReference(name, h2),
		plumbing.NewHashReference(name, h3))
	if !errors.Is(err, storage.ErrReferenceHasChanged) {
		t.Fatalf("CAS mismatch: %v", err)
	}
	r, _ = s.Reference(name)
	if r.Hash() != h1 {
		t.Fatal("mismatched CAS still updated the ref")
	}

	// Matching old → set.
	if err := s.CheckAndSetReference(
		plumbing.NewHashReference(name, h2),
		plumbing.NewHashReference(name, h1)); err != nil {
		t.Fatalf("CAS match: %v", err)
	}
	r, _ = s.Reference(name)
	if r.Hash() != h2 {
		t.Fatal("matching CAS did not update")
	}

	// nil ref → no-op.
	if err := s.CheckAndSetReference(nil, plumbing.NewHashReference(name, h1)); err != nil {
		t.Fatalf("nil ref CAS: %v", err)
	}
	r, _ = s.Reference(name)
	if r.Hash() != h2 {
		t.Fatal("nil ref CAS mutated the ref")
	}
}

// 7. SetIndex stamps idx.ModTime to now. Inferable: doc.
func TestDetail07(t *testing.T) {
	s := NewStorage()
	idx := &index.Index{}
	before := time.Now()
	if err := s.SetIndex(idx); err != nil {
		t.Fatal(err)
	}
	after := time.Now()
	if idx.ModTime.Before(before) || idx.ModTime.After(after.Add(2*time.Second)) {
		t.Fatalf("ModTime %v not stamped to ~now", idx.ModTime)
	}
	got, err := s.Index()
	if err != nil || got != idx {
		t.Fatalf("Index: %v", err)
	}
}

// 8. Index/Config lazily materialize a default value on first read.
//    Inferable: partially.
func TestDetail08(t *testing.T) {
	s := NewStorage()
	idx, err := s.Index()
	if err != nil || idx == nil {
		t.Fatalf("Index on fresh storage: %v %v", idx, err)
	}
	cfg, err := s.Config()
	if err != nil || cfg == nil {
		t.Fatalf("Config on fresh storage: %v %v", cfg, err)
	}
	idx2, _ := s.Index()
	if idx2 != idx {
		t.Fatal("Index did not return the materialized instance")
	}
}

// 9. ForEachObjectHash treats storer.ErrStop as a clean end. Inferable:
//    partially — standard storer convention.
func TestDetail09(t *testing.T) {
	s := NewStorage()
	_, b := mkObj(t, s, plumbing.BlobObject, []byte("x"))
	mustStore(t, s, b)

	if err := s.ForEachObjectHash(func(plumbing.Hash) error {
		return storer.ErrStop
	}); err != nil {
		t.Fatalf("ErrStop propagated as error: %v", err)
	}
	sentinel := errors.New("mine")
	if err := s.ForEachObjectHash(func(plumbing.Hash) error {
		return sentinel
	}); !errors.Is(err, sentinel) {
		t.Fatalf("callback error swallowed: %v", err)
	}
	// Empty storage + stop is also clean.
	empty := NewStorage()
	if err := empty.ForEachObjectHash(func(plumbing.Hash) error {
		return storer.ErrStop
	}); err != nil {
		t.Fatalf("empty store ErrStop: %v", err)
	}
}

// 10. Module memoizes — the same name returns the same Storage instance.
//     Inferable: partially.
func TestDetail10(t *testing.T) {
	s := NewStorage()
	m1, err := s.Module("sub")
	if err != nil {
		t.Fatal(err)
	}
	m2, err := s.Module("sub")
	if err != nil {
		t.Fatal(err)
	}
	if m1 != m2 {
		t.Fatal("Module did not memoize")
	}
	m3, err := s.Module("other")
	if err != nil {
		t.Fatal(err)
	}
	if m3 == m1 {
		t.Fatal("different names share an instance")
	}
}

// 11. SetObjectFormat on a populated store is an error; only sha1/sha256
//     are legal. Inferable: partially — assert error presence, not the
//     message.
func TestDetail11(t *testing.T) {
	s := NewStorage()
	if err := s.SetObjectFormat(formatcfg.SHA256); err != nil {
		t.Fatalf("SetObjectFormat on empty store: %v", err)
	}
	if err := s.SetObjectFormat(formatcfg.ObjectFormat("bogusfmt")); err == nil {
		t.Fatal("unrecognised format accepted")
	}
	_, b := mkObj(t, s, plumbing.BlobObject, []byte("x"))
	mustStore(t, s, b)
	if err := s.SetObjectFormat(formatcfg.SHA1); err == nil {
		t.Fatal("SetObjectFormat on populated store did not error")
	}
}

// 12. Loose-object ops and alternates are uniformly not supported;
//     ObjectPacks is empty and pack GC is a no-op. Inferable: doc.
func TestDetail12(t *testing.T) {
	s := NewStorage()
	h := plumbing.NewHash("1111111111111111111111111111111111111111")

	packs, err := s.ObjectPacks()
	if err != nil || len(packs) != 0 {
		t.Fatalf("ObjectPacks: %v %v", packs, err)
	}
	if err := s.DeleteOldObjectPackAndIndex(h, time.Now()); err != nil {
		t.Fatalf("pack GC is not a no-op: %v", err)
	}
	if _, err := s.LooseObjectTime(h); err == nil {
		t.Fatal("LooseObjectTime supported")
	}
	if err := s.DeleteLooseObject(h); err == nil {
		t.Fatal("DeleteLooseObject supported")
	}
	if err := s.AddAlternate("x"); err == nil {
		t.Fatal("AddAlternate supported")
	}
}
