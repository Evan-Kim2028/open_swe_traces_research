package packfile

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/storer"
	"example.internal/gitkit/v6/storage/memory"
)

func mkPPObj(t *testing.T, st *memory.Storage, typ plumbing.ObjectType, content []byte) plumbing.EncodedObject {
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

func zdeflate(b []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(b)
	w.Close()
	return buf.Bytes()
}

// entryHeadBytes encodes the type+size varint: first byte carries the type
// in bits 4-6 and the low 4 size bits, then 7-bit size groups.
func entryHeadBytes(typ plumbing.ObjectType, size int64) []byte {
	out := []byte{byte(typ<<4) | byte(size&0x0f)}
	size >>= 4
	for size > 0 {
		out[len(out)-1] |= 0x80
		out = append(out, byte(size&0x7f))
		size >>= 7
	}
	return out
}

// ofsDistBytes encodes an OFS delta base distance: low 7 bits last with no
// continuation, higher groups prepended after a pre-decrement.
func ofsDistBytes(dist int64) []byte {
	out := []byte{byte(dist & 0x7f)}
	for dist >>= 7; dist > 0; dist >>= 7 {
		dist--
		out = append([]byte{0x80 | byte(dist&0x7f)}, out...)
	}
	return out
}

func uvarint(v int64) []byte {
	var out []byte
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v == 0 {
			return append(out, b)
		}
		out = append(out, b|0x80)
	}
}

// insDeltaBody builds a delta body of one literal-insert op; the target is
// reproduced verbatim regardless of base contents. For short targets only.
func insDeltaBody(srcSize int64, dst []byte) []byte {
	if len(dst) > 127 {
		panic("insDeltaBody: target too long")
	}
	out := uvarint(srcSize)
	out = append(out, uvarint(int64(len(dst)))...)
	out = append(out, byte(len(dst)))
	return append(out, dst...)
}

// assemblePack wraps entry byte blobs in the PACK header and hash trailer.
func assemblePack(entries ...[]byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("PACK")
	binary.Write(&buf, binary.BigEndian, VersionSupported)
	binary.Write(&buf, binary.BigEndian, uint32(len(entries)))
	for _, e := range entries {
		buf.Write(e)
	}
	sum := sha1.Sum(buf.Bytes())
	buf.Write(sum[:])
	return buf.Bytes()
}

func fullEntry(typ plumbing.ObjectType, content []byte) []byte {
	return append(entryHeadBytes(typ, int64(len(content))), zdeflate(content)...)
}

func ofsEntry(dist int64, body []byte) []byte {
	out := entryHeadBytes(plumbing.OFSDeltaObject, int64(len(body)))
	out = append(out, ofsDistBytes(dist)...)
	return append(out, zdeflate(body)...)
}

func refEntry(base plumbing.Hash, body []byte) []byte {
	out := entryHeadBytes(plumbing.REFDeltaObject, int64(len(body)))
	out = append(out, base.Bytes()...)
	return append(out, zdeflate(body)...)
}

func deltaBytes(t *testing.T, base, target []byte) []byte {
	t.Helper()
	st := memory.NewStorage()
	b := mkPPObj(t, st, plumbing.BlobObject, base)
	o := mkPPObj(t, st, plumbing.BlobObject, target)
	d, err := GetDelta(b, o)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	r, err := d.Reader()
	if err != nil {
		t.Fatalf("delta Reader: %v", err)
	}
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read delta: %v", err)
	}
	return body
}

// emitPack encodes the given objects in the given order.
func emitPack(t *testing.T, st *memory.Storage, useRef bool, objs ...*ObjectToPack) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := NewEncoder(&buf, st, useRef, WithObjectSelector(&ppSelector{objs: objs}))
	if _, err := enc.Encode(nil, 10); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.Bytes()
}

type ppSelector struct{ objs []*ObjectToPack }

func (s *ppSelector) ObjectsToPack([]plumbing.Hash, uint) ([]*ObjectToPack, error) {
	return s.objs, nil
}

// countingStore counts committed object writes: RawObjectWriter closes and
// direct SetEncodedObject calls.
type countingStore struct {
	*memory.Storage
	writes int
}

func (c *countingStore) SetEncodedObject(o plumbing.EncodedObject) (plumbing.Hash, error) {
	c.writes++
	return c.Storage.SetEncodedObject(o)
}

func (c *countingStore) RawObjectWriter(typ plumbing.ObjectType, sz int64) (io.WriteCloser, error) {
	w, err := c.Storage.RawObjectWriter(typ, sz)
	return &countingWriter{w, c}, err
}

type countingWriter struct {
	io.WriteCloser
	c *countingStore
}

func (w *countingWriter) Close() error {
	w.c.writes++
	return w.WriteCloser.Close()
}

// lowMemStore claims low-memory capability over an in-memory backend.
type lowMemStore struct{ *memory.Storage }

func (lowMemStore) LowMemoryMode() bool { return true }

type recObserver struct {
	events []string
	errAt  int // index at which to fail, -1 = never
	err    error
}

func (o *recObserver) fail() error {
	if o.errAt >= 0 && len(o.events) >= o.errAt {
		if o.err == nil {
			o.err = errors.New("observer abort")
		}
		return o.err
	}
	return nil
}

func (o *recObserver) OnHeader(count uint32) error {
	o.events = append(o.events, "header")
	return o.fail()
}
func (o *recObserver) OnInflatedObjectHeader(t plumbing.ObjectType, sz, pos int64) error {
	o.events = append(o.events, "objhdr")
	return o.fail()
}
func (o *recObserver) OnInflatedObjectContent(h plumbing.Hash, pos int64, crc uint32, c []byte) error {
	o.events = append(o.events, "content")
	return o.fail()
}
func (o *recObserver) OnFooter(h plumbing.Hash) error {
	o.events = append(o.events, "footer")
	return o.fail()
}

func storedContent(t *testing.T, st *memory.Storage, h plumbing.Hash) []byte {
	t.Helper()
	o, err := st.EncodedObject(plumbing.AnyObject, h)
	if err != nil {
		t.Fatalf("fetch %v: %v", h, err)
	}
	r, err := o.Reader()
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return b
}

// TestDetail01: a Parser is single-shot; a second Parse returns the consumed
// sentinel even when the first call failed.
func TestDetail01(t *testing.T) {
	st := memory.NewStorage()
	a := mkPPObj(t, st, plumbing.BlobObject, []byte("payload"))
	pack := emitPack(t, st, false, newObjectToPack(a))

	p := NewParser(bytes.NewReader(pack))
	if _, err := p.Parse(); err != nil {
		t.Fatalf("first Parse: %v", err)
	}
	if _, err := p.Parse(); !errors.Is(err, ErrParserConsumed) {
		t.Fatalf("second Parse: %v, want ErrParserConsumed", err)
	}

	p2 := NewParser(bytes.NewReader([]byte("not a pack at all")))
	if _, err := p2.Parse(); err == nil {
		t.Fatal("expected error on garbage input")
	}
	if _, err := p2.Parse(); !errors.Is(err, ErrParserConsumed) {
		t.Fatalf("second Parse after failure: %v, want ErrParserConsumed", err)
	}
}

// TestDetail02: deltas are queued during the scan and resolved afterwards —
// a delta listed before its base still resolves.
func TestDetail02(t *testing.T) {
	st := memory.NewStorage()
	bc := bytes.Repeat([]byte("a"), 50)
	tc := bytes.Repeat([]byte("b"), 50)
	base := mkPPObj(t, st, plumbing.BlobObject, bc)
	orig := mkPPObj(t, st, plumbing.BlobObject, tc)

	// ref-delta first, base second: only a queue-then-resolve parser links it.
	pack := assemblePack(
		refEntry(base.Hash(), deltaBytes(t, bc, tc)),
		fullEntry(plumbing.BlobObject, bc),
	)

	out := memory.NewStorage()
	if _, err := NewParser(bytes.NewReader(pack), WithStorage(out)).Parse(); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := storedContent(t, out, orig.Hash()); !bytes.Equal(got, tc) {
		t.Fatal("delta listed before its base did not resolve")
	}
}

// TestDetail03: REF-delta children and OFS-delta children are advanced
// together at each parent — a REF delta whose base is an OFS delta resolves.
func TestDetail03(t *testing.T) {
	st := memory.NewStorage()
	bc := bytes.Repeat([]byte("x"), 40)
	c1 := bytes.Repeat([]byte("y"), 40)
	c2 := bytes.Repeat([]byte("z"), 40)
	mid := mkPPObj(t, st, plumbing.BlobObject, c1)
	target := mkPPObj(t, st, plumbing.BlobObject, c2)

	// [base][ofs-delta -> c1][ref-delta -> c2, base hash = hash(c1)]
	e0 := fullEntry(plumbing.BlobObject, bc)
	e1body := deltaBytes(t, bc, c1)
	e1 := ofsEntry(int64(len(e0)), e1body) // base is immediately before
	e2body := deltaBytes(t, c1, c2)
	e2 := refEntry(mid.Hash(), e2body)     // ref base is the resolved c1 hash
	pack := assemblePack(e0, e1, e2)

	out := memory.NewStorage()
	if _, err := NewParser(bytes.NewReader(pack), WithStorage(out)).Parse(); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := storedContent(t, out, target.Hash()); !bytes.Equal(got, c2) {
		t.Fatal("REF delta on an OFS-delta base did not resolve")
	}
}

// TestDetail04: a REF delta whose base hash is not in the pack becomes an
// external reference — it resolves if storage holds the base, fails if not.
func TestDetail04(t *testing.T) {
	st := memory.NewStorage()
	bc := bytes.Repeat([]byte("q"), 40)
	tc := bytes.Repeat([]byte("r"), 40)
	extBase := mkPPObj(t, st, plumbing.BlobObject, bc)

	e0 := fullEntry(plumbing.BlobObject, []byte("filler"))
	e1 := refEntry(extBase.Hash(), deltaBytes(t, bc, tc))
	pack := assemblePack(e0, e1)

	withBase := memory.NewStorage()
	mkPPObj(t, withBase, plumbing.BlobObject, bc)
	if _, err := NewParser(bytes.NewReader(pack), WithStorage(withBase)).Parse(); err != nil {
		t.Fatalf("thin-pack Parse with base in storage: %v", err)
	}
	if got := storedContent(t, withBase, extBase.Hash()); !bytes.Equal(got, bc) {
		t.Fatal("external base missing from storage")
	}

	if _, err := NewParser(bytes.NewReader(pack), WithStorage(memory.NewStorage())).Parse(); err == nil {
		t.Fatal("expected error for unresolvable external ref-delta")
	}
}

// TestDetail05: an OFS delta whose recorded base offset matches no in-pack
// object is rejected — there is no external fallback for offsets.
func TestDetail05(t *testing.T) {
	bc := bytes.Repeat([]byte("m"), 40)
	e0 := fullEntry(plumbing.BlobObject, bc)
	// distance chosen so the "base" lands inside entry 0's data, never on an
	// entry boundary.
	bogus := ofsEntry(int64(len(e0))-3, insDeltaBody(int64(len(bc)), []byte("zz")))
	pack := assemblePack(e0, bogus)

	if _, err := NewParser(bytes.NewReader(pack), WithStorage(memory.NewStorage())).Parse(); err == nil {
		t.Fatal("expected error for OFS delta with unmatched base offset")
	}
}

// TestDetail06: delta chains deeper than maxDeltaChainDepth are rejected.
func TestDetail06(t *testing.T) {
	bc := bytes.Repeat([]byte("d"), 30)
	body := insDeltaBody(int64(len(bc)), bytes.Repeat([]byte("e"), 30))

	var entries [][]byte
	off := int64(12) // PACK header size
	var prevOff int64
	for i := 0; i <= maxDeltaChainDepth+5; i++ {
		var e []byte
		if i == 0 {
			e = fullEntry(plumbing.BlobObject, bc)
		} else {
			e = ofsEntry(off-prevOff, body)
		}
		entries = append(entries, e)
		prevOff = off
		off += int64(len(e))
	}
	pack := assemblePack(entries...)

	if _, err := NewParser(bytes.NewReader(pack), WithStorage(memory.NewStorage())).Parse(); err == nil {
		t.Fatal("expected error for chain deeper than maxDeltaChainDepth")
	}
}

// TestDetail07: low-memory mode requires BOTH a LowMemoryCapable storage and
// a seekable source — with a non-seekable source it disables silently and the
// parse still resolves correctly by buffering.
func TestDetail07(t *testing.T) {
	st := memory.NewStorage()
	tc := bytes.Repeat([]byte("b"), 40)
	base := mkPPObj(t, st, plumbing.BlobObject, bytes.Repeat([]byte("a"), 40))
	orig := mkPPObj(t, st, plumbing.BlobObject, tc)
	d, err := GetDelta(base, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	botp := newObjectToPack(base)
	dotp := newDeltaObjectToPack(botp, orig, d)
	pack := emitPack(t, st, false, botp, dotp)

	for _, store := range []storer.EncodedObjectStorer{
		lowMemStore{memory.NewStorage()}, memory.NewStorage(), nil,
	} {
		nonSeek := struct{ io.Reader }{bytes.NewReader(pack)}
		var opts []ParserOption
		if store != nil {
			opts = append(opts, WithStorage(store))
		}
		if _, err := NewParser(nonSeek, opts...).Parse(); err != nil {
			t.Fatalf("non-seekable Parse (storage=%T): %v", store, err)
		}
	}
}

// TestDetail08: under a low-memory-capable storage + seekable source the
// parser still resolves delta content correctly (buffer release is internal;
// observable contract is correctness).
func TestDetail08(t *testing.T) {
	st := memory.NewStorage()
	bc := bytes.Repeat([]byte("a"), 40)
	tc := bytes.Repeat([]byte("c"), 40)
	base := mkPPObj(t, st, plumbing.BlobObject, bc)
	orig := mkPPObj(t, st, plumbing.BlobObject, tc)
	d, err := GetDelta(base, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	botp := newObjectToPack(base)
	dotp := newDeltaObjectToPack(botp, orig, d)
	pack := emitPack(t, st, false, botp, dotp)

	out := lowMemStore{memory.NewStorage()}
	if _, err := NewParser(bytes.NewReader(pack), WithStorage(out)).Parse(); err != nil {
		t.Fatalf("low-memory Parse: %v", err)
	}
	if got := storedContent(t, out.Storage, orig.Hash()); !bytes.Equal(got, tc) {
		t.Fatal("low-memory mode produced wrong delta content")
	}
}

// TestDetail09: a source yielding zero objects before EOF reports the
// empty-packfile sentinel, not raw io.EOF.
func TestDetail09(t *testing.T) {
	_, err := NewParser(bytes.NewReader(nil)).Parse()
	if err == nil {
		t.Fatal("expected error on empty input")
	}
	if errors.Is(err, io.EOF) && !errors.Is(err, ErrEmptyPackfile) {
		t.Fatalf("raw io.EOF leaked: %v", err)
	}
	if !errors.Is(err, ErrEmptyPackfile) {
		t.Fatalf("got %v, want ErrEmptyPackfile", err)
	}
}

// TestDetail10: growHint is clamped to the kept 1 GiB bound.
func TestDetail10(t *testing.T) {
	if got := growHint(1 << 40); got > maxObjectPreallocBytes {
		t.Fatalf("growHint(1<<40)=%d exceeds bound", got)
	}
	if got := growHint(64); got != 64 {
		t.Fatalf("growHint(64)=%d", got)
	}
	if got := growHint(-5); got != 0 {
		t.Fatalf("growHint(-5)=%d, want 0", got)
	}
}

// TestDetail11: a resolved delta is retrievable under the hash of its
// resolved contents (the pre-recorded-ID path is internal; the observable
// shape is that the stored object answers to its content hash).
func TestDetail11(t *testing.T) {
	st := memory.NewStorage()
	base := mkPPObj(t, st, plumbing.BlobObject, bytes.Repeat([]byte("a"), 40))
	tc := bytes.Repeat([]byte("t"), 40)
	orig := mkPPObj(t, st, plumbing.BlobObject, tc)
	d, err := GetDelta(base, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	botp := newObjectToPack(base)
	dotp := newDeltaObjectToPack(botp, orig, d)
	pack := emitPack(t, st, false, botp, dotp)

	out := memory.NewStorage()
	if _, err := NewParser(bytes.NewReader(pack), WithStorage(out)).Parse(); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := storedContent(t, out, orig.Hash()); !bytes.Equal(got, tc) {
		t.Fatal("resolved delta not stored under its content hash")
	}
}

// TestDetail12: an external-ref parent's real type and size come from the
// fetched object — the resolved delta inherits the base's type.
func TestDetail12(t *testing.T) {
	st := memory.NewStorage()
	bc := []byte("tree-ish base content padded out")
	tc := bytes.Repeat([]byte("u"), len(bc))
	extBase := mkPPObj(t, st, plumbing.TreeObject, bc)

	e0 := fullEntry(plumbing.BlobObject, []byte("filler"))
	e1 := refEntry(extBase.Hash(), deltaBytes(t, bc, tc))
	pack := assemblePack(e0, e1)

	out := memory.NewStorage()
	mkPPObj(t, out, plumbing.TreeObject, bc)
	if _, err := NewParser(bytes.NewReader(pack), WithStorage(out)).Parse(); err != nil {
		t.Fatalf("thin-pack Parse: %v", err)
	}
	got, err := out.EncodedObject(plumbing.AnyObject, extBase.Hash())
	if err != nil {
		t.Fatalf("fetch base: %v", err)
	}
	if got.Type() != plumbing.TreeObject {
		t.Fatalf("external base type=%v, want TreeObject", got.Type())
	}
	// resolved delta must carry the base's real type
	res, err := out.EncodedObject(plumbing.AnyObject,
		mkPPObj(t, memory.NewStorage(), plumbing.TreeObject, tc).Hash())
	if err != nil {
		t.Fatalf("resolved delta fetch: %v", err)
	}
	if res.Type() != plumbing.TreeObject {
		t.Fatalf("resolved type=%v, want base's TreeObject", res.Type())
	}
}

// TestDetail13: parserCache.Reset caps its reservation at maxObjectsPrealloc
// even when the advertised count is far larger.
func TestDetail13(t *testing.T) {
	c := newParserCache()
	c.Reset(1 << 30)
	if cap(c.oi) > maxObjectsPrealloc {
		t.Fatalf("oi capacity %d exceeds maxObjectsPrealloc", cap(c.oi))
	}
}

// TestDetail14: each pack entry is stored exactly once — non-deltas by the
// scan, resolved deltas by the parser (no double writes).
func TestDetail14(t *testing.T) {
	st := memory.NewStorage()
	b1 := mkPPObj(t, st, plumbing.BlobObject, []byte("one"))
	b2 := mkPPObj(t, st, plumbing.BlobObject, bytes.Repeat([]byte("a"), 40))
	orig := mkPPObj(t, st, plumbing.BlobObject, bytes.Repeat([]byte("b"), 40))
	d, err := GetDelta(b2, orig)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	b2otp := newObjectToPack(b2)
	dotp := newDeltaObjectToPack(b2otp, orig, d)
	pack := emitPack(t, st, false,
		newObjectToPack(b1), b2otp, dotp)

	out := &countingStore{Storage: memory.NewStorage()}
	if _, err := NewParser(bytes.NewReader(pack), WithStorage(out)).Parse(); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if out.writes != 3 {
		t.Fatalf("SetEncodedObject called %d times for 3 pack entries", out.writes)
	}
}

// TestDetail15: observer callbacks fire in section order — header, per-object
// header+content, footer — and the first observer error aborts the parse.
func TestDetail15(t *testing.T) {
	st2 := memory.NewStorage()
	a := mkPPObj(t, st2, plumbing.BlobObject, []byte("x"))
	b := mkPPObj(t, st2, plumbing.BlobObject, []byte("y"))
	pack := emitPack(t, st2, false, newObjectToPack(a), newObjectToPack(b))

	obs := &recObserver{errAt: -1}
	if _, err := NewParser(bytes.NewReader(pack),
		WithStorage(memory.NewStorage()), WithScannerObservers(obs)).Parse(); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{"header", "objhdr", "content", "objhdr", "content", "footer"}
	if len(obs.events) != len(want) {
		t.Fatalf("events=%v, want %v", obs.events, want)
	}
	for i, w := range want {
		if obs.events[i] != w {
			t.Fatalf("events=%v, want %v", obs.events, want)
		}
	}

	sentinel := errors.New("observer stop")
	obs2 := &recObserver{errAt: 3, err: sentinel} // fail on 2nd objhdr
	if _, err := NewParser(bytes.NewReader(pack),
		WithStorage(memory.NewStorage()), WithScannerObservers(obs2)).Parse(); !errors.Is(err, sentinel) {
		t.Fatalf("Parse: %v, want observer error", err)
	}
	if len(obs2.events) >= len(want) {
		t.Fatalf("events continued past abort: %v", obs2.events)
	}
}
