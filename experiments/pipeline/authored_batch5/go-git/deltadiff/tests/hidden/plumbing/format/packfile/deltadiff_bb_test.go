package packfile

import (
	"bytes"
	"errors"
	"io"
	"math/bits"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	packutil "example.internal/gitkit/v6/plumbing/format/packfile/util"
	"example.internal/gitkit/v6/storage/memory"
)

// randBytes supplies the helper removed with the in-tree test files; kept
// tests still reference it.
func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*31 + i>>4)
	}
	return b
}

// ddObj wraps bytes as a storable EncodedObject.
func ddObj(t *testing.T, st *memory.Storage, content []byte) plumbing.EncodedObject {
	t.Helper()
	o := st.NewEncodedObject()
	o.SetType(plumbing.BlobObject)
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
	return o
}

// op is one decoded delta instruction.
type op struct {
	insert bool
	lit    []byte
	off    int
	sz     int
}

// decodeOps parses a delta stream: two LEB128 sizes then instructions.
func decodeOps(t *testing.T, delta []byte) (srcSz, tgtSz int, ops []op) {
	t.Helper()
	s, rest, err := packutil.DecodeLEB128(delta)
	if err != nil {
		t.Fatalf("src size: %v", err)
	}
	srcSz = int(s)
	g, rest, err := packutil.DecodeLEB128(rest)
	if err != nil {
		t.Fatalf("tgt size: %v", err)
	}
	tgtSz = int(g)
	for len(rest) > 0 {
		b := rest[0]
		rest = rest[1:]
		if b&0x80 != 0 {
			var o op
			shift := uint(0)
			for i := 0; i < 4; i++ {
				if b&(1<<uint(i)) != 0 {
					o.off |= int(rest[0]) << shift
					rest = rest[1:]
				}
				shift += 8
			}
			shift = 0
			for i := 4; i < 7; i++ {
				if b&(1<<uint(i)) != 0 {
					o.sz |= int(rest[0]) << shift
					rest = rest[1:]
				}
				shift += 8
			}
			if o.sz == 0 {
				o.sz = 0x10000
			}
			ops = append(ops, o)
		} else {
			if b == 0 {
				t.Fatal("zero opcode")
			}
			n := int(b)
			if len(rest) < n {
				t.Fatal("insert overruns delta")
			}
			ops = append(ops, op{insert: true, lit: rest[:n]})
			rest = rest[n:]
		}
	}
	return srcSz, tgtSz, ops
}

func roundTrip(t *testing.T, src, tgt []byte) []byte {
	t.Helper()
	delta := DiffDelta(src, tgt)
	out, err := PatchDelta(src, delta)
	if err != nil {
		t.Fatalf("PatchDelta: %v", err)
	}
	if !bytes.Equal(out, tgt) {
		t.Fatal("round trip mismatch")
	}
	return delta
}

// pseudo-random byte blocks, deterministic
func prb(seed byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = seed*31 + byte(i)*7 + byte(i>>3)
	}
	return b
}

// TestDetail01: the stream opens with source size then target size, each
// little-endian LEB128, before any instruction.
func TestDetail01(t *testing.T) {
	src, tgt := prb(1, 200), prb(2, 90)
	delta := roundTrip(t, src, tgt)
	srcSz, tgtSz, ops := decodeOps(t, delta)
	if srcSz != len(src) || tgtSz != len(tgt) {
		t.Fatalf("sizes=(%d,%d), want (%d,%d)", srcSz, tgtSz, len(src), len(tgt))
	}
	if len(ops) == 0 {
		t.Fatal("no instructions")
	}
}

// TestDetail02: insert instructions carry a literal length byte (high bit
// clear) chunked at 127 — a longer literal run is split, never merged.
func TestDetail02(t *testing.T) {
	// source below the block size forces an all-literal target
	src, tgt := prb(9, 8), prb(3, 300)
	delta := roundTrip(t, src, tgt)
	_, _, ops := decodeOps(t, delta)
	var lit []byte
	inserts := 0
	for _, o := range ops {
		if !o.insert {
			t.Fatalf("copy op in all-literal delta: %+v", o)
		}
		if len(o.lit) > 127 {
			t.Fatalf("insert op of %d bytes exceeds 127", len(o.lit))
		}
		inserts++
		lit = append(lit, o.lit...)
	}
	if inserts < 3 {
		t.Fatalf("300-byte literal in %d inserts, want >=3", inserts)
	}
	if !bytes.Equal(lit, tgt) {
		t.Fatal("literal stream mismatch")
	}
}

// TestDetail03: copy instructions set the high bit and pack offset/length in
// flag-selected bytes — a decoded copy lands exactly where the match is.
func TestDetail03(t *testing.T) {
	src := prb(5, 400)
	tgt := append(append([]byte("head-"), src[100:250]...), []byte("-tail")...)
	delta := roundTrip(t, src, tgt)
	_, _, ops := decodeOps(t, delta)
	var sawCopy bool
	for _, o := range ops {
		if o.insert {
			continue
		}
		sawCopy = true
		if o.off < 0 || o.sz <= 0 {
			t.Fatalf("bad copy %+v", o)
		}
		if !bytes.Equal(src[o.off:o.off+o.sz], tgt) && o.off+o.sz > len(src) {
			t.Fatalf("copy out of range %+v", o)
		}
	}
	if !sawCopy {
		t.Fatal("no copy op emitted for a shared block")
	}
}

// TestDetail04: a copy longer than the 64KB opcode ceiling is emitted as
// repeated maxCopySize copies advancing the offset.
func TestDetail04(t *testing.T) {
	src := prb(7, 100*1024)
	delta := roundTrip(t, src, bytes.Clone(src))
	_, _, ops := decodeOps(t, delta)
	var copies int
	var total int
	for _, o := range ops {
		if o.insert {
			continue
		}
		copies++
		total += o.sz
		if o.sz > maxCopySize {
			t.Fatalf("copy of %d exceeds %d", o.sz, maxCopySize)
		}
	}
	if copies < 2 {
		t.Fatalf("100KB match in %d copies, want >=2", copies)
	}
	if total < len(src) {
		t.Fatalf("copied %d of %d bytes", total, len(src))
	}
}

// TestDetail05: a match shorter than the 16-byte fingerprint block is emitted
// as literal insert bytes — the sub-block tail is never indexed.
func TestDetail05(t *testing.T) {
	src := prb(11, 128)
	tgt := append(prb(12, 40), src[:8]...) // only an 8-byte shared run
	delta := roundTrip(t, src, tgt)
	_, _, ops := decodeOps(t, delta)
	for _, o := range ops {
		if !o.insert {
			t.Fatalf("copy op emitted for a sub-block match: %+v", o)
		}
	}
}

// TestDetail06: when the source is smaller than the block size the whole
// target falls back to literals; findMatch signals it with a negative length.
func TestDetail06(t *testing.T) {
	src, tgt := prb(9, 8), prb(3, 64)
	var idx deltaIndex
	idx.init(src)
	_, l := idx.findMatch(src, tgt, 0)
	if l >= 0 {
		t.Fatalf("findMatch len=%d, want negative for tiny source", l)
	}
	// and the encoder emits all literals (already covered by roundTrip)
	roundTrip(t, src, tgt)
}

// TestDetail07: fewer than a block's bytes remaining in the target returns
// the remaining length directly.
func TestDetail07(t *testing.T) {
	src := prb(5, 128)
	tgt := append(prb(6, 50), prb(6, 10)...)
	var idx deltaIndex
	idx.init(src)
	_, l := idx.findMatch(src, tgt, len(tgt)-10)
	if l != 10 {
		t.Fatalf("findMatch tail len=%d, want 10", l)
	}
}

// TestDetail08: pending literals are flushed before each copy — in the op
// stream an insert always precedes the copy it feeds.
func TestDetail08(t *testing.T) {
	blk1, blk2 := prb(5, 16), prb(6, 16)
	lit := prb(20, 20)
	src := append(blk1, blk2...)
	tgt := append(append(append(prb(21, 20), blk1...), lit...), blk2...)
	delta := roundTrip(t, src, tgt)
	_, _, ops := decodeOps(t, delta)
	// ops must reproduce the target IN ORDER: pending literals appear as one
	// insert immediately before the copy that follows them — a flush ordered
	// after a copy would break sequential reconstruction.
	var rebuilt []byte
	for _, o := range ops {
		if o.insert {
			rebuilt = append(rebuilt, o.lit...)
			continue
		}
		rebuilt = append(rebuilt, src[o.off:o.off+o.sz]...)
	}
	if !bytes.Equal(rebuilt, tgt) {
		t.Fatal("ops do not reproduce target in order")
	}
	// and at least one literal run was actually flushed ahead of a copy
	var kinds []bool
	for _, o := range ops {
		kinds = append(kinds, o.insert)
	}
	sawInsertThenCopy := false
	for i := 0; i+1 < len(kinds); i++ {
		if kinds[i] && !kinds[i+1] {
			sawInsertThenCopy = true
		}
	}
	if !sawInsertThenCopy {
		t.Fatalf("no insert->copy pair in op stream: %v", kinds)
	}
}

// TestDetail09: the index scans the source backward block by block — the
// chain head after init is the last block scanned, i.e. the LOWEST offset
// among identical non-consecutive blocks. (The consecutive-collapse rule is
// internal; the observable shape is which occurrence findMatch returns.)
func TestDetail09(t *testing.T) {
	blk := prb(8, 16)
	src := append(append(append(blk, prb(9, 16)...), blk...), prb(10, 16)...)
	var idx deltaIndex
	idx.init(src)
	tgt := append(prb(11, 24), blk...)
	off, l := idx.findMatch(src, tgt, len(tgt)-16)
	if l < 16 {
		t.Fatalf("no match for repeated block: l=%d", l)
	}
	if !bytes.Equal(src[off:off+16], blk) {
		t.Fatalf("match at %d does not equal the block", off)
	}
	if off != 0 {
		t.Fatalf("match offset=%d, want earliest occurrence 0 (backward scan)", off)
	}
}

// TestDetail10: hash chains longer than the cap are severed — a source of
// many identical blocks still indexes and matches correctly.
func TestDetail10(t *testing.T) {
	blk := prb(13, 16)
	src := bytes.Repeat(blk, 100) // 100 identical blocks >> 64 chain cap
	var idx deltaIndex
	idx.init(src)
	tgt := append(prb(14, 24), blk...)
	off, l := idx.findMatch(src, tgt, len(tgt)-16)
	if l < 16 || !bytes.Equal(src[off:off+16], blk) {
		t.Fatalf("severed-chain match wrong: off=%d l=%d", off, l)
	}
}

// TestDetail11: the table size is the next power of two at least the
// worst-case block count.
func TestDetail11(t *testing.T) {
	for _, n := range []int{1, 3, 5, 16, 17, 100, 255, 256, 257} {
		got := tableSize(n)
		if got < n || got&(got-1) != 0 {
			t.Fatalf("tableSize(%d)=%d: not a power of two >= n", n, got)
		}
	}
	for _, x := range []uint32{1, 0x8000, 0xffffffff, 0x00000001, 0x00f00000} {
		if leadingZeros(x) != bits.LeadingZeros32(x) {
			t.Fatalf("leadingZeros(%x) wrong", x)
		}
	}
}

// TestDetail12: GetDelta returns a MemoryObject typed OFSDeltaObject whose
// size is the delta byte length.
func TestDetail12(t *testing.T) {
	st := memory.NewStorage()
	base := ddObj(t, st, prb(5, 200))
	target := ddObj(t, st, prb(6, 120))
	d, err := GetDelta(base, target)
	if err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	if d.Type() != plumbing.OFSDeltaObject {
		t.Fatalf("type=%v, want OFSDeltaObject", d.Type())
	}
	r, err := d.Reader()
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if d.Size() != int64(len(body)) {
		t.Fatalf("size=%d, delta len=%d", d.Size(), len(body))
	}
}

// TestDetail13: reader failures propagate before any delta bytes are
// produced, and both readers are closed.
func TestDetail13(t *testing.T) {
	bad := &failObj{err: errors.New("read failure")}
	st := memory.NewStorage()
	good := ddObj(t, st, []byte("ok"))

	if _, err := GetDelta(bad, good); err == nil {
		t.Fatal("base reader failure swallowed")
	}
	if _, err := GetDelta(good, bad); err == nil {
		t.Fatal("target reader failure swallowed")
	}

	c1, c2 := &closeObj{}, &closeObj{}
	if _, err := GetDelta(c1, c2); err != nil {
		t.Fatalf("GetDelta: %v", err)
	}
	if !c1.closed || !c2.closed {
		t.Fatal("reader not closed")
	}
}

// failObj's Reader always fails.
type failObj struct{ err error }

func (o *failObj) Hash() plumbing.Hash          { return plumbing.ZeroHash }
func (o *failObj) Type() plumbing.ObjectType    { return plumbing.BlobObject }
func (o *failObj) SetType(t plumbing.ObjectType) {}
func (o *failObj) Size() int64                  { return 1 }
func (o *failObj) SetSize(int64)                {}
func (o *failObj) Reader() (io.ReadCloser, error) {
	return nil, o.err
}
func (o *failObj) Writer() (io.WriteCloser, error) { return nil, o.err }

// closeObj's Reader tracks Close.
type closeObj struct{ closed bool }

func (o *closeObj) Hash() plumbing.Hash          { return plumbing.ZeroHash }
func (o *closeObj) Type() plumbing.ObjectType    { return plumbing.BlobObject }
func (o *closeObj) SetType(t plumbing.ObjectType) {}
func (o *closeObj) Size() int64                  { return 4 }
func (o *closeObj) SetSize(int64)                {}
func (o *closeObj) Reader() (io.ReadCloser, error) {
	return &trackRC{Reader: bytes.NewReader([]byte("data")), o: o}, nil
}
func (o *closeObj) Writer() (io.WriteCloser, error) {
	return nil, errors.New("no write")
}

type trackRC struct {
	io.Reader
	o *closeObj
}

func (r *trackRC) Close() error {
	r.o.closed = true
	return nil
}
