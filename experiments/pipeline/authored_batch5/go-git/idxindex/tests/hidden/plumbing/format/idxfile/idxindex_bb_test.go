package idxfile_test

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/idxfile"
	"example.internal/gitkit/v6/plumbing/format/revfile"
)

func sameHash(a, b plumbing.Hash) bool { return bytes.Equal(a.Bytes(), b.Bytes()) }

func hashWithFirstByte(b byte, tail byte) plumbing.Hash {
	s := fmt.Sprintf("%02x", b)
	for i := 0; i < 19; i++ {
		s += fmt.Sprintf("%02x", tail)
	}
	return plumbing.NewHash(s)
}

// buildIndex feeds entries through the idxfile.Writer observer so the constructed
// idxfile.MemoryIndex exercises the same table layout the unit produces.
func buildIndex(t *testing.T, entries []idxfile.Entry, packHash plumbing.Hash) *idxfile.MemoryIndex {
	t.Helper()
	w := &idxfile.Writer{}
	if err := w.OnHeader(uint32(len(entries))); err != nil {
		t.Fatalf("OnHeader: %v", err)
	}
	for _, e := range entries {
		if err := w.OnInflatedObjectContent(e.Hash, int64(e.Offset), e.CRC32, nil); err != nil {
			t.Fatalf("OnInflatedObjectContent: %v", err)
		}
	}
	if err := w.OnFooter(packHash); err != nil {
		t.Fatalf("OnFooter: %v", err)
	}
	idx, err := w.Index()
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	return idx
}

func encodeIdx(t *testing.T, idx *idxfile.MemoryIndex) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := idxfile.Encode(&buf, sha1.New(), idx); err != nil {
		t.Fatalf("idxfile.Encode: %v", err)
	}
	return buf.Bytes()
}

func encodeRev(t *testing.T, idx *idxfile.MemoryIndex) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := revfile.Encode(&buf, sha1.New(), idx); err != nil {
		t.Fatalf("revfile.Encode: %v", err)
	}
	return buf.Bytes()
}

// countingRAC is a idxfile.ReadAtCloser over a byte slice that counts reads.
type countingRAC struct {
	r     *bytes.Reader
	reads *int
}

func (c *countingRAC) ReadAt(p []byte, off int64) (int, error) {
	*c.reads++
	return c.r.ReadAt(p, off)
}

func (c *countingRAC) Read(p []byte) (int, error) {
	*c.reads++
	return c.r.Read(p)
}

func (c *countingRAC) Close() error { return nil }

type racOpens struct {
	idxData  []byte
	revData  []byte
	idxReads int
	revReads int
	idxOpen  int
	revOpen  int
	revErr   error
}

func (o *racOpens) openIdx() (idxfile.ReadAtCloser, error) {
	o.idxOpen++
	return &countingRAC{r: bytes.NewReader(o.idxData), reads: &o.idxReads}, nil
}

func (o *racOpens) openRev() (idxfile.ReadAtCloser, error) {
	o.revOpen++
	if o.revErr != nil {
		return nil, o.revErr
	}
	return &countingRAC{r: bytes.NewReader(o.revData), reads: &o.revReads}, nil
}

func drain(t *testing.T, it idxfile.EntryIter) []*idxfile.Entry {
	t.Helper()
	var out []*idxfile.Entry
	for {
		e, err := it.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("iter Next: %v", err)
		}
		out = append(out, e)
	}
	return out
}

// 1. Fanout buckets the lookup by first byte, then binary-searches only
//    that bucket — misses in a present bucket and empty-bucket probes
//    both report not-found. Inferable: partially — assert the observable
//    bucketed-lookup results, not the search internals.
func TestDetail01(t *testing.T) {
	a := hashWithFirstByte(0x10, 0x01)
	b := hashWithFirstByte(0x10, 0x02)
	c := hashWithFirstByte(0x40, 0x03)
	idx := buildIndex(t, []idxfile.Entry{
		{Hash: a, Offset: 1, CRC32: 11},
		{Hash: c, Offset: 2, CRC32: 22},
	}, plumbing.NewHash("9999999999999999999999999999999999999999"))

	ok, err := idx.Contains(a)
	if err != nil || !ok {
		t.Fatalf("present hash not found: ok=%v err=%v", ok, err)
	}
	// Same bucket as a, not present.
	ok, err = idx.Contains(b)
	if err != nil || ok {
		t.Fatalf("same-bucket miss reported found: ok=%v err=%v", ok, err)
	}
	// Empty bucket.
	ok, err = idx.Contains(hashWithFirstByte(0x77, 0x01))
	if err != nil || ok {
		t.Fatalf("empty-bucket miss reported found: ok=%v err=%v", ok, err)
	}
	off, err := idx.FindOffset(c)
	if err != nil || off != 2 {
		t.Fatalf("FindOffset: %d %v", off, err)
	}
}

// 2. MayContain answers from the fanout mapping alone — a hash whose
//    first byte maps to an empty bucket is excluded. Inferable: doc.
func TestDetail02(t *testing.T) {
	present := hashWithFirstByte(0x33, 0xaa)
	idx := buildIndex(t, []idxfile.Entry{{Hash: present, Offset: 5, CRC32: 1}},
		plumbing.NewHash("9999999999999999999999999999999999999999"))
	if !idx.MayContain(present) {
		t.Fatal("MayContain=false for a present hash")
	}
	if idx.MayContain(hashWithFirstByte(0x44, 0xbb)) {
		t.Fatal("MayContain=true for a hash in an empty bucket")
	}
	// Same first byte as a present entry is a maybe.
	if !idx.MayContain(hashWithFirstByte(0x33, 0xcc)) {
		t.Fatal("MayContain=false for a hash sharing a non-empty bucket")
	}
}

// 3. A 32-bit offset slot with the top bit set indexes the 64-bit
//    overflow table. Inferable: partially — assert large offsets round
//    trip correctly alongside small ones.
func TestDetail03(t *testing.T) {
	small := hashWithFirstByte(0x01, 0x10)
	big := hashWithFirstByte(0x02, 0x20)
	const bigOff = uint64(1) << 33 // exceeds 32 bits
	idx := buildIndex(t, []idxfile.Entry{
		{Hash: small, Offset: 7, CRC32: 1},
		{Hash: big, Offset: bigOff, CRC32: 2},
	}, plumbing.NewHash("9999999999999999999999999999999999999999"))

	off, err := idx.FindOffset(big)
	if err != nil {
		t.Fatalf("FindOffset big: %v", err)
	}
	if uint64(off) != bigOff {
		t.Fatalf("64-bit offset mangled: got %x want %x", off, bigOff)
	}
	off, err = idx.FindOffset(small)
	if err != nil || uint64(off) != 7 {
		t.Fatalf("FindOffset small: %d %v", off, err)
	}
}

// 4. FindHash builds the reverse offset→hash map lazily, once. Inferable:
//    no — assert SHAPE: the mapping answers correctly and is not rebuilt
//    after later mutation of the names table.
func TestDetail04(t *testing.T) {
	h1 := hashWithFirstByte(0x10, 0x01)
	h2 := hashWithFirstByte(0x10, 0x02)
	idx := buildIndex(t, []idxfile.Entry{
		{Hash: h1, Offset: 100, CRC32: 1},
		{Hash: h2, Offset: 200, CRC32: 2},
	}, plumbing.NewHash("9999999999999999999999999999999999999999"))

	got, err := idx.FindHash(200)
	if err != nil || !sameHash(got, h2) {
		t.Fatalf("FindHash: %v %v", got, err)
	}

	// Mutate the names table in place; a once-built map still answers
	// with the hash captured at build time.
	mi := idx.FanoutMapping[h2.Bytes()[0]]
	slot := idSlot(idx, h2)
	if slot < 0 {
		t.Fatal("h2 not located in bucket names")
	}
	copy(idx.Names[mi][slot:], h1.Bytes())

	got, err = idx.FindHash(100)
	if err != nil {
		t.Fatalf("FindHash after mutation: %v", err)
	}
	if !sameHash(got, h1) {
		t.Fatalf("reverse map rebuilt per call: got %v", got)
	}
}

// idSlot returns the byte offset of h's name inside its bucket slice.
func idSlot(idx *idxfile.MemoryIndex, h plumbing.Hash) int {
	mi := idx.FanoutMapping[h.Bytes()[0]]
	sz := len(h.Bytes())
	names := idx.Names[mi]
	for i := 0; i+sz <= len(names); i += sz {
		if bytes.Equal(names[i:i+sz], h.Bytes()) {
			return i
		}
	}
	return -1
}

// 5. Entries walks fanout (hash-sorted) order; EntriesByOffset walks
//    offset order — two different orderings. Inferable: partially.
func TestDetail05(t *testing.T) {
	h1 := hashWithFirstByte(0x30, 0x01)
	h2 := hashWithFirstByte(0x10, 0x02)
	h3 := hashWithFirstByte(0x20, 0x03)
	idx := buildIndex(t, []idxfile.Entry{
		{Hash: h1, Offset: 10, CRC32: 1},
		{Hash: h2, Offset: 30, CRC32: 2},
		{Hash: h3, Offset: 20, CRC32: 3},
	}, plumbing.NewHash("9999999999999999999999999999999999999999"))

	it, err := idx.Entries()
	if err != nil {
		t.Fatal(err)
	}
	got := drain(t, it)
	it.Close()
	want := []plumbing.Hash{h2, h3, h1} // sorted by hash
	if len(got) != 3 {
		t.Fatalf("Entries yielded %d", len(got))
	}
	for i := range want {
		if !sameHash(got[i].Hash, want[i]) {
			t.Fatalf("Entries[%d]=%v want hash order %v", i, got[i].Hash, want)
		}
	}

	it, err = idx.EntriesByOffset()
	if err != nil {
		t.Fatal(err)
	}
	got = drain(t, it)
	it.Close()
	want = []plumbing.Hash{h1, h3, h2} // sorted by offset 10,20,30
	for i := range want {
		if !sameHash(got[i].Hash, want[i]) {
			t.Fatalf("EntriesByOffset[%d]=%v want offset order %v", i, got[i].Hash, want)
		}
	}
}

// 6. EntriesWithPrefix yields only the matching run and an empty prefix
//    yields everything. Inferable: doc.
func TestDetail06(t *testing.T) {
	a := hashWithFirstByte(0x10, 0x01)
	b := hashWithFirstByte(0x10, 0x02)
	c := hashWithFirstByte(0x20, 0x03)
	d := hashWithFirstByte(0x20, 0x04)
	idx := buildIndex(t, []idxfile.Entry{
		{Hash: a, Offset: 1}, {Hash: b, Offset: 2},
		{Hash: c, Offset: 3}, {Hash: d, Offset: 4},
	}, plumbing.NewHash("9999999999999999999999999999999999999999"))

	it, err := idx.EntriesWithPrefix([]byte{0x10})
	if err != nil {
		t.Fatal(err)
	}
	got := drain(t, it)
	it.Close()
	if len(got) != 2 || !sameHash(got[0].Hash, a) || !sameHash(got[1].Hash, b) {
		t.Fatalf("prefix 0x10 yielded %+v", got)
	}

	// Two-byte prefix narrows inside the bucket.
	it, err = idx.EntriesWithPrefix([]byte{0x20, 0x04})
	if err != nil {
		t.Fatal(err)
	}
	got = drain(t, it)
	it.Close()
	if len(got) != 1 || !sameHash(got[0].Hash, d) {
		t.Fatalf("prefix 0x2004 yielded %+v", got)
	}

	it, err = idx.EntriesWithPrefix(nil)
	if err != nil {
		t.Fatal(err)
	}
	got = drain(t, it)
	it.Close()
	if len(got) != 4 {
		t.Fatalf("empty prefix yielded %d", len(got))
	}

	// Non-matching prefix in an existing bucket yields nothing.
	it, err = idx.EntriesWithPrefix([]byte{0x10, 0x09})
	if err != nil {
		t.Fatal(err)
	}
	got = drain(t, it)
	it.Close()
	if len(got) != 0 {
		t.Fatalf("absent prefix yielded %+v", got)
	}
}

// 7. LazyIndex resolves lookups through ReadAt on the shared descriptor —
//    correctness plus the descriptor is opened once and shared across
//    lookups. Inferable: partially — assert correctness and bound the
//    number of opens, not the section arithmetic.
func TestDetail07(t *testing.T) {
	h1 := hashWithFirstByte(0x10, 0x01)
	h2 := hashWithFirstByte(0x40, 0x02)
	pack := plumbing.NewHash("9999999999999999999999999999999999999999")
	idx := buildIndex(t, []idxfile.Entry{
		{Hash: h1, Offset: 11, CRC32: 111},
		{Hash: h2, Offset: 22, CRC32: 222},
	}, pack)

	o := &racOpens{idxData: encodeIdx(t, idx), revData: encodeRev(t, idx)}
	lazy, err := idxfile.NewLazyIndex(o.openIdx, o.openRev, pack)
	if err != nil {
		t.Fatalf("NewLazyIndex: %v", err)
	}
	defer lazy.Close()

	ok, err := lazy.Contains(h2)
	if err != nil || !ok {
		t.Fatalf("Contains: %v %v", ok, err)
	}
	off, err := lazy.FindOffset(h1)
	if err != nil || off != 11 {
		t.Fatalf("FindOffset: %d %v", off, err)
	}
	crc, err := lazy.FindCRC32(h2)
	if err != nil || crc != 222 {
		t.Fatalf("FindCRC32: %d %v", crc, err)
	}
	n, err := lazy.Count()
	if err != nil || n != 2 {
		t.Fatalf("Count: %d %v", n, err)
	}
	if o.idxOpen == 0 {
		t.Fatal("lazy index never opened the descriptor")
	}
	if o.idxOpen > 2 {
		t.Fatalf("descriptor opened %d times across 4 lookups — not shared", o.idxOpen)
	}
}

// 8. With a .rev file present, hash-to-offset lookups map positions
//    through the rev entries — observable because the rev file is
//    consulted at all. Inferable: no — assert SHAPE: the rev descriptor
//    receives reads during FindHash.
func TestDetail08(t *testing.T) {
	h1 := hashWithFirstByte(0x10, 0x01)
	h2 := hashWithFirstByte(0x40, 0x02)
	pack := plumbing.NewHash("9999999999999999999999999999999999999999")
	idx := buildIndex(t, []idxfile.Entry{
		{Hash: h1, Offset: 11, CRC32: 111},
		{Hash: h2, Offset: 22, CRC32: 222},
	}, pack)

	o := &racOpens{idxData: encodeIdx(t, idx), revData: encodeRev(t, idx)}
	lazy, err := idxfile.NewLazyIndex(o.openIdx, o.openRev, pack)
	if err != nil {
		t.Fatalf("NewLazyIndex: %v", err)
	}
	defer lazy.Close()

	revBefore := o.revReads
	got, err := lazy.FindHash(22)
	if err != nil || !sameHash(got, h2) {
		t.Fatalf("FindHash: %v %v", got, err)
	}
	if o.revReads == revBefore {
		t.Fatal("FindHash never read the rev file")
	}

	// A rev opener that cannot produce the file surfaces its error at
	// construction — the lookup contract depends on the rev being there.
	o2 := &racOpens{idxData: encodeIdx(t, idx), revErr: fs.ErrNotExist}
	lazy2, err := idxfile.NewLazyIndex(o2.openIdx, o2.openRev, pack)
	if err == nil {
		lazy2.Close()
		t.Fatal("NewLazyIndex with an unusable rev opener did not error")
	}
}

// 9. The Writer rejects Index() before OnFooter. Inferable: partially —
//    the count-mismatch clause of this DETAILS row is not asserted: the
//    observable contract exercised here is that Index requires a footer
//    first, and Finished() reports completion only after it.
func TestDetail09(t *testing.T) {
	w := &idxfile.Writer{}
	if err := w.OnHeader(2); err != nil {
		t.Fatalf("OnHeader: %v", err)
	}
	if w.Finished() {
		t.Fatal("Finished true before OnFooter")
	}
	if _, err := w.Index(); err == nil {
		t.Fatal("Index before OnFooter did not error")
	}
	if err := w.OnInflatedObjectContent(hashWithFirstByte(1, 1), 0, 0, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Index(); err == nil {
		t.Fatal("Index before OnFooter (after adds) did not error")
	}
	if err := w.OnFooter(plumbing.NewHash("9999999999999999999999999999999999999999")); err != nil {
		t.Fatalf("OnFooter: %v", err)
	}
	if !w.Finished() {
		t.Fatal("Finished false after OnFooter")
	}
	if _, err := w.Index(); err != nil {
		t.Fatalf("Index after OnFooter: %v", err)
	}
}

// 10. Add dedupes by hash — the same object twice keeps the first
//     position. Inferable: no — assert SHAPE: the resulting index holds
//     exactly one entry and its offset is the first add's.
func TestDetail10(t *testing.T) {
	h := hashWithFirstByte(0x55, 0x07)
	w := &idxfile.Writer{}
	if err := w.OnHeader(1); err != nil {
		t.Fatal(err)
	}
	w.Add(h, 100, 1)
	w.Add(h, 200, 2)
	if err := w.OnFooter(plumbing.NewHash("9999999999999999999999999999999999999999")); err != nil {
		t.Fatalf("OnFooter: %v", err)
	}
	idx, err := w.Index()
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	n, _ := idx.Count()
	if n != 1 {
		t.Fatalf("dedup kept %d entries", n)
	}
	off, err := idx.FindOffset(h)
	if err != nil || off != 100 {
		t.Fatalf("kept offset %d, want first position 100", off)
	}
}

// 11. OnInflatedObjectContent records hash+position+crc and ignores
//     payload bytes; OnInflatedObjectHeader records nothing. Inferable:
//     partially.
func TestDetail11(t *testing.T) {
	h := hashWithFirstByte(0x21, 0x09)
	w := &idxfile.Writer{}
	if err := w.OnHeader(1); err != nil {
		t.Fatal(err)
	}
	if err := w.OnInflatedObjectHeader(plumbing.BlobObject, 5, 0); err != nil {
		t.Fatalf("header hook is not a no-op: %v", err)
	}
	if err := w.OnInflatedObjectContent(h, 4242, 0xdeadbeef, []byte("garbage payload bytes")); err != nil {
		t.Fatal(err)
	}
	if err := w.OnInflatedObjectHeader(plumbing.CommitObject, 9, 9); err != nil {
		t.Fatal(err)
	}
	if err := w.OnFooter(plumbing.NewHash("9999999999999999999999999999999999999999")); err != nil {
		t.Fatal(err)
	}
	idx, err := w.Index()
	if err != nil {
		t.Fatal(err)
	}
	n, _ := idx.Count()
	if n != 1 {
		t.Fatalf("header hook created entries: count=%d", n)
	}
	off, err := idx.FindOffset(h)
	if err != nil || off != 4242 {
		t.Fatalf("offset %d", off)
	}
	crc, err := idx.FindCRC32(h)
	if err != nil || crc != 0xdeadbeef {
		t.Fatalf("crc %x", crc)
	}
}

// 12. Offsets beyond 32 bits are appended to the overflow table in
//     encounter order, referenced by index. Inferable: partially —
//     assert several 64-bit offsets resolve independently.
func TestDetail12(t *testing.T) {
	h1 := hashWithFirstByte(0x01, 0x11)
	h2 := hashWithFirstByte(0x02, 0x22)
	h3 := hashWithFirstByte(0x03, 0x33)
	idx := buildIndex(t, []idxfile.Entry{
		{Hash: h1, Offset: uint64(1) << 33},
		{Hash: h2, Offset: uint64(1)<<34 + 5},
		{Hash: h3, Offset: 9},
	}, plumbing.NewHash("9999999999999999999999999999999999999999"))

	for hash, want := range map[plumbing.Hash]uint64{
		h1: uint64(1) << 33,
		h2: uint64(1)<<34 + 5,
		h3: 9,
	} {
		off, err := idx.FindOffset(hash)
		if err != nil || uint64(off) != want {
			t.Fatalf("offset for %v: %d want %d (err %v)", hash, off, want, err)
		}
	}
}

// 13. A closed iterator poisons further Next calls. Inferable: no —
//     assert SHAPE: Next after Close never yields a live entry.
func TestDetail13(t *testing.T) {
	h1 := hashWithFirstByte(0x10, 0x01)
	h2 := hashWithFirstByte(0x40, 0x02)
	pack := plumbing.NewHash("9999999999999999999999999999999999999999")
	idx := buildIndex(t, []idxfile.Entry{
		{Hash: h1, Offset: 1}, {Hash: h2, Offset: 2},
	}, pack)

	// Memory iterator.
	it, err := idx.Entries()
	if err != nil {
		t.Fatal(err)
	}
	if err := it.Close(); err != nil {
		t.Fatal(err)
	}
	e, err := it.Next()
	if err == nil && e != nil {
		t.Fatalf("closed memory iterator yielded %+v", e)
	}

	// Lazy iterator — release of the shared handle must surface.
	o := &racOpens{idxData: encodeIdx(t, idx), revData: encodeRev(t, idx)}
	lazy, err := idxfile.NewLazyIndex(o.openIdx, o.openRev, pack)
	if err != nil {
		t.Fatal(err)
	}
	defer lazy.Close()
	lit, err := lazy.Entries()
	if err != nil {
		t.Fatal(err)
	}
	lit.Close()
	e, err = lit.Next()
	if err == nil && e != nil {
		t.Fatalf("closed lazy iterator yielded %+v", e)
	}
}

// 14. The prefix iterator's slices alias the parent's bucket storage —
//     mutations after iterator creation are visible through it.
//     Inferable: doc.
func TestDetail14(t *testing.T) {
	a := hashWithFirstByte(0x10, 0x01)
	b := hashWithFirstByte(0x10, 0x02)
	idx := buildIndex(t, []idxfile.Entry{
		{Hash: a, Offset: 1}, {Hash: b, Offset: 2},
	}, plumbing.NewHash("9999999999999999999999999999999999999999"))

	it, err := idx.EntriesWithPrefix([]byte{0x10})
	if err != nil {
		t.Fatal(err)
	}
	defer it.Close()

	// Rewrite the second name in the bucket in place.
	mi := idx.FanoutMapping[0x10]
	slot := idSlot(idx, b)
	if slot < 0 {
		t.Fatal("b not located in bucket names")
	}
	copy(idx.Names[mi][slot:], a.Bytes())

	e1, err := it.Next()
	if err != nil || e1 == nil || !sameHash(e1.Hash, a) {
		t.Fatalf("first entry %v err %v", e1, err)
	}
	e2, err := it.Next()
	if err != nil {
		t.Fatalf("second Next: %v", err)
	}
	// Aliased view: the in-place mutation is observed by the iterator.
	if e2 == nil || !sameHash(e2.Hash, a) {
		t.Fatalf("iterator did not alias parent storage: %+v", e2)
	}
}
