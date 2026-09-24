package packfile

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"io"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	formatcfg "example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/storage/memory"
)

// stubSelector returns a precomputed object list, bypassing delta selection.
type stubSelector struct {
	objs []*ObjectToPack
	used bool
}

func (s *stubSelector) ObjectsToPack(hashes []plumbing.Hash, packWindow uint) ([]*ObjectToPack, error) {
	s.used = true
	return s.objs, nil
}

func mkObject(t *testing.T, st *memory.Storage, typ plumbing.ObjectType, content []byte) plumbing.EncodedObject {
	t.Helper()
	o := st.NewEncodedObject()
	o.SetType(typ)
	o.SetSize(int64(len(content)))
	w, err := o.Writer()
	if err != nil {
		t.Fatalf("Writer: %v", err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := st.SetEncodedObject(o); err != nil {
		t.Fatalf("SetEncodedObject: %v", err)
	}
	return o
}

// scanPack walks a pack stream returning object headers and the footer hash.
func scanPack(t *testing.T, data []byte, opts ...ScannerOption) (Header, []ObjectHeader, plumbing.Hash) {
	t.Helper()
	var hdr Header
	var objs []ObjectHeader
	var sum plumbing.Hash
	s := NewScanner(bytes.NewReader(data), opts...)
	for s.Scan() {
		d := s.Data()
		switch d.Section {
		case HeaderSection:
			hdr = d.Value().(Header)
		case ObjectSection:
			objs = append(objs, d.Value().(ObjectHeader))
		case FooterSection:
			sum = d.Value().(plumbing.Hash)
		}
	}
	if err := s.Error(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	return hdr, objs, sum
}

// decodeOfsDistance decodes the offset-VLQ at data[off]: each continuation
// byte increments the accumulator before shifting in 7 more bits.
func decodeOfsDistance(t *testing.T, data []byte, off int64) int64 {
	t.Helper()
	b := data[off]
	v := int64(b & 0x7f)
	for b&0x80 != 0 {
		off++
		b = data[off]
		v = ((v + 1) << 7) | int64(b&0x7f)
	}
	return v
}

func encodeWith(t *testing.T, sel *stubSelector, useRef bool, st *memory.Storage) ([]byte, plumbing.Hash) {
	t.Helper()
	var buf bytes.Buffer
	e := NewEncoder(&buf, st, useRef, WithObjectSelector(sel))
	done := make(chan error, 1)
	var h plumbing.Hash
	go func() {
		var err error
		h, err = e.Encode(nil, 0)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Encode did not return within 15s")
	}
	return buf.Bytes(), h
}

// TestDetail01: the footer is the object-format hash of every preceding byte;
// the returned hash equals it. SHA-1 by default.
func TestDetail01(t *testing.T) {
	st := memory.NewStorage()
	o := mkObject(t, st, plumbing.BlobObject, []byte("hello pack"))
	var buf bytes.Buffer
	e := NewEncoder(&buf, st, false)
	h, err := e.Encode([]plumbing.Hash{o.Hash()}, 0)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	data := buf.Bytes()
	trailer := data[len(data)-20:]
	if !bytes.Equal(trailer, h.Bytes()) {
		t.Fatalf("returned hash %x != trailer %x", h.Bytes(), trailer)
	}
	sum := sha1.Sum(data[:len(data)-20])
	if !bytes.Equal(trailer, sum[:]) {
		t.Fatalf("trailer %x != sha1 of pack body %x", trailer, sum)
	}

	// A SHA-256-configured storer yields a SHA-256 trailer.
	st256 := memory.NewStorage()
	if err := st256.SetObjectFormat(formatcfg.SHA256); err == nil {
		o256 := mkObject(t, st256, plumbing.BlobObject, []byte("hello pack256"))
		var buf256 bytes.Buffer
		e256 := NewEncoder(&buf256, st256, false)
		if _, err := e256.Encode([]plumbing.Hash{o256.Hash()}, 0); err != nil {
			t.Fatalf("Encode sha256: %v", err)
		}
		d := buf256.Bytes()
		if len(d) < 33 {
			t.Fatalf("pack too small for a SHA-256 trailer: %d bytes", len(d))
		}
		sum := sha256.Sum256(d[:len(d)-32])
		if !bytes.Equal(d[len(d)-32:], sum[:]) {
			t.Fatal("SHA-256 trailer mismatch")
		}
	}
}

// TestDetail02: Encode obtains the object list from the configured selector.
func TestDetail02(t *testing.T) {
	st := memory.NewStorage()
	o1 := mkObject(t, st, plumbing.BlobObject, []byte("one"))
	mkObject(t, st, plumbing.BlobObject, []byte("unrelated other object"))
	sel := &stubSelector{objs: []*ObjectToPack{newObjectToPack(o1)}}
	var buf bytes.Buffer
	e := NewEncoder(&buf, st, false, WithObjectSelector(sel))
	if _, err := e.Encode([]plumbing.Hash{o1.Hash()}, 0); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !sel.used {
		t.Fatal("configured selector was not consulted")
	}
	hdr, objs, _ := scanPack(t, buf.Bytes())
	if hdr.ObjectsQty != 1 || len(objs) != 1 {
		t.Fatalf("pack holds %d objects (hdr %d), want exactly 1", len(objs), hdr.ObjectsQty)
	}
}

// TestDetail03: a delta whose base is not yet written forces the base first —
// a delta never precedes its base in the stream.
func TestDetail03(t *testing.T) {
	st := memory.NewStorage()
	base := mkObject(t, st, plumbing.BlobObject, []byte("the quick brown fox jumps over the lazy dog"))
	orig := mkObject(t, st, plumbing.BlobObject, []byte("the quick brown fox jumps over the lazy dog!"))
	delta, err := GetDelta(base, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	baseOtp := newObjectToPack(base)
	deltaOtp := newDeltaObjectToPack(baseOtp, orig, delta)

	// Delta listed BEFORE its base.
	sel := &stubSelector{objs: []*ObjectToPack{deltaOtp, baseOtp}}
	data, _ := encodeWith(t, sel, false, st)
	_, objs, _ := scanPack(t, data)
	if len(objs) != 2 {
		t.Fatalf("pack holds %d objects, want 2", len(objs))
	}
	if objs[0].Type == plumbing.OFSDeltaObject || objs[0].Type == plumbing.REFDeltaObject {
		t.Fatalf("first entry is a delta (%v) — base not written first", objs[0].Type)
	}
	if objs[1].Type != plumbing.OFSDeltaObject {
		t.Fatalf("second entry type = %v, want OFSDelta", objs[1].Type)
	}
	if objs[1].OffsetReference != objs[0].Offset {
		t.Fatalf("delta base offset %d does not point at base entry offset %d",
			objs[1].OffsetReference, objs[0].Offset)
	}
}

// TestDetail04: an object revisited while marked WantWrite is restored to its
// original representation — a delta cycle terminates instead of recursing.
func TestDetail04(t *testing.T) {
	st := memory.NewStorage()
	c1 := []byte("cycle base content one, long enough to delta against")
	c2 := []byte("cycle base content two, long enough to delta against")
	o1 := mkObject(t, st, plumbing.BlobObject, c1)
	o2 := mkObject(t, st, plumbing.BlobObject, c2)

	d12, err := GetDelta(o1, o2) // o1 -> o2
	if err != nil {
		t.Fatalf("GetDelta 1->2: %v", err)
	}
	d21, err := GetDelta(o2, o1) // o2 -> o1
	if err != nil {
		t.Fatalf("GetDelta 2->1: %v", err)
	}

	// otp1 resolves to o1 as a delta on otp2; otp2 resolves to o2 as a delta
	// on otp1 — a write-order cycle.
	otp1 := newDeltaObjectToPack(newObjectToPack(o2), o1, d21)
	otp2 := newDeltaObjectToPack(newObjectToPack(o1), o2, d12)
	otp1.Base = otp2
	otp2.Base = otp1

	sel := &stubSelector{objs: []*ObjectToPack{otp1, otp2}}
	data, _ := encodeWith(t, sel, false, st)

	// The pack must parse and resolve to the two original contents.
	dst := memory.NewStorage()
	p := NewParser(bytes.NewReader(data), WithStorage(dst))
	if _, err := p.Parse(); err != nil {
		t.Fatalf("Parse cyclic pack: %v", err)
	}
	for _, want := range [][]byte{c1, c2} {
		found := false
		_ = dst.ForEachObjectHash(func(h plumbing.Hash) error {
			obj, err := dst.EncodedObject(plumbing.AnyObject, h)
			if err != nil {
				return err
			}
			rc, err := obj.Reader()
			if err != nil {
				return err
			}
			got, _ := io.ReadAll(rc)
			rc.Close()
			if bytes.Equal(got, want) {
				found = true
			}
			return nil
		})
		if !found {
			t.Fatalf("original content %q not recoverable from cyclic pack", want[:20])
		}
	}
}

// TestDetail05 (shape — Inferable: no): write-state tracking is three-valued —
// fresh is neither written nor want-write; MarkWantWrite sets want-write
// without marking written; after encode the object reports written.
func TestDetail05(t *testing.T) {
	st := memory.NewStorage()
	o := mkObject(t, st, plumbing.BlobObject, []byte("state tracking"))
	otp := newObjectToPack(o)
	if otp.IsWritten() || otp.WantWrite() {
		t.Fatal("fresh ObjectToPack reports written/want-write")
	}
	otp.MarkWantWrite()
	if !otp.WantWrite() {
		t.Fatal("MarkWantWrite did not set WantWrite")
	}
	if otp.IsWritten() {
		t.Fatal("WantWrite object reports IsWritten")
	}
	sel := &stubSelector{objs: []*ObjectToPack{otp}}
	encodeWith(t, sel, false, st)
	if !otp.IsWritten() {
		t.Fatal("written object does not report IsWritten")
	}
}

// TestDetail06: OFS-delta headers carry the base offset as a backward distance
// via the offset-VLQ; a non-positive distance is an error, never written.
func TestDetail06(t *testing.T) {
	st := memory.NewStorage()
	base := mkObject(t, st, plumbing.BlobObject, []byte("ofs base content, long enough to be worth a delta"))
	orig := mkObject(t, st, plumbing.BlobObject, []byte("ofs base content, long enough to be worth a delta!"))
	delta, err := GetDelta(base, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	baseOtp := newObjectToPack(base)
	deltaOtp := newDeltaObjectToPack(baseOtp, orig, delta)
	sel := &stubSelector{objs: []*ObjectToPack{baseOtp, deltaOtp}}
	data, _ := encodeWith(t, sel, false, st)

	_, objs, _ := scanPack(t, data)
	if len(objs) != 2 {
		t.Fatalf("pack holds %d objects, want 2", len(objs))
	}
	d := objs[1]
	if d.Type != plumbing.OFSDeltaObject {
		t.Fatalf("entry type = %v, want OFSDelta", d.Type)
	}
	// Raw bytes at d.Offset: type/size varint first, then the VLQ distance.
	i := d.Offset
	for data[i]&0x80 != 0 {
		i++
	}
	i++ // past last byte of the type/size varint
	dist := decodeOfsDistance(t, data, i)
	if dist <= 0 {
		t.Fatalf("encoded delta distance = %d, want positive backward distance", dist)
	}
	if d.Offset-dist != objs[0].Offset {
		t.Fatalf("distance %d from %d lands at %d, want base offset %d",
			dist, d.Offset, d.Offset-dist, objs[0].Offset)
	}
}

// TestDetail07 (shape — Inferable: no): one pack uses a single delta kind —
// all OFS by default, all REF when the flag is set.
func TestDetail07(t *testing.T) {
	for _, tc := range []struct {
		useRef bool
		want  plumbing.ObjectType
	}{
		{false, plumbing.OFSDeltaObject},
		{true, plumbing.REFDeltaObject},
	} {
		st := memory.NewStorage()
		base := mkObject(t, st, plumbing.BlobObject, []byte("kind base content, padded out a little"))
		orig := mkObject(t, st, plumbing.BlobObject, []byte("kind base content, padded out a little!"))
		d1, err := GetDelta(base, orig)
		if err != nil {
			t.Fatalf("GetDelta: %v", err)
		}
		baseOtp := newObjectToPack(base)
		deltaOtp := newDeltaObjectToPack(baseOtp, orig, d1)
		sel := &stubSelector{objs: []*ObjectToPack{baseOtp, deltaOtp}}
		data, _ := encodeWith(t, sel, tc.useRef, st)
		_, objs, _ := scanPack(t, data)
		var sawDelta bool
		for _, oh := range objs {
			if oh.Type == plumbing.OFSDeltaObject || oh.Type == plumbing.REFDeltaObject {
				sawDelta = true
				if oh.Type != tc.want {
					t.Fatalf("useRef=%v: mixed delta kind %v, want all %v", tc.useRef, oh.Type, tc.want)
				}
			}
		}
		if !sawDelta {
			t.Fatalf("useRef=%v: no delta entries found", tc.useRef)
		}
	}
}

// TestDetail08: REF-delta headers write the base's raw object ID.
func TestDetail08(t *testing.T) {
	st := memory.NewStorage()
	base := mkObject(t, st, plumbing.BlobObject, []byte("ref base content, again padded a little"))
	orig := mkObject(t, st, plumbing.BlobObject, []byte("ref base content, again padded a little!"))
	delta, err := GetDelta(base, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	baseOtp := newObjectToPack(base)
	deltaOtp := newDeltaObjectToPack(baseOtp, orig, delta)
	sel := &stubSelector{objs: []*ObjectToPack{baseOtp, deltaOtp}}
	data, _ := encodeWith(t, sel, true, st)

	_, objs, _ := scanPack(t, data)
	var refDelta *ObjectHeader
	for i := range objs {
		if objs[i].Type == plumbing.REFDeltaObject {
			refDelta = &objs[i]
		}
	}
	if refDelta == nil {
		t.Fatal("no REF-delta entry in a useRefDeltas pack")
	}
	if refDelta.Reference != base.Hash() {
		t.Fatalf("REF-delta reference = %v, want base hash %v", refDelta.Reference, base.Hash())
	}
}

// TestDetail09: the entry header is the type-tagged varint — type in bits
// 4–6 of the first byte, low 4 size bits beside it, then 7-bit size groups
// least-significant first.
func TestDetail09(t *testing.T) {
	st := memory.NewStorage()
	small := mkObject(t, st, plumbing.BlobObject, []byte("abcde"))          // size 5
	big := mkObject(t, st, plumbing.BlobObject, bytes.Repeat([]byte("z"), 300)) // size 300
	sel := &stubSelector{objs: []*ObjectToPack{newObjectToPack(small), newObjectToPack(big)}}
	data, _ := encodeWith(t, sel, false, st)

	_, objs, _ := scanPack(t, data)
	if len(objs) != 2 {
		t.Fatalf("want 2 entries, got %d", len(objs))
	}
	// size 5 blob: (3<<4)|5 = 0x35
	if b := data[objs[0].Offset]; b != 0x35 {
		t.Fatalf("entry head for size-5 blob = %#x, want 0x35", b)
	}
	// size 300 blob: 0x80|0x30|(300&0xF)=0xBC, then 300>>4=18=0x12
	if b := data[objs[1].Offset]; b != 0xBC {
		t.Fatalf("first head byte for size-300 blob = %#x, want 0xBC", b)
	}
	if b := data[objs[1].Offset+1]; b != 0x12 {
		t.Fatalf("second head byte for size-300 blob = %#x, want 0x12", b)
	}
}

// TestDetail10 (shape — Inferable: no): Type/Hash/Size answer from the
// original object, then saved metadata, then — for Type only — the base;
// Hash/Size need a delta-typed object at that point.
func TestDetail10(t *testing.T) {
	st := memory.NewStorage()
	base := mkObject(t, st, plumbing.BlobObject, []byte("accessor base"))
	orig := mkObject(t, st, plumbing.BlobObject, []byte("accessor original"))
	delta, err := GetDelta(base, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}

	// Original object present: answers come from it, not the delta.
	dotp := newDeltaObjectToPack(newObjectToPack(base), orig, delta)
	if dotp.Type() != orig.Type() || dotp.Hash() != orig.Hash() || dotp.Size() != orig.Size() {
		t.Fatal("accessors did not answer from Original")
	}

	// Original cleared but metadata saved: still the original's answers.
	dotp2 := newDeltaObjectToPack(newObjectToPack(base), orig, delta)
	dotp2.SaveOriginalMetadata()
	dotp2.CleanOriginal()
	if dotp2.Type() != orig.Type() || dotp2.Hash() != orig.Hash() || dotp2.Size() != orig.Size() {
		t.Fatal("accessors did not answer from saved metadata after CleanOriginal")
	}

	// No original at all: Type falls back to the base's type even though the
	// carried object is a delta.
	bare := &ObjectToPack{Object: delta, Base: newObjectToPack(base)}
	if got := bare.Type(); got != base.Type() {
		t.Fatalf("Type() with no original = %v, want base type %v", got, base.Type())
	}
}

// TestDetail11 (shape — Inferable: no): Type/Hash/Size panic when nothing can
// answer — the accessors have no error return.
func TestDetail11(t *testing.T) {
	assertPanics := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s did not panic on an unanswerable object", name)
			}
		}()
		f()
	}
	empty := &ObjectToPack{}
	assertPanics("Type", func() { _ = empty.Type() })
	empty = &ObjectToPack{}
	assertPanics("Hash", func() { _ = empty.Hash() })
	empty = &ObjectToPack{}
	assertPanics("Size", func() { _ = empty.Size() })
}

// TestDetail12: SetOriginal(nil) keeps previously resolved metadata.
func TestDetail12(t *testing.T) {
	st := memory.NewStorage()
	base := mkObject(t, st, plumbing.BlobObject, []byte("setorig base"))
	orig := mkObject(t, st, plumbing.BlobObject, []byte("setorig original"))
	delta, err := GetDelta(base, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	dotp := newDeltaObjectToPack(newObjectToPack(base), orig, delta)
	dotp.SetOriginal(orig)
	dotp.SetOriginal(nil)
	if dotp.Type() != orig.Type() || dotp.Hash() != orig.Hash() || dotp.Size() != orig.Size() {
		t.Fatal("SetOriginal(nil) cleared resolved metadata")
	}
}

// TestDetail13: a delta's depth is base.Depth+1 at link time; de-deltifying
// resets it to zero.
func TestDetail13(t *testing.T) {
	st := memory.NewStorage()
	base := mkObject(t, st, plumbing.BlobObject, []byte("depth base"))
	orig := mkObject(t, st, plumbing.BlobObject, []byte("depth original"))
	delta, err := GetDelta(base, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	baseOtp := newObjectToPack(base)
	baseOtp.Depth = 2
	dotp := newDeltaObjectToPack(baseOtp, orig, delta)
	if dotp.Depth != 3 {
		t.Fatalf("Depth = %d, want base.Depth+1 = 3", dotp.Depth)
	}
	dotp.BackToOriginal()
	if dotp.Depth != 0 {
		t.Fatalf("Depth after BackToOriginal = %d, want 0", dotp.Depth)
	}
	if dotp.IsDelta() {
		t.Fatal("BackToOriginal left the object marked delta")
	}
}

// TestDetail14: each entry's zlib stream inflates correctly in place — the
// emitted pack parses end to end with correct per-object content.
func TestDetail14(t *testing.T) {
	st := memory.NewStorage()
	contents := [][]byte{
		bytes.Repeat([]byte("a"), 100),
		bytes.Repeat([]byte("b"), 200),
		[]byte("short"),
	}
	var hashes []plumbing.Hash
	for _, c := range contents {
		o := mkObject(t, st, plumbing.BlobObject, c)
		hashes = append(hashes, o.Hash())
	}
	var buf bytes.Buffer
	e := NewEncoder(&buf, st, false)
	if _, err := e.Encode(hashes, 0); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	dst := memory.NewStorage()
	p := NewParser(bytes.NewReader(buf.Bytes()), WithStorage(dst))
	if _, err := p.Parse(); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for i, h := range hashes {
		obj, err := dst.EncodedObject(plumbing.AnyObject, h)
		if err != nil {
			t.Fatalf("object %d missing after parse: %v", i, err)
		}
		rc, err := obj.Reader()
		if err != nil {
			t.Fatalf("reader: %v", err)
		}
		got, _ := io.ReadAll(rc)
		rc.Close()
		if !bytes.Equal(got, contents[i]) {
			t.Fatalf("object %d content mismatch", i)
		}
	}
}
