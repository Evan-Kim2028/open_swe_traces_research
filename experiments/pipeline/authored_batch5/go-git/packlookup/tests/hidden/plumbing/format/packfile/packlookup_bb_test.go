package packfile

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	_ "crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	billy "github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/osfs"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/idxfile"
	"example.internal/gitkit/v6/storage/memory"
)

// --- fixtures -------------------------------------------------------

func lkBlob(t *testing.T, st *memory.Storage, content string) plumbing.Hash {
	t.Helper()
	obj := st.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	w, err := obj.Writer()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	h, err := st.SetEncodedObject(obj)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// lkEncodedPack encodes all of st's objects into one pack and returns
// the pack bytes plus a correct in-memory index over it.
func lkEncodedPack(t *testing.T, st *memory.Storage, hashes []plumbing.Hash, refDeltas bool) ([]byte, idxfile.Index, plumbing.Hash) {
	t.Helper()
	var buf bytes.Buffer
	ph, err := NewEncoder(&buf, st, refDeltas).Encode(hashes, 10)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	pack := buf.Bytes()

	iw := &idxfile.Writer{}
	p := NewParser(bytes.NewReader(pack), WithScannerObservers(iw))
	if _, err := p.Parse(); err != nil {
		t.Fatalf("fixture parse: %v", err)
	}
	idx, err := iw.Index()
	if err != nil {
		t.Fatalf("fixture index: %v", err)
	}
	return pack, idx, ph
}

// lkSpyFile wraps an os.File with open/close/read counters.
type lkSpyFile struct {
	*os.File
	closes *int32
	reads  *int32
}

func (f *lkSpyFile) Close() error {
	atomic.AddInt32(f.closes, 1)
	return f.File.Close()
}

func (f *lkSpyFile) ReadAt(p []byte, off int64) (int, error) {
	atomic.AddInt32(f.reads, 1)
	return f.File.ReadAt(p, off)
}

// lkCountingFS counts Open calls (for FSObject reopen-by-path checks).
type lkCountingFS struct {
	billy.Filesystem
	opens *int32
}

func (f *lkCountingFS) Open(path string) (billy.File, error) {
	atomic.AddInt32(f.opens, 1)
	return f.Filesystem.Open(path)
}

// lkHandle is a resolver-owned PackHandle over in-memory pack bytes.
type lkHandle struct {
	data          []byte
	hash          plumbing.Hash
	cursors       int32
	closedCursors int32
	closed        int32
}

type lkCursor struct {
	*bytes.Reader
	h *lkHandle
}

func (c *lkCursor) Close() error {
	atomic.AddInt32(&c.h.closedCursors, 1)
	return nil
}

func (h *lkHandle) OpenPackReader() (io.ReadSeekCloser, error) {
	atomic.AddInt32(&h.cursors, 1)
	return &lkCursor{Reader: bytes.NewReader(h.data), h: h}, nil
}

func (h *lkHandle) OpenRandomReader() (RandomReader, error) {
	atomic.AddInt32(&h.cursors, 1)
	return &lkCursor{Reader: bytes.NewReader(h.data), h: h}, nil
}

func (h *lkHandle) PackHash() (plumbing.Hash, error) { return h.hash, nil }

// --- hand-assembled pack helpers ------------------------------------

func lkZdeflate(b []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(b)
	w.Close()
	return buf.Bytes()
}

func lkEntryHead(typ plumbing.ObjectType, size int64) []byte {
	out := []byte{byte(typ<<4) | byte(size&0x0f)}
	size >>= 4
	for size > 0 {
		out[len(out)-1] |= 0x80
		out = append(out, byte(size&0x7f))
		size >>= 7
	}
	return out
}

func lkUvarint(v int64) []byte {
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

// lkRefDeltaEntry builds a REF_DELTA entry pointing at baseHash.
func lkRefDeltaEntry(baseHash plumbing.Hash, srcSize int64, dst []byte) []byte {
	body := lkUvarint(srcSize)
	body = append(body, lkUvarint(int64(len(dst)))...)
	body = append(body, byte(len(dst)))
	body = append(body, dst...)
	out := lkEntryHead(plumbing.REFDeltaObject, int64(len(body)))
	out = append(out, baseHash.Bytes()...)
	return append(out, lkZdeflate(body)...)
}

func lkAssemble(entries ...[]byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("PACK")
	binary.Write(&buf, binary.BigEndian, uint32(2))
	binary.Write(&buf, binary.BigEndian, uint32(len(entries)))
	for _, e := range entries {
		buf.Write(e)
	}
	sum := sha1.Sum(buf.Bytes())
	buf.Write(sum[:])
	return buf.Bytes()
}

// lkMemIndex builds an index over explicit (hash, offset) pairs.
func lkMemIndex(t *testing.T, packHash plumbing.Hash, entries map[plumbing.Hash]int64) idxfile.Index {
	t.Helper()
	w := &idxfile.Writer{}
	if err := w.OnHeader(uint32(len(entries))); err != nil {
		t.Fatal(err)
	}
	for h, off := range entries {
		if err := w.OnInflatedObjectContent(h, off, 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.OnFooter(packHash); err != nil {
		t.Fatal(err)
	}
	idx, err := w.Index()
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

// testPackObject / buildTestPack restore the harness shape kept
// scanner_test.go expects — compile-only, unused by the hidden tests.
type testPackObject struct {
	typ     plumbing.ObjectType
	content []byte
	base    plumbing.Hash
	offset  int64
}

func buildTestPack(t *testing.T, objs ...testPackObject) ([]byte, []int64) {
	t.Helper()
	var entries [][]byte
	var offsets []int64
	off := int64(12)
	for _, o := range objs {
		e := lkEntryHead(o.typ, int64(len(o.content)))
		e = append(e, lkZdeflate(o.content)...)
		offsets = append(offsets, off)
		off += int64(len(e))
		entries = append(entries, e)
	}
	return lkAssemble(entries...), offsets
}

// lkWritePack writes pack bytes into a fresh tempdir and returns
// (fs rooted at the dir, billy file, second raw os handle).
func lkWritePack(t *testing.T, pack []byte) (billy.Filesystem, billy.File, *os.File) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.pack"), pack, 0o644); err != nil {
		t.Fatal(err)
	}
	fsys := osfs.New(dir)
	f, err := fsys.Open("x.pack")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.Open(filepath.Join(dir, "x.pack"))
	if err != nil {
		t.Fatal(err)
	}
	return fsys, f, raw
}

// --- tests ----------------------------------------------------------

// 1. WithPackHandle makes the resolver own the descriptor — the
//    constructor closes the file argument and Close never touches the
//    resolver's handle. Inferable: doc.
func TestDetail01(t *testing.T) {
	st := memory.NewStorage()
	h := lkBlob(t, st, "resolver-owned\n")
	pack, idx, ph := lkEncodedPack(t, st, []plumbing.Hash{h}, false)

	_, _, raw := lkWritePack(t, pack)
	closes := int32(0)
	spy := &lkSpyFile{File: raw, closes: &closes}

	handle := &lkHandle{data: pack, hash: ph}
	resolverCalls := int32(0)
	p := NewPackfile(spy, WithIdx(idx), WithPackHandle(func() (PackHandle, error) {
		atomic.AddInt32(&resolverCalls, 1)
		return handle, nil
	}))

	obj, err := p.Get(h)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if obj.Type() != plumbing.BlobObject {
		t.Fatalf("type %v", obj.Type())
	}
	if atomic.LoadInt32(&closes) == 0 {
		t.Fatal("file argument was not closed by the constructor")
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&handle.closed) != 0 {
		t.Fatal("resolver-owned handle was closed by Packfile.Close")
	}
}

// 2. init runs once: resolves the handle, scans the pack signature,
//    reads the trailing checksum as the pack ID (via the handle's
//    PackHash when a resolver is in play, else a seek-to-end read),
//    and only then serves reads. Inferable: partially.
func TestDetail02(t *testing.T) {
	st := memory.NewStorage()
	h := lkBlob(t, st, "init once\n")
	pack, idx, ph := lkEncodedPack(t, st, []plumbing.Hash{h}, false)

	handle := &lkHandle{data: pack, hash: ph}
	resolverCalls := int32(0)
	_, dummy, _ := lkWritePack(t, pack)
	p := NewPackfile(dummy, WithIdx(idx), WithPackHandle(func() (PackHandle, error) {
		atomic.AddInt32(&resolverCalls, 1)
		return handle, nil
	}))
	for i := 0; i < 3; i++ {
		if _, err := p.Get(h); err != nil {
			t.Fatalf("Get %d: %v", i, err)
		}
	}
	if n := atomic.LoadInt32(&resolverCalls); n != 1 {
		t.Fatalf("resolver called %d times, want 1", n)
	}
	id, err := p.ID()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(id.Bytes(), ph.Bytes()) {
		t.Fatalf("ID %v != pack hash %v", id, ph)
	}
	p.Close()

	// Legacy mode: the ID is the trailer read off the file itself.
	_, bf, _ := lkWritePack(t, pack)
	p2 := NewPackfile(bf, WithIdx(idx))
	defer p2.Close()
	id2, err := p2.ID()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(id2.Bytes(), ph.Bytes()) {
		t.Fatalf("legacy ID %v != pack hash %v", id2, ph)
	}

	// A bad signature surfaces as an init error on first use.
	_, bf3, _ := lkWritePack(t, []byte("definitely not a pack"))
	p3 := NewPackfile(bf3, WithIdx(idx))
	defer p3.Close()
	if _, err := p3.Get(h); err == nil {
		t.Fatal("bad pack signature produced no init error")
	}
}

// 3. Every read path re-checks closed AFTER taking the mutex — a
//    concurrent Close returns fs.ErrClosed rather than racing a
//    torn-down scanner. Inferable: partially.
func TestDetail03(t *testing.T) {
	st := memory.NewStorage()
	h := lkBlob(t, st, "race\n")
	pack, idx, _ := lkEncodedPack(t, st, []plumbing.Hash{h}, false)
	_, bf, _ := lkWritePack(t, pack)
	p := NewPackfile(bf, WithIdx(idx))

	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := p.Get(h)
			errs <- err
		}()
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil && !errors.Is(err, fs.ErrClosed) {
			t.Fatalf("concurrent Get saw %v, want nil or ErrClosed", err)
		}
	}
	// After Close the failure is stable.
	if _, err := p.Get(h); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("post-close Get: %v", err)
	}
}

// 4. Get resolves hash→offset through the index, seeks the scanner to
//    the entry, and reads the header; a scan that ends cleanly without
//    an object header is ErrObjectNotFound, not the scanner's error.
//    Inferable: partially.
func TestDetail04(t *testing.T) {
	st := memory.NewStorage()
	h := lkBlob(t, st, "lookup\n")
	pack, idx, _ := lkEncodedPack(t, st, []plumbing.Hash{h}, false)
	_, bf, _ := lkWritePack(t, pack)
	p := NewPackfile(bf, WithIdx(idx))
	defer p.Close()

	if _, err := p.Get(h); err != nil {
		t.Fatalf("Get existing: %v", err)
	}
	missing := plumbing.NewHash("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	if _, err := p.Get(missing); !errors.Is(err, plumbing.ErrObjectNotFound) {
		t.Fatalf("Get missing: %v, want ErrObjectNotFound", err)
	}
}

// 5. Non-delta objects on a filesystem-backed pack are returned as
//    lazy FSObjects — the bytes are not inflated until Reader is
//    called. Deltas always inflate through getMemoryObject.
//    Inferable: doc.
func TestDetail05(t *testing.T) {
	st := memory.NewStorage()
	h := lkBlob(t, st, "lazy blob content\n")
	pack, idx, _ := lkEncodedPack(t, st, []plumbing.Hash{h}, false)

	fsys, bf, _ := lkWritePack(t, pack)
	p := NewPackfile(bf, WithIdx(idx), WithFs(fsys))
	defer p.Close()

	obj, err := p.Get(h)
	if err != nil {
		t.Fatal(err)
	}
	fso, ok := obj.(*FSObject)
	if !ok {
		t.Fatalf("filesystem pack returned %T, want *FSObject", obj)
	}
	r, err := fso.Reader()
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if string(got) != "lazy blob content\n" {
		t.Fatalf("content %q", got)
	}

	// Without WithFs the same Get yields a memory object.
	_, bf2, _ := lkWritePack(t, pack)
	p2 := NewPackfile(bf2, WithIdx(idx))
	defer p2.Close()
	obj2, err := p2.Get(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := obj2.(*FSObject); ok {
		t.Fatal("in-memory packfile returned an FSObject")
	}
}

// 6. A delta object's stored type is replaced by its BASE's type — the
//    returned object reports the resolved type, not OFSDelta/REFDelta.
//    Inferable: partially.
func TestDetail06(t *testing.T) {
	st := memory.NewStorage()
	base := make([]byte, 512)
	for i := range base {
		base[i] = 'a' + byte(i%20)
	}
	derived := append([]byte(nil), base...)
	derived[100] = 'Z'
	hBase := lkBlob(t, st, string(base))
	hDelta := lkBlob(t, st, string(derived))

	pack, idx, _ := lkEncodedPack(t, st, []plumbing.Hash{hBase, hDelta}, false)
	_, bf, _ := lkWritePack(t, pack)
	p := NewPackfile(bf, WithIdx(idx))
	defer p.Close()

	obj, err := p.Get(hDelta)
	if err != nil {
		t.Fatalf("Get delta: %v", err)
	}
	if obj.Type() == plumbing.OFSDeltaObject || obj.Type() == plumbing.REFDeltaObject {
		t.Fatalf("delta type leaked: %v", obj.Type())
	}
	if obj.Type() != plumbing.BlobObject {
		t.Fatalf("resolved type %v, want BlobObject", obj.Type())
	}
	r, _ := obj.Reader()
	got, _ := io.ReadAll(r)
	r.Close()
	if !bytes.Equal(got, derived) {
		t.Fatal("delta content mismatch")
	}
}

// 7. REFDelta bases resolve through the hash index first (cache, then
//    Get); OFSDelta bases resolve by offset — the two delta kinds use
//    different base lookups. Inferable: partially.
func TestDetail07(t *testing.T) {
	for _, refDeltas := range []bool{false, true} {
		st := memory.NewStorage()
		base := bytes.Repeat([]byte("0123456789abcdef"), 40)
		derived := append([]byte(nil), base...)
		copy(derived[200:], "CHANGED!")
		hBase := lkBlob(t, st, string(base))
		hDelta := lkBlob(t, st, string(derived))

		pack, idx, _ := lkEncodedPack(t, st, []plumbing.Hash{hBase, hDelta}, refDeltas)
		_, bf, _ := lkWritePack(t, pack)
		p := NewPackfile(bf, WithIdx(idx))

		obj, err := p.Get(hDelta)
		if err != nil {
			t.Fatalf("refDeltas=%v Get: %v", refDeltas, err)
		}
		if obj.Type() != plumbing.BlobObject {
			t.Fatalf("refDeltas=%v type %v", refDeltas, obj.Type())
		}
		r, _ := obj.Reader()
		got, _ := io.ReadAll(r)
		r.Close()
		if !bytes.Equal(got, derived) {
			t.Fatalf("refDeltas=%v content mismatch", refDeltas)
		}
		p.Close()
	}
}

// 8. A delta whose base can't be found errors — partial packs with
//    missing bases are not silently inflated. Inferable: partially.
func TestDetail08(t *testing.T) {
	absent := plumbing.NewHash("dddddddddddddddddddddddddddddddddddddddd")
	deltaEntry := lkRefDeltaEntry(absent, 10, []byte("x"))
	pack := lkAssemble(deltaEntry)
	packHash, _ := plumbing.FromBytes(pack[len(pack)-20:])
	// The delta's own idx key is any hash → offset of the entry (12).
	key := plumbing.NewHash("abababababababababababababababababababab")
	idx := lkMemIndex(t, packHash, map[plumbing.Hash]int64{key: 12})

	_, bf, _ := lkWritePack(t, pack)
	p := NewPackfile(bf, WithIdx(idx))
	defer p.Close()
	if _, err := p.Get(key); err == nil {
		t.Fatal("delta with missing base inflated without error")
	}
}

// 9. FSObject.Reader probes the descriptor with a 1-byte ReadAt and
//    reopens the pack by path when the FD was closed underneath it —
//    never on transient errors. Inferable: no — assert observable
//    probe/reopen behaviour.
func TestDetail09(t *testing.T) {
	// probePack shape: live descriptor → nil; closed → a closed-FD
	// error; past-end → an error is propagated, not swallowed.
	live := &lkCursor{Reader: bytes.NewReader([]byte("data")), h: &lkHandle{}}
	if err := probePack(live, 0); err != nil {
		t.Fatalf("probe on live descriptor: %v", err)
	}
	closed, _ := os.Open(os.DevNull)
	closed.Close()
	if err := probePack(closed, 0); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("probe on closed descriptor: %v", err)
	}
	if err := probePack(live, 100); err == nil {
		t.Fatal("probe past EOF returned nil")
	}

	// End-to-end reopen: an FSObject whose underlying FD is closed
	// behind its back reopens the pack by path and still serves bytes.
	st := memory.NewStorage()
	h := lkBlob(t, st, "reopen me\n")
	pack, idx, _ := lkEncodedPack(t, st, []plumbing.Hash{h}, false)

	var opens int32
	fsys, bf, _ := lkWritePack(t, pack)
	cfs := &lkCountingFS{Filesystem: fsys, opens: &opens}
	p := NewPackfile(bf, WithIdx(idx), WithFs(cfs))
	defer p.Close()

	obj, err := p.Get(h)
	if err != nil {
		t.Fatal(err)
	}
	fso := obj.(*FSObject)

	// Kill the descriptor underneath the FSObject: the packfile's own
	// file handle is the FD the object reads through.
	if err := bf.Close(); err != nil {
		t.Logf("billy close: %v", err)
	}

	before := atomic.LoadInt32(&opens)
	r, err := fso.Reader()
	if err != nil {
		t.Fatalf("Reader after FD kill: %v", err)
	}
	got, _ := io.ReadAll(r)
	r.Close()
	if string(got) != "reopen me\n" {
		t.Fatalf("content %q after reopen", got)
	}
	if atomic.LoadInt32(&opens) == before {
		t.Fatal("pack was not reopened by path after FD close")
	}
}

// 10. FSObject readers are concurrency-safe because they read through
//     SectionReader/ReadAt — never Seek on the shared handle.
//     Inferable: doc.
func TestDetail10(t *testing.T) {
	st := memory.NewStorage()
	var contents []string
	var hashes []plumbing.Hash
	for i := 0; i < 8; i++ {
		c := strings.Repeat(string(rune('a'+i)), 200)
		contents = append(contents, c)
		hashes = append(hashes, lkBlob(t, st, c))
	}
	pack, idx, _ := lkEncodedPack(t, st, hashes, false)
	fsys, bf, _ := lkWritePack(t, pack)
	p := NewPackfile(bf, WithIdx(idx), WithFs(fsys))
	defer p.Close()

	var wg sync.WaitGroup
	fail := make(chan string, 64)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for k := 0; k < 5; k++ {
				obj, err := p.Get(hashes[i])
				if err != nil {
					fail <- err.Error()
					return
				}
				r, err := obj.Reader()
				if err != nil {
					fail <- err.Error()
					return
				}
				got, _ := io.ReadAll(r)
				r.Close()
				if string(got) != contents[i] {
					fail <- "content mismatch"
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(fail)
	for m := range fail {
		t.Fatal(m)
	}
}

// 11. The iterator resolves delta headers to their base type when
//     filtering by type — a delta entry counts as its resolved type,
//     and non-matching entries are skipped, not errored. Inferable:
//     partially.
func TestDetail11(t *testing.T) {
	st := memory.NewStorage()
	base := bytes.Repeat([]byte("0123456789abcdef"), 40)
	derived := append([]byte(nil), base...)
	copy(derived[200:], "DIFF!")
	hBase := lkBlob(t, st, string(base))
	hDelta := lkBlob(t, st, string(derived))

	pack, idx, _ := lkEncodedPack(t, st, []plumbing.Hash{hBase, hDelta}, false)
	_, bf, _ := lkWritePack(t, pack)
	p := NewPackfile(bf, WithIdx(idx))
	defer p.Close()

	// Both entries resolve to BlobObject.
	it, err := p.GetByType(plumbing.BlobObject)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	err = it.ForEach(func(o plumbing.EncodedObject) error {
		if o.Type() != plumbing.BlobObject {
			t.Fatalf("iterator yielded %v", o.Type())
		}
		n++
		return nil
	})
	it.Close()
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("GetByType(Blob) yielded %d, want 2 (delta resolves to base type)", n)
	}

	// Non-matching type: skipped without error.
	it2, err := p.GetByType(plumbing.CommitObject)
	if err != nil {
		t.Fatal(err)
	}
	n = 0
	err = it2.ForEach(func(o plumbing.EncodedObject) error {
		n++
		return nil
	})
	it2.Close()
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("GetByType(Commit) yielded %d", n)
	}
}

// 12. Close is idempotent and only closes what the packfile owns — the
//     scanner cursor under a resolver, or the file itself otherwise.
//     Inferable: doc.
func TestDetail12(t *testing.T) {
	st := memory.NewStorage()
	h := lkBlob(t, st, "close\n")
	pack, idx, ph := lkEncodedPack(t, st, []plumbing.Hash{h}, false)

	// Legacy: the file argument is closed exactly once.
	var closes int32
	_, _, raw := lkWritePack(t, pack)
	spy := &lkSpyFile{File: raw, closes: &closes}
	p := NewPackfile(spy, WithIdx(idx))
	if _, err := p.Get(h); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if n := atomic.LoadInt32(&closes); n != 1 {
		t.Fatalf("file closed %d times, want 1", n)
	}

	// Resolver mode: the scanner cursor is released, the handle is not.
	handle := &lkHandle{data: pack, hash: ph}
	_, _, raw2 := lkWritePack(t, pack)
	spy2 := &lkSpyFile{File: raw2, closes: &closes}
	p2 := NewPackfile(spy2, WithIdx(idx), WithPackHandle(func() (PackHandle, error) {
		return handle, nil
	}))
	if _, err := p2.Get(h); err != nil {
		t.Fatal(err)
	}
	if err := p2.Close(); err != nil {
		t.Fatal(err)
	}
	if err := p2.Close(); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&handle.closed) != 0 {
		t.Fatal("resolver-owned handle closed")
	}
	if atomic.LoadInt32(&handle.closedCursors) == 0 {
		t.Fatal("scanner cursor was not released on Close")
	}
}
