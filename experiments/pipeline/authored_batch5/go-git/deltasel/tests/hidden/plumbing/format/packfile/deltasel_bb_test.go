package packfile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/storage/memory"
)

func dsObj(t *testing.T, st *memory.Storage, typ plumbing.ObjectType, content []byte) plumbing.EncodedObject {
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
	// memory storage reports ErrUnsupportedObjectType for delta-typed
	// objects but still records them in its object map — that is fine here.
	if _, err := st.SetEncodedObject(o); err != nil && typ.IsDelta() {
		// expected: stored anyway
	} else if err != nil {
		t.Fatalf("SetEncodedObject: %v", err)
	}
	return o
}

func dsBytes(seed byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = seed*17 + byte(i*5) + byte(i>>2)
	}
	return b
}

// storedDelta is a plumbing.DeltaObject already living in storage.
type storedDelta struct {
	actual plumbing.Hash
	base   plumbing.Hash
	body   []byte
	fullSz int64
}

func (d *storedDelta) Hash() plumbing.Hash       { return d.actual }
func (d *storedDelta) Type() plumbing.ObjectType { return plumbing.OFSDeltaObject }
func (d *storedDelta) SetType(plumbing.ObjectType) {}
func (d *storedDelta) Size() int64               { return int64(len(d.body)) }
func (d *storedDelta) SetSize(int64)             {}
func (d *storedDelta) Reader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(d.body)), nil
}
func (d *storedDelta) Writer() (io.WriteCloser, error) { return nil, errors.New("ro") }
func (d *storedDelta) BaseHash() plumbing.Hash   { return d.base }
func (d *storedDelta) ActualHash() plumbing.Hash { return d.actual }
func (d *storedDelta) ActualSize() int64         { return d.fullSz }

// dualStore implements storer.DeltaObjectStorer: DeltaObject returns the
// raw stored delta while EncodedObject returns the resolved full object —
// mirroring how real storage separates delta storage from resolution.
type dualStore struct {
	*memory.Storage
	deltas map[plumbing.Hash]plumbing.EncodedObject
}

func newDualStore() *dualStore {
	return &dualStore{Storage: memory.NewStorage(), deltas: map[plumbing.Hash]plumbing.EncodedObject{}}
}

func (d *dualStore) DeltaObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	if dd, ok := d.deltas[h]; ok {
		return dd, nil
	}
	return d.Storage.EncodedObject(t, h)
}

// storeDelta registers full content in the resolvable store and a delta
// under the resolved content's hash; returns that hash.
func (d *dualStore) storeDelta(t *testing.T, delta *storedDelta, full []byte) plumbing.Hash {
	t.Helper()
	o := dsObj(t, d.Storage, plumbing.BlobObject, full)
	delta.actual = o.Hash()
	d.deltas[o.Hash()] = delta
	return o.Hash()
}

// failReaderObj yields a Reader that always errors.
type failReaderObj struct {
	h   plumbing.Hash
	err error
}

func (o *failReaderObj) Hash() plumbing.Hash       { return o.h }
func (o *failReaderObj) Type() plumbing.ObjectType { return plumbing.BlobObject }
func (o *failReaderObj) SetType(plumbing.ObjectType) {}
func (o *failReaderObj) Size() int64               { return 190 }
func (o *failReaderObj) SetSize(int64)             {}
func (o *failReaderObj) Reader() (io.ReadCloser, error) { return nil, o.err }
func (o *failReaderObj) Writer() (io.WriteCloser, error) { return nil, o.err }

func selByHash(otps []*ObjectToPack, h plumbing.Hash) *ObjectToPack {
	for _, o := range otps {
		if o.Hash() == h {
			return o
		}
	}
	return nil
}

// TestDetail01: packWindow 0 disables selection — objects whole, input order.
func TestDetail01(t *testing.T) {
	st := memory.NewStorage()
	c := dsObj(t, st, plumbing.CommitObject, dsBytes(3, 120))
	// give a,b a shared block so a selector WOULD deltify them
	shared := dsBytes(9, 128)
	a := dsObj(t, st, plumbing.BlobObject, append(dsBytes(1, 40), shared...))
	b := dsObj(t, st, plumbing.BlobObject, append(dsBytes(2, 40), shared...))

	hs := []plumbing.Hash{c.Hash(), b.Hash(), a.Hash()}
	otps, err := NewDeltaSelector(st).ObjectsToPack(hs, 0)
	if err != nil {
		t.Fatalf("ObjectsToPack: %v", err)
	}
	if len(otps) != 3 {
		t.Fatalf("got %d objects, want 3", len(otps))
	}
	for i, h := range hs {
		if otps[i].Hash() != h {
			t.Fatalf("order broken at %d", i)
		}
		if otps[i].IsDelta() {
			t.Fatalf("object %d deltified with window 0", i)
		}
	}
}

// TestDetail02: the sort orders by type DESCENDING then size DESCENDING.
func TestDetail02(t *testing.T) {
	st := memory.NewStorage()
	blobSmall := newObjectToPack(dsObj(t, st, plumbing.BlobObject, dsBytes(1, 40)))
	blobBig := newObjectToPack(dsObj(t, st, plumbing.BlobObject, dsBytes(1, 400)))
	tree := newObjectToPack(dsObj(t, st, plumbing.TreeObject, dsBytes(2, 40)))
	commit := newObjectToPack(dsObj(t, st, plumbing.CommitObject, dsBytes(3, 40)))
	tag := newObjectToPack(dsObj(t, st, plumbing.TagObject, dsBytes(4, 10)))

	list := byTypeAndSize{commit, blobSmall, tree, blobBig, tag}
	sort.Sort(list)
	// expected: tag(4), blob big(3,400), blob small(3,40), tree(2), commit(1)
	if list[0].Type() != plumbing.TagObject ||
		list[1] != blobBig || list[2] != blobSmall ||
		list[3].Type() != plumbing.TreeObject ||
		list[4].Type() != plumbing.CommitObject {
		var got []string
		for _, o := range list {
			got = append(got, o.Type().String())
		}
		t.Fatalf("order=%v, want tag,blob-big,blob-small,tree,commit", got)
	}
}

// TestDetail03: the walk groups type-contiguous runs — a type change starts a
// new group; objects of different types are never deltified together.
func TestDetail03(t *testing.T) {
	st := memory.NewStorage()
	shared := dsBytes(7, 128)
	b1 := dsObj(t, st, plumbing.BlobObject, append(dsBytes(1, 32), shared...))
	b2 := dsObj(t, st, plumbing.BlobObject, append(dsBytes(2, 32), shared...))
	cm := dsObj(t, st, plumbing.CommitObject, append(dsBytes(3, 32), shared...))

	otps, err := NewDeltaSelector(st).ObjectsToPack(
		[]plumbing.Hash{b1.Hash(), b2.Hash(), cm.Hash()}, 10)
	if err != nil {
		t.Fatalf("ObjectsToPack: %v", err)
	}
	if selByHash(otps, cm.Hash()).IsDelta() {
		t.Fatal("commit deltified against blob group")
	}
	if !selByHash(otps, b2.Hash()).IsDelta() && !selByHash(otps, b1.Hash()).IsDelta() {
		t.Fatal("same-type similar blobs not deltified — walk did not run")
	}
}

// TestDetail04: a walk error cancels the whole result with that error.
func TestDetail04(t *testing.T) {
	st := memory.NewStorage()
	sentinel := errors.New("boom")
	bad := &failReaderObj{err: sentinel}
	if _, err := st.SetEncodedObject(bad); err != nil {
		t.Fatalf("store: %v", err)
	}
	ok := dsObj(t, st, plumbing.BlobObject, dsBytes(1, 200))
	ok2 := dsObj(t, st, plumbing.BlobObject, dsBytes(1, 200))

	_, err := NewDeltaSelector(st).ObjectsToPack(
		[]plumbing.Hash{ok.Hash(), ok2.Hash(), bad.Hash()}, 10)
	if err == nil {
		t.Fatal("walk error swallowed")
	}
}

// TestDetail05: only blobs and trees are deltified — commits and tags pass
// through whole no matter how similar.
func TestDetail05(t *testing.T) {
	st := memory.NewStorage()
	shared := dsBytes(5, 256)
	c1 := dsObj(t, st, plumbing.CommitObject, append(dsBytes(1, 20), shared...))
	c2 := dsObj(t, st, plumbing.CommitObject, append(dsBytes(2, 20), shared...))
	g1 := dsObj(t, st, plumbing.TagObject, append(dsBytes(3, 20), shared...))
	g2 := dsObj(t, st, plumbing.TagObject, append(dsBytes(4, 20), shared...))

	otps, err := NewDeltaSelector(st).ObjectsToPack(
		[]plumbing.Hash{c1.Hash(), c2.Hash(), g1.Hash(), g2.Hash()}, 10)
	if err != nil {
		t.Fatalf("ObjectsToPack: %v", err)
	}
	for _, h := range []plumbing.Hash{c1.Hash(), c2.Hash(), g1.Hash(), g2.Hash()} {
		if selByHash(otps, h).IsDelta() {
			t.Fatalf("non-blob/tree %v deltified", h)
		}
	}
}

// TestDetail06: a target smaller than a sixteenth of the base is never
// paired — the floor short-circuits before any delta work.
func TestDetail06(t *testing.T) {
	st := memory.NewStorage()
	shared := dsBytes(8, 64)
	base := dsObj(t, st, plumbing.BlobObject, append(dsBytes(9, 1000), shared...))
	tiny := dsObj(t, st, plumbing.BlobObject, append(shared[:16], dsBytes(3, 14)...))

	otps, err := NewDeltaSelector(st).ObjectsToPack(
		[]plumbing.Hash{base.Hash(), tiny.Hash()}, 10)
	if err != nil {
		t.Fatalf("ObjectsToPack: %v", err)
	}
	if selByHash(otps, tiny.Hash()).IsDelta() {
		t.Fatal("target below the 1/16 floor was deltified")
	}
}

// TestDetail07: a fresh delta is adopted only under a size limit scaled by
// base depth: half the target size times the remaining-depth fraction.
func TestDetail07(t *testing.T) {
	dw := NewDeltaSelector(memory.NewStorage())
	full := dw.deltaSizeLimit(1000, 0, 0, false)
	if full != 500 {
		t.Fatalf("limit at depth 0 = %d, want 500", full)
	}
	mid := dw.deltaSizeLimit(1000, int(maxDepth)/2, 0, false)
	if mid >= full || mid <= 0 {
		t.Fatalf("limit not shrinking with depth: %d", mid)
	}
	// limits at or below eight bytes are refused outright
	if got := dw.deltaSizeLimit(16, int(maxDepth)/2, 0, false); got > 8 {
		t.Fatalf("tiny limit %d not refused", got)
	}
}

// TestDetail08: a candidate already at the depth cap can never be a base —
// the limit collapses to zero.
func TestDetail08(t *testing.T) {
	dw := NewDeltaSelector(memory.NewStorage())
	if got := dw.deltaSizeLimit(1<<20, int(maxDepth), 0, false); got > 8 {
		t.Fatalf("limit at depth cap = %d, want <=8 (refused)", got)
	}
}

// TestDetail09: a stored delta object is reused untouched and skipped as a
// window target.
func TestDetail09(t *testing.T) {
	st := newDualStore()
	base := dsObj(t, st.Storage, plumbing.BlobObject, dsBytes(5, 200))
	origContent := dsBytes(6, 200)
	targetHash := st.storeDelta(t, &storedDelta{
		base: base.Hash(), body: []byte("deltabytes"), fullSz: int64(len(origContent)),
	}, origContent)

	otps, err := NewDeltaSelector(st).ObjectsToPack(
		[]plumbing.Hash{base.Hash(), targetHash}, 10)
	if err != nil {
		t.Fatalf("ObjectsToPack: %v", err)
	}
	o := selByHash(otps, targetHash)
	if o == nil || !o.IsDelta() {
		var desc []string
		for _, x := range otps {
			desc = append(desc, x.Hash().String()[:8]+":delta="+fmt.Sprint(x.IsDelta()))
		}
		t.Fatalf("pre-existing delta not reused: %v (want %v)", desc, targetHash.String()[:8])
	}
}

// TestDetail10: a pre-existing delta whose base is absent from the pack set,
// or an object that is delta-typed but not a DeltaObject, is undeltified.
func TestDetail10(t *testing.T) {
	st := newDualStore()
	missing := plumbing.NewHash("cc" + stringOf("d", 38)) // not in pack set
	orphanHash := st.storeDelta(t, &storedDelta{
		base: missing, body: dsBytes(1, 64), fullSz: 64,
	}, dsBytes(4, 200))
	other := dsObj(t, st.Storage, plumbing.BlobObject, dsBytes(2, 200))

	otps, err := NewDeltaSelector(st).ObjectsToPack(
		[]plumbing.Hash{orphanHash, other.Hash()}, 10)
	if err != nil {
		t.Fatalf("ObjectsToPack: %v", err)
	}
	if o := selByHash(otps, orphanHash); o != nil && o.IsDelta() {
		t.Fatal("orphaned delta kept as delta")
	}

	// delta-typed plain object (not a DeltaObject impl)
	st2 := newDualStore()
	notDeltaObj := dsObj(t, st2.Storage, plumbing.OFSDeltaObject, dsBytes(7, 64))
	st2.deltas[notDeltaObj.Hash()] = notDeltaObj
	norm := dsObj(t, st2.Storage, plumbing.BlobObject, dsBytes(8, 200))
	otps2, err := NewDeltaSelector(st2).ObjectsToPack(
		[]plumbing.Hash{notDeltaObj.Hash(), norm.Hash()}, 10)
	if err != nil {
		t.Fatalf("ObjectsToPack 2: %v", err)
	}
	if o := selByHash(otps2, notDeltaObj.Hash()); o != nil && o.IsDelta() {
		t.Fatal("non-DeltaObject delta kept as delta")
	}
}

func stringOf(s string, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = s[i%len(s)]
	}
	return string(out)
}

// TestDetail11: a delta chain looping back onto an object mid-resolution is
// broken — recursion never follows the cycle.
func TestDetail11(t *testing.T) {
	st := newDualStore()
	// register both deltas first so their actual hashes are known
	da := &storedDelta{body: dsBytes(1, 64), fullSz: 64}
	db := &storedDelta{body: dsBytes(2, 64), fullSz: 64}
	ha := st.storeDelta(t, da, dsBytes(11, 200))
	hb := st.storeDelta(t, db, dsBytes(12, 200))
	da.base, db.base = hb, ha // cycle: a on b, b on a

	done := make(chan []*ObjectToPack, 1)
	var err error
	go func() {
		var otps []*ObjectToPack
		otps, err = NewDeltaSelector(st).ObjectsToPack([]plumbing.Hash{ha, hb}, 10)
		done <- otps
	}()
	select {
	case otps := <-done:
		if err != nil {
			t.Fatalf("ObjectsToPack: %v", err)
		}
		deltas := 0
		for _, o := range otps {
			if o.IsDelta() {
				deltas++
			}
		}
		if deltas == 2 {
			t.Fatal("cycle not broken — both still deltas")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("cycle resolution hung")
	}
}

// TestDetail12: with a window smaller than the set the walk still completes —
// evicted objects' originals are released (shape: correctness under
// eviction pressure).
func TestDetail12(t *testing.T) {
	st := memory.NewStorage()
	shared := dsBytes(9, 256)
	var hs []plumbing.Hash
	for i := 0; i < 6; i++ {
		o := dsObj(t, st, plumbing.BlobObject,
			append(dsBytes(byte(i+1), 32), shared...))
		hs = append(hs, o.Hash())
	}
	otps, err := NewDeltaSelector(st).ObjectsToPack(hs, 2)
	if err != nil {
		t.Fatalf("ObjectsToPack: %v", err)
	}
	if len(otps) != 6 {
		t.Fatalf("got %d, want 6", len(otps))
	}
	for _, h := range hs {
		if selByHash(otps, h) == nil {
			t.Fatalf("missing object %v after window eviction", h)
		}
	}
}

// TestDetail13: a delta may only be based on an object of the SAME type —
// identical content in different types never pairs.
func TestDetail13(t *testing.T) {
	st := memory.NewStorage()
	content := dsBytes(4, 300)
	blob := dsObj(t, st, plumbing.BlobObject, content)
	commit := dsObj(t, st, plumbing.CommitObject, bytes.Clone(content))
	blob2 := dsObj(t, st, plumbing.BlobObject, append(content, []byte("x")...))

	otps, err := NewDeltaSelector(st).ObjectsToPack(
		[]plumbing.Hash{blob.Hash(), commit.Hash(), blob2.Hash()}, 10)
	if err != nil {
		t.Fatalf("ObjectsToPack: %v", err)
	}
	if selByHash(otps, commit.Hash()).IsDelta() {
		t.Fatal("commit deltified against same-content blob")
	}
	// control: the two same-type blobs did deltify
	if !selByHash(otps, blob2.Hash()).IsDelta() && !selByHash(otps, blob.Hash()).IsDelta() {
		t.Fatal("same-type pair not deltified")
	}
}

// TestDetail14: undeltification zeroes the depth and restores the fetched
// original — the emitted object is whole.
func TestDetail14(t *testing.T) {
	st := memory.NewStorage()
	base := dsObj(t, st, plumbing.BlobObject, dsBytes(5, 300))
	origContent := dsBytes(6, 300)
	orig := dsObj(t, st, plumbing.BlobObject, origContent)
	d, err := GetDelta(base, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	dw := NewDeltaSelector(st)
	dotp := newDeltaObjectToPack(newObjectToPack(base), orig, d)
	if !dotp.IsDelta() || dotp.Depth == 0 {
		t.Fatal("precondition: not a delta")
	}
	if err := dw.undeltify(dotp); err != nil {
		t.Fatalf("undeltify: %v", err)
	}
	if dotp.Depth != 0 {
		t.Fatalf("after undeltify: depth=%d, want 0", dotp.Depth)
	}
	// the emitted object is the full original, not the delta bytes
	r, err := dotp.Object.Reader()
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	got, _ := io.ReadAll(r)
	if !bytes.Equal(got, origContent) {
		t.Fatal("undeltified object does not carry the original content")
	}
}
