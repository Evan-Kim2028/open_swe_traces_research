package packhandle_test

import (
	"bytes"
	"crypto/sha1"
	_ "crypto/sha256"
	"errors"
	"io"
	"io/fs"
	"sync/atomic"
	"testing"
	"time"

	"example.internal/gitkit/v6/internal/packhandle"
	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/idxfile"
	"example.internal/gitkit/v6/plumbing/format/packfile"
	"example.internal/gitkit/v6/plumbing/format/revfile"
	"example.internal/gitkit/v6/storage/memory"
	"example.internal/gitkit/v6/x/fdpool"
)

// spyRAC is a ReadAtCloser over bytes that reports opens/closes/reads.
type spyRAC struct {
	*bytes.Reader
	fx *fixture
}

func (s *spyRAC) ReadAt(p []byte, off int64) (int, error) {
	atomic.AddInt32(&s.fx.readAts, 1)
	return s.Reader.ReadAt(p, off)
}

func (s *spyRAC) Close() error {
	atomic.AddInt32(&s.fx.closes, 1)
	return s.fx.closeErr
}

type fixture struct {
	data      []byte
	opens     int32
	closes    int32
	readAts   int32
	sizeCalls int32
	openErr   error
	sizeErr   error
	closeErr  error
}

func (f *fixture) source() packhandle.Source {
	return packhandle.Source{
		Open: func() (packhandle.ReadAtCloser, error) {
			if f.openErr != nil {
				return nil, f.openErr
			}
			atomic.AddInt32(&f.opens, 1)
			return &spyRAC{Reader: bytes.NewReader(f.data), fx: f}, nil
		},
		Size: func() (int64, error) {
			atomic.AddInt32(&f.sizeCalls, 1)
			if f.sizeErr != nil {
				return 0, f.sizeErr
			}
			return int64(len(f.data)), nil
		},
	}
}

// buildPack encodes a one-blob pack and returns the bytes plus the
// footer hash the encoder computed.
func buildPack(t *testing.T) ([]byte, plumbing.Hash) {
	t.Helper()
	st := memory.NewStorage()
	obj := st.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	w, err := obj.Writer()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("packhandle fixture blob\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	h, err := st.SetEncodedObject(obj)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	ph, err := packfile.NewEncoder(&buf, st, false).Encode([]plumbing.Hash{h}, 0)
	if err != nil {
		t.Fatalf("encode pack: %v", err)
	}
	return buf.Bytes(), ph
}

// buildIdxRev returns encoded idx+rev bytes for one entry — the lazy
// index only needs well-formed sources, not entries matching the pack.
func buildIdxRev(t *testing.T, packHash plumbing.Hash) (idxBytes, revBytes []byte) {
	t.Helper()
	w := &idxfile.Writer{}
	if err := w.OnHeader(1); err != nil {
		t.Fatal(err)
	}
	if err := w.OnInflatedObjectContent(
		plumbing.NewHash("1111111111111111111111111111111111111111"), 12, 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.OnFooter(packHash); err != nil {
		t.Fatal(err)
	}
	idx, err := w.Index()
	if err != nil {
		t.Fatal(err)
	}
	var ib, rb bytes.Buffer
	if err := idxfile.Encode(&ib, sha1.New(), idx); err != nil {
		t.Fatal(err)
	}
	if err := revfile.Encode(&rb, sha1.New(), idx); err != nil {
		t.Fatal(err)
	}
	return ib.Bytes(), rb.Bytes()
}

func newHandle(t *testing.T, fx *fixture, h plumbing.Hash) *packhandle.PackHandle {
	t.Helper()
	ph, err := packhandle.New(packhandle.Sources{Pack: fx.source()}, h)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { ph.Close() })
	return ph
}

// 1. Construction refuses a missing Pack.Open or Pack.Size, and a zero
//    pack hash — each with its own error var. Inferable: doc.
func TestDetail01(t *testing.T) {
	pack, ph := buildPack(t)
	good := (&fixture{data: pack}).source()

	if _, err := packhandle.New(packhandle.Sources{}, ph); !errors.Is(err, packhandle.ErrPackSourceRequired) {
		t.Fatalf("empty sources: %v", err)
	}
	noOpen := packhandle.Sources{Pack: packhandle.Source{Size: good.Size}}
	if _, err := packhandle.New(noOpen, ph); !errors.Is(err, packhandle.ErrPackSourceRequired) {
		t.Fatalf("missing Open: %v", err)
	}
	noSize := packhandle.Sources{Pack: packhandle.Source{Open: good.Open}}
	if _, err := packhandle.New(noSize, ph); !errors.Is(err, packhandle.ErrPackSourceRequired) {
		t.Fatalf("missing Size: %v", err)
	}
	full := packhandle.Sources{Pack: good}
	if _, err := packhandle.New(full, plumbing.ZeroHash); !errors.Is(err, packhandle.ErrInvalidPackHash) {
		t.Fatalf("zero hash: %v", err)
	}
}

// 2. The pack size is fetched once and cached in an atomic — failures
//    are NOT cached, so a transient Size error retries next call.
//    Inferable: doc.
func TestDetail02(t *testing.T) {
	pack, ph := buildPack(t)
	fx := &fixture{data: pack}
	h := newHandle(t, fx, ph)

	c1, err := h.OpenPackReader()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	c2, err := h.OpenPackReader()
	if err != nil {
		t.Fatalf("open2: %v", err)
	}
	if n := atomic.LoadInt32(&fx.sizeCalls); n != 1 {
		t.Fatalf("Size called %d times for two cursors, want 1", n)
	}
	c1.Close()
	c2.Close()

	// Transient failure: not cached — the next call retries.
	fx2 := &fixture{data: pack, sizeErr: errors.New("flaky stat")}
	h2 := newHandle(t, fx2, ph)
	if _, err := h2.OpenPackReader(); err == nil {
		t.Fatal("first cursor should fail on Size error")
	}
	fx2.sizeErr = nil
	if _, err := h2.OpenPackReader(); err != nil {
		t.Fatalf("retry after Size error: %v", err)
	}
	if n := atomic.LoadInt32(&fx2.sizeCalls); n != 2 {
		t.Fatalf("Size called %d times, want 2 (failure not cached)", n)
	}
}

// 3. Every cursor holds one reference on the shared pack file — the FD
//    outlives its cursor until Close releases it, and the grace timer
//    can only fire with zero live cursors. Inferable: doc.
func TestDetail03(t *testing.T) {
	pack, ph := buildPack(t)
	fx := &fixture{data: pack}
	h := newHandle(t, fx, ph)

	c, err := h.OpenPackReader()
	if err != nil {
		t.Fatal(err)
	}
	// Past the documented ~1s grace window, the FD is still open
	// because the cursor pins it.
	time.Sleep(1500 * time.Millisecond)
	if n := atomic.LoadInt32(&fx.closes); n != 0 {
		t.Fatalf("pack FD closed with a live cursor (closes=%d)", n)
	}
	c.Close()
	deadline := time.Now().Add(4 * time.Second)
	for atomic.LoadInt32(&fx.closes) == 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if atomic.LoadInt32(&fx.closes) == 0 {
		t.Fatal("grace timer never fired after last cursor release")
	}
}

// 4. A closed handle answers fs.ErrClosed from open paths, Meta and
//    Index — the closed flag is checked before any FD work. Inferable: doc.
func TestDetail04(t *testing.T) {
	pack, ph := buildPack(t)
	fx := &fixture{data: pack}
	h := newHandle(t, fx, ph)

	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := h.OpenPackReader(); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("OpenPackReader on closed: %v", err)
	}
	if _, err := h.OpenRandomReader(); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("OpenRandomReader on closed: %v", err)
	}
	if _, err := h.Meta(); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("Meta on closed: %v", err)
	}
	if _, err := h.Index(); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("Index on closed: %v", err)
	}
}

// 5. Meta parses and VALIDATES: magic PACK, version 2 or 3, and the
//    footer must equal the pinned pack hash — a well-formed pack with a
//    wrong footer still errors. Inferable: partially.
func TestDetail05(t *testing.T) {
	pack, ph := buildPack(t)

	m, err := newHandle(t, &fixture{data: pack}, ph).Meta()
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	if m.Version != 2 && m.Version != 3 {
		t.Fatalf("version %d", m.Version)
	}
	if m.Count != 1 {
		t.Fatalf("count %d", m.Count)
	}
	if !bytes.Equal(m.ID.Bytes(), ph.Bytes()) {
		t.Fatalf("ID %v != pack hash %v", m.ID, ph)
	}

	// Bad magic.
	bad := append([]byte(nil), pack...)
	copy(bad[:4], "XXXX")
	if _, err := newHandle(t, &fixture{data: bad}, ph).Meta(); err == nil {
		t.Fatal("Meta accepted bad magic")
	}
	// Unsupported version.
	badV := append([]byte(nil), pack...)
	badV[7] = 9
	if _, err := newHandle(t, &fixture{data: badV}, ph).Meta(); err == nil {
		t.Fatal("Meta accepted version 9")
	}
	// Well-formed pack pinned against the wrong hash.
	other := plumbing.NewHash("9999999999999999999999999999999999999999")
	if _, err := newHandle(t, &fixture{data: pack}, other).Meta(); err == nil {
		t.Fatal("Meta accepted a footer != pinned hash")
	}
}

// 6. Meta caches its first success; parse failures retry. Inferable: doc.
func TestDetail06(t *testing.T) {
	pack, ph := buildPack(t)
	fx := &fixture{data: pack}
	h := newHandle(t, fx, ph)

	m1, err := h.Meta()
	if err != nil {
		t.Fatal(err)
	}
	before := atomic.LoadInt32(&fx.readAts)
	m2, err := h.Meta()
	if err != nil {
		t.Fatal(err)
	}
	if m1 != m2 {
		t.Fatalf("Meta changed between calls: %+v vs %+v", m1, m2)
	}
	if n := atomic.LoadInt32(&fx.readAts) - before; n != 0 {
		t.Fatalf("second Meta re-read the pack (%d new reads) — success not cached", n)
	}

	// A failed parse is not cached — the next call reads again.
	bad := append([]byte(nil), pack...)
	copy(bad[:4], "XXXX")
	fx2 := &fixture{data: bad}
	h2 := newHandle(t, fx2, ph)
	if _, err := h2.Meta(); err == nil {
		t.Fatal("expected parse failure")
	}
	before = atomic.LoadInt32(&fx2.readAts)
	if _, err := h2.Meta(); err == nil {
		t.Fatal("expected parse failure on retry")
	}
	if atomic.LoadInt32(&fx2.readAts) == before {
		t.Fatal("parse failure was cached — retry never re-read")
	}
}

// 7. Close is idempotent via a once-wrapper, sets closed BEFORE
//    releasing FDs, and joins the pack and index errors. Inferable:
//    partially.
func TestDetail07(t *testing.T) {
	pack, ph := buildPack(t)
	fx := &fixture{data: pack, closeErr: errors.New("pack close boom")}
	h := newHandle(t, fx, ph)

	// Force the pack FD open (Meta reads through it).
	if _, err := h.Meta(); err != nil {
		t.Fatal(err)
	}
	err := h.Close()
	if err == nil {
		t.Fatal("Close swallowed the pack close error")
	}
	// closed flag was set before/without depending on close success.
	if _, err := h.Meta(); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("handle not closed after errored Close: %v", err)
	}
	// The once-wrapper runs the close body once: the second call
	// re-executes nothing on the underlying file.
	if err := h.Close(); err == nil {
		t.Log("second Close returned nil")
	}
	if n := atomic.LoadInt32(&fx.closes); n != 1 {
		t.Fatalf("underlying pack closed %d times, want 1", n)
	}
}

// 8. CloseIdleDescriptors releases FDs without closing the handle and
//    keeps the meta/index caches — active readers finish normally.
//    Inferable: doc.
func TestDetail08(t *testing.T) {
	pack, ph := buildPack(t)
	idxB, revB := buildIdxRev(t, ph)
	fx := &fixture{data: pack}
	idxFx := &fixture{data: idxB}
	revFx := &fixture{data: revB}
	h, err := packhandle.New(packhandle.Sources{
		Pack: fx.source(), Idx: idxFx.source(), Rev: revFx.source(),
	}, ph)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.Close() })

	m1, err := h.Meta()
	if err != nil {
		t.Fatal(err)
	}
	i1, err := h.Index()
	if err != nil {
		t.Fatal(err)
	}
	readsBefore := atomic.LoadInt32(&fx.readAts)

	if err := h.CloseIdleDescriptors(); err != nil {
		t.Fatalf("CloseIdleDescriptors: %v", err)
	}
	if atomic.LoadInt32(&fx.closes) == 0 {
		t.Fatal("CloseIdleDescriptors did not release the pack FD")
	}

	// Handle still open: new cursors reopen the FD.
	c, err := h.OpenPackReader()
	if err != nil {
		t.Fatalf("cursor after idle-close: %v", err)
	}
	c.Close()
	if atomic.LoadInt32(&fx.opens) < 2 {
		t.Fatal("pack FD was not reopened on demand")
	}

	// Meta and index caches survive: same values, no re-parse.
	m2, err := h.Meta()
	if err != nil {
		t.Fatal(err)
	}
	if m1 != m2 || atomic.LoadInt32(&fx.readAts) != readsBefore {
		t.Fatal("meta cache was dropped by CloseIdleDescriptors")
	}
	i2, err := h.Index()
	if err != nil {
		t.Fatal(err)
	}
	if i1 != i2 {
		t.Fatal("index cache was dropped by CloseIdleDescriptors")
	}
}

// 9. Cursor Read converts a short trailing ReadAt EOF into a clean read
//    — data returned with err=nil, EOF only when offset reaches size.
//    Inferable: no — assert the observable EOF-normalisation.
func TestDetail09(t *testing.T) {
	pack, ph := buildPack(t)
	fx := &fixture{data: pack}
	h := newHandle(t, fx, ph)

	c, err := h.OpenPackReader()
	if err != nil {
		t.Fatal(err)
	}
	n := int64(len(pack))
	if _, err := c.Seek(n-3, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 8)
	got, err := c.Read(buf)
	if err != nil || got != 3 {
		t.Fatalf("trailing read: n=%d err=%v", got, err)
	}
	if !bytes.Equal(buf[:3], pack[n-3:]) {
		t.Fatal("trailing read returned wrong bytes")
	}
	got, err = c.Read(buf)
	if got != 0 || err != io.EOF {
		t.Fatalf("at size: n=%d err=%v, want EOF", got, err)
	}
	c.Close()
}

// 10. Seek rejects unknown whence and negative absolute positions with
//     distinct error vars; SeekEnd resolves against the cached size.
//     Inferable: partially.
func TestDetail10(t *testing.T) {
	pack, ph := buildPack(t)
	fx := &fixture{data: pack}
	h := newHandle(t, fx, ph)

	c, err := h.OpenPackReader()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if _, err := c.Seek(0, 99); !errors.Is(err, packhandle.ErrInvalidSeekWhence) {
		t.Fatalf("bad whence: %v", err)
	}
	if _, err := c.Seek(-1, io.SeekStart); !errors.Is(err, packhandle.ErrNegativeSeekPosition) {
		t.Fatalf("negative absolute: %v", err)
	}
	if _, err := c.Seek(-int64(len(pack))-1, io.SeekEnd); !errors.Is(err, packhandle.ErrNegativeSeekPosition) {
		t.Fatalf("negative via SeekEnd: %v", err)
	}
	pos, err := c.Seek(-4, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if pos != int64(len(pack))-4 {
		t.Fatalf("SeekEnd(-4) = %d, want %d", pos, len(pack)-4)
	}
}

// 11. Cursor Close releases its reference exactly once — a second Close
//     is a no-op. Inferable: partially.
func TestDetail11(t *testing.T) {
	pack, ph := buildPack(t)
	fx := &fixture{data: pack}
	h := newHandle(t, fx, ph)

	c, err := h.OpenPackReader()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("second cursor Close: %v", err)
	}
	// After release the cursor refuses work.
	if _, err := c.Read(make([]byte, 4)); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("read on closed cursor: %v", err)
	}
}

// 12. Index builds a LazyIndex over the idx/rev sources and refuses
//     when either source is absent — with ErrSourceUnconfigured, not a
//     nil pointer crash. Inferable: doc.
func TestDetail12(t *testing.T) {
	pack, ph := buildPack(t)
	idxB, revB := buildIdxRev(t, ph)
	packFx := &fixture{data: pack}

	// Missing idx.
	h := newHandle(t, packFx, ph)
	if _, err := h.Index(); !errors.Is(err, packhandle.ErrSourceUnconfigured) {
		t.Fatalf("Index without idx source: %v", err)
	}
	h.Close()

	// Missing rev.
	h = newHandle(t, packFx, ph)
	h2src := packhandle.Sources{Pack: packFx.source(), Idx: (&fixture{data: idxB}).source()}
	h2, err := packhandle.New(h2src, ph)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h2.Index(); !errors.Is(err, packhandle.ErrSourceUnconfigured) {
		t.Fatalf("Index without rev source: %v", err)
	}
	h2.Close()

	// Both present: a working index comes back.
	full := packhandle.Sources{
		Pack: packFx.source(),
		Idx:  (&fixture{data: idxB}).source(),
		Rev:  (&fixture{data: revB}).source(),
	}
	h3, err := packhandle.New(full, ph)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := h3.Index()
	if err != nil {
		t.Fatalf("Index with both sources: %v", err)
	}
	if idx == nil {
		t.Fatal("nil index")
	}
	h3.Close()
}

// 13. With a pool, the grace timer is inert — the pool's LRU owns FD
//     lifetime; without one, the 1s grace governs. Inferable: doc.
func TestDetail13(t *testing.T) {
	pack, ph := buildPack(t)
	fx := &fixture{data: pack}

	pool := fdpool.New(4)
	h, err := packhandle.NewWithPool(packhandle.Sources{Pack: fx.source()}, ph, pool)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.Close() })

	c, err := h.OpenPackReader()
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
	// Well past the 1s grace window: the pool still holds the FD.
	time.Sleep(1600 * time.Millisecond)
	if n := atomic.LoadInt32(&fx.closes); n != 0 {
		t.Fatalf("pooled pack FD closed by the grace timer (closes=%d)", n)
	}
	// The handle itself is still fully functional.
	if _, err := h.Meta(); err != nil {
		t.Fatalf("Meta on pooled handle: %v", err)
	}
}
