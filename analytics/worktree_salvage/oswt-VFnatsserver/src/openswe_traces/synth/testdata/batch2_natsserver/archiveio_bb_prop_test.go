// Hidden black-box property suite for the archiveio unit.
// Drives only the exported API in server/archive (api.md): MagicBytes,
// Header, NewWriter/WriteHeader/Write/Close/Flush, NewReader/Next/Read,
// and the documented error values.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package archive_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"math/rand"
	"os"
	"strconv"
	"testing"

	archive "example.internal/msgkit/v2/server/archive"
)

const bbArcHiddenSeed = 20260919

func bbArcSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbArcHiddenSeed
}

func bbArcRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbArcSeed()))
}

func bbArcRandBytes(rng *rand.Rand, n int) []byte {
	b := make([]byte, n)
	rng.Read(b)
	return b
}

// bbArcEncodeEntry encodes one entry in the documented wire order:
// nameLen uvarint, timestamp varint, sequence uvarint, headerSize uvarint,
// payloadSize uvarint, raw name.
func bbArcEncodeEntry(name string, ts int64, seq uint64, hsz, psz int64, payload []byte) []byte {
	var buf bytes.Buffer
	var tmp [binary.MaxVarintLen64]byte
	putU := func(v uint64) { n := binary.PutUvarint(tmp[:], v); buf.Write(tmp[:n]) }
	putV := func(v int64) { n := binary.PutVarint(tmp[:], v); buf.Write(tmp[:n]) }
	putU(uint64(len(name)))
	putV(ts)
	putU(seq)
	putU(uint64(hsz))
	putU(uint64(psz))
	buf.WriteString(name)
	buf.Write(payload)
	return buf.Bytes()
}

type bbArcFlusher struct{ bytes.Buffer }

func (f *bbArcFlusher) Flush() error { return nil }

type bbArcFlushNoErr struct{ bytes.Buffer }

func (f *bbArcFlushNoErr) Flush() {}

// Detail 1: magic is written once, lazily, before the first entry — not at
// construction.
func TestDetail01_LazyMagicEmission(t *testing.T) {
	var buf bytes.Buffer
	w := archive.NewWriter(&buf)
	if buf.Len() != 0 {
		t.Fatalf("NewWriter wrote %d bytes before first entry", buf.Len())
	}
	rng := bbArcRng(t)
	name := "e" + strconv.Itoa(rng.Intn(1000))
	// HeaderSize+PayloadSize of 0 leaves no pending bytes, so the next
	// WriteHeader is legal.
	if err := w.WriteHeader(&archive.Header{Name: name, HeaderSize: 0, PayloadSize: 0}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if buf.Len() <= len(archive.MagicBytes) {
		t.Fatalf("after first WriteHeader buf=%d bytes, want magic+entry", buf.Len())
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte(archive.MagicBytes)) {
		t.Fatalf("stream does not start with magic %q: %q", archive.MagicBytes, buf.Bytes()[:8])
	}
	// Second entry does not repeat the magic.
	before := buf.Len()
	if err := w.WriteHeader(&archive.Header{Name: "f", HeaderSize: 0, PayloadSize: 0}); err != nil {
		t.Fatalf("WriteHeader 2: %v", err)
	}
	if bytes.Count(buf.Bytes(), []byte(archive.MagicBytes)) != 1 {
		t.Fatalf("magic emitted %d times, want 1", bytes.Count(buf.Bytes(), []byte(archive.MagicBytes)))
	}
	_ = before
	// An empty archive is 0 bytes end to end.
	var ebuf bytes.Buffer
	ew := archive.NewWriter(&ebuf)
	if err := ew.Close(); err != nil {
		t.Fatal(err)
	}
	if ebuf.Len() != 0 {
		t.Fatalf("empty archive wrote %d bytes, want 0", ebuf.Len())
	}
}

// Detail 2: header wire order — nameLen uvarint, timestamp varint, sequence
// uvarint, headerSize uvarint, payloadSize uvarint, raw name.
func TestDetail02_HeaderWireOrder(t *testing.T) {
	rng := bbArcRng(t)
	for iter := 0; iter < 40; iter++ {
		name := "n" + strconv.Itoa(rng.Intn(1<<20))
		ts := rng.Int63n(math.MaxInt32) - rng.Int63n(math.MaxInt32)
		seq := rng.Uint64()
		hsz := int64(rng.Intn(64))
		hdrBytes := bbArcRandBytes(rng, int(hsz))
		payload := bbArcRandBytes(rng, rng.Intn(64))
		body := append(append([]byte(nil), hdrBytes...), payload...)
		stream := append([]byte(archive.MagicBytes),
			bbArcEncodeEntry(name, ts, seq, hsz, int64(len(payload)), body)...)
		r := archive.NewReader(bytes.NewReader(stream))
		hdr, err := r.Next()
		if err != nil {
			t.Fatalf("iter %d: Next err=%v", iter, err)
		}
		if hdr.Name != name || hdr.Timestamp != ts || hdr.Sequence != seq ||
			hdr.HeaderSize != hsz || hdr.PayloadSize != int64(len(payload)) {
			t.Fatalf("iter %d: hdr %+v want name=%q ts=%d seq=%d hsz=%d psz=%d",
				iter, hdr, name, ts, seq, hsz, len(payload))
		}
		got, err := io.ReadAll(r)
		if err != nil || !bytes.Equal(got, body) {
			t.Fatalf("iter %d: body %x/%v want %x", iter, got, err, body)
		}
	}
	// Byte-level: first field on wire is nameLen as uvarint.
	var buf bytes.Buffer
	w := archive.NewWriter(&buf)
	name := "abcde"
	if err := w.WriteHeader(&archive.Header{Name: name, HeaderSize: 0, PayloadSize: 0}); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()[len(archive.MagicBytes):]
	nl, n := binary.Uvarint(raw)
	if n <= 0 || int(nl) != len(name) {
		t.Fatalf("first header field: uvarint=%d n=%d, want nameLen=%d", nl, n, len(name))
	}
}

// Detail 3: WriteHeader rejects with distinct errors: closed, nil header,
// negative sizes, overflowing sum, prior incomplete entry. Precedence:
// closed -> nil -> negative field -> incomplete prior -> overflowing sum.
func TestDetail03_WriteHeaderRejections(t *testing.T) {
	// closed beats everything.
	var buf bytes.Buffer
	w := archive.NewWriter(&buf)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteHeader(nil); !errors.Is(err, archive.ErrClosed) {
		t.Fatalf("closed+nil header: %v, want ErrClosed", err)
	}
	if err := w.WriteHeader(&archive.Header{Name: "x", HeaderSize: -1}); !errors.Is(err, archive.ErrClosed) {
		t.Fatalf("closed+negative: %v, want ErrClosed", err)
	}
	// nil beats negative.
	w = archive.NewWriter(&buf)
	if err := w.WriteHeader(nil); !errors.Is(err, archive.ErrNilHeader) {
		t.Fatalf("nil header: %v, want ErrNilHeader", err)
	}
	// negative sizes -> ErrNegativeEntrySize (even when prior entry incomplete
	// is absent and sum overflows too).
	if err := w.WriteHeader(&archive.Header{Name: "x", HeaderSize: -2, PayloadSize: math.MaxInt64}); !errors.Is(err, archive.ErrNegativeEntrySize) {
		t.Fatalf("negative+overflow: %v, want ErrNegativeEntrySize", err)
	}
	// Incomplete prior entry beats overflowing sum.
	w = archive.NewWriter(&buf)
	if err := w.WriteHeader(&archive.Header{Name: "a", HeaderSize: 0, PayloadSize: 10}); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteHeader(&archive.Header{Name: "b", HeaderSize: math.MaxInt64 - 1, PayloadSize: 10}); !errors.Is(err, archive.ErrIncompleteEntry) {
		t.Fatalf("incomplete-prior+overflow: %v, want ErrIncompleteEntry", err)
	}
	// Pure overflow: the wrapped sum is negative, so it reports as
	// ErrNegativeEntrySize rather than a fourth class.
	w = archive.NewWriter(&buf)
	err := w.WriteHeader(&archive.Header{Name: "b", HeaderSize: math.MaxInt64 - 1, PayloadSize: math.MaxInt64 - 1})
	if !errors.Is(err, archive.ErrNegativeEntrySize) {
		t.Fatalf("overflow sum: %v, want ErrNegativeEntrySize (wrapped sum)", err)
	}
}

// Detail 4: a zero-total-payload entry leaves no active entry — Write then
// reports no-active-entry and Close is clean.
func TestDetail04_ZeroPayloadEntry(t *testing.T) {
	var buf bytes.Buffer
	w := archive.NewWriter(&buf)
	// A header-only entry still takes Writes against its HeaderSize.
	if err := w.WriteHeader(&archive.Header{Name: "z", HeaderSize: 3, PayloadSize: 0}); err != nil {
		t.Fatal(err)
	}
	if n, err := w.Write([]byte("abc")); n != 3 || err != nil {
		t.Fatalf("Write header bytes: n=%d err=%v", n, err)
	}
	// HeaderSize+PayloadSize both zero leaves no active entry.
	w = archive.NewWriter(&buf)
	if err := w.WriteHeader(&archive.Header{Name: "zz", HeaderSize: 0, PayloadSize: 0}); err != nil {
		t.Fatal(err)
	}
	if n, err := w.Write([]byte("x")); !errors.Is(err, archive.ErrNoActiveEntry) || n != 0 {
		t.Fatalf("Write after 0/0 header: n=%d err=%v want ErrNoActiveEntry", n, err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close after 0/0 entry: %v", err)
	}
}

// Detail 5: Write — closed -> ErrClosed first; empty slice -> (0,nil) even
// with no entry; no active entry -> ErrNoActiveEntry; oversized write ->
// partial + ErrWriteTooLong; short underlying write -> io.ErrShortWrite;
// completing clears the entry.
func TestDetail05_WriteSemantics(t *testing.T) {
	var buf bytes.Buffer
	w := archive.NewWriter(&buf)
	// Empty slice with no active entry -> (0, nil).
	if n, err := w.Write(nil); n != 0 || err != nil {
		t.Fatalf("Write(nil) fresh: n=%d err=%v", n, err)
	}
	// Closed beats everything, including empty slice.
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(nil); !errors.Is(err, archive.ErrClosed) {
		t.Fatalf("Write(nil) on closed: %v, want ErrClosed", err)
	}
	if _, err := w.Write([]byte("x")); !errors.Is(err, archive.ErrClosed) {
		t.Fatalf("Write on closed: %v, want ErrClosed", err)
	}
	// No active entry.
	w = archive.NewWriter(&buf)
	if _, err := w.Write([]byte("x")); !errors.Is(err, archive.ErrNoActiveEntry) {
		t.Fatalf("Write no entry: %v, want ErrNoActiveEntry", err)
	}
	// Oversized write -> partial + ErrWriteTooLong.
	rng := bbArcRng(t)
	w = archive.NewWriter(&buf)
	if err := w.WriteHeader(&archive.Header{Name: "e", HeaderSize: 0, PayloadSize: 4}); err != nil {
		t.Fatal(err)
	}
	n, err := w.Write(bbArcRandBytes(rng, 10))
	if n != 4 || !errors.Is(err, archive.ErrWriteTooLong) {
		t.Fatalf("oversized Write: n=%d err=%v, want 4+ErrWriteTooLong", n, err)
	}
	// Completing clears the entry.
	w = archive.NewWriter(&buf)
	if err := w.WriteHeader(&archive.Header{Name: "e", HeaderSize: 0, PayloadSize: 3}); err != nil {
		t.Fatal(err)
	}
	if n, err := w.Write([]byte("ab")); n != 2 || err != nil {
		t.Fatalf("partial write: n=%d err=%v", n, err)
	}
	if n, err := w.Write([]byte("c")); n != 1 || err != nil {
		t.Fatalf("completing write: n=%d err=%v", n, err)
	}
	if _, err := w.Write([]byte("x")); !errors.Is(err, archive.ErrNoActiveEntry) {
		t.Fatalf("Write after completion: %v, want ErrNoActiveEntry", err)
	}
	// Short underlying write -> io.ErrShortWrite.
	w = archive.NewWriter(&bbArcShortWriter{limit: 2})
	if err := w.WriteHeader(&archive.Header{Name: "e", HeaderSize: 0, PayloadSize: 8}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("abcdefgh")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short underlying write: %v, want io.ErrShortWrite", err)
	}
}

type bbArcShortWriter struct {
	limit int
	buf   bytes.Buffer
}

func (s *bbArcShortWriter) Write(p []byte) (int, error) {
	if len(p) > s.limit {
		n, _ := s.buf.Write(p[:s.limit])
		return n, nil
	}
	return s.buf.Write(p)
}

// Detail 6: Close idempotent (second -> nil); close with pending payload ->
// ErrIncompleteEntry.
func TestDetail06_CloseIdempotent(t *testing.T) {
	var buf bytes.Buffer
	w := archive.NewWriter(&buf)
	if err := w.WriteHeader(&archive.Header{Name: "e", HeaderSize: 0, PayloadSize: 5}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); !errors.Is(err, archive.ErrIncompleteEntry) {
		t.Fatalf("Close with pending payload: %v, want ErrIncompleteEntry", err)
	}
	w = archive.NewWriter(&buf)
	if err := w.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("second Close: %v, want nil", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("third Close: %v, want nil", err)
	}
}

// Detail 7: Flush — closed -> ErrClosed; forwards to Flush() error or
// Flush() interfaces; nil otherwise.
func TestDetail07_FlushForwarding(t *testing.T) {
	var plain bytes.Buffer
	w := archive.NewWriter(&plain)
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush on bytes.Buffer: %v, want nil", err)
	}
	fw := &bbArcFlusher{}
	w = archive.NewWriter(fw)
	if err := w.WriteHeader(&archive.Header{Name: "e", HeaderSize: 0, PayloadSize: 0}); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush forwarding: %v", err)
	}
	fne := &bbArcFlushNoErr{}
	w = archive.NewWriter(fne)
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() (no error) forwarding: %v", err)
	}
	// Underlying flush error propagates.
	bad := &bbArcBadFlusher{}
	w = archive.NewWriter(bad)
	if err := w.Flush(); !errors.Is(err, errBBArcFlush) {
		t.Fatalf("Flush error propagation: %v", err)
	}
	// Closed -> ErrClosed.
	w = archive.NewWriter(&plain)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); !errors.Is(err, archive.ErrClosed) {
		t.Fatalf("Flush on closed: %v, want ErrClosed", err)
	}
}

var errBBArcFlush = errors.New("flush boom")

type bbArcBadFlusher struct{ bytes.Buffer }

func (b *bbArcBadFlusher) Flush() error { return errBBArcFlush }

// Detail 8: Next discards unread payload first (truncated discard ->
// invalid); validates magic once (wrong -> invalid; short -> EOF); clean
// EOF on first field -> io.EOF, on later fields -> ErrInvalidArchive.
func TestDetail08_NextDiscardAndMagic(t *testing.T) {
	rng := bbArcRng(t)
	// Two entries; reader reads Next twice without consuming payload.
	p1, p2 := bbArcRandBytes(rng, 9), bbArcRandBytes(rng, 5)
	stream := append([]byte(archive.MagicBytes),
		append(bbArcEncodeEntry("a", 1, 1, 0, int64(len(p1)), p1),
			bbArcEncodeEntry("b", 2, 2, 0, int64(len(p2)), p2)...)...)
	r := archive.NewReader(bytes.NewReader(stream))
	h1, err := r.Next()
	if err != nil || h1.Name != "a" {
		t.Fatalf("Next 1: %+v err=%v", h1, err)
	}
	h2, err := r.Next()
	if err != nil || h2.Name != "b" {
		t.Fatalf("Next 2 (should discard p1 first): %+v err=%v", h2, err)
	}
	// Truncated discard -> invalid.
	trunc := append([]byte(archive.MagicBytes), bbArcEncodeEntry("a", 1, 1, 0, 20, p1)...)
	trunc = trunc[:len(trunc)-3] // drop 3 payload bytes
	trunc = append(trunc, bbArcEncodeEntry("b", 2, 2, 0, 0, nil)...)
	r = archive.NewReader(bytes.NewReader(trunc))
	if _, err := r.Next(); err != nil {
		t.Fatalf("Next on trunc stream: %v", err)
	}
	if _, err := r.Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("Next after truncated discard: %v, want ErrInvalidArchive", err)
	}
	// Wrong magic -> invalid.
	bad := append([]byte("NATSXXXX"), bbArcEncodeEntry("a", 1, 1, 0, 0, nil)...)
	r = archive.NewReader(bytes.NewReader(bad))
	if _, err := r.Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("wrong magic: %v, want ErrInvalidArchive", err)
	}
	// Short magic -> EOF or invalid (fewer bytes than magic).
	r = archive.NewReader(bytes.NewReader([]byte("NAT")))
	if _, err := r.Next(); err == nil {
		t.Fatal("short magic should error")
	}
	// Clean EOF on first field -> io.EOF.
	r = archive.NewReader(bytes.NewReader([]byte(archive.MagicBytes)))
	if _, err := r.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("clean EOF after magic: %v, want io.EOF", err)
	}
	r = archive.NewReader(bytes.NewReader(nil))
	if _, err := r.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("empty stream: %v, want io.EOF", err)
	}
	// Partial field after magic -> ErrInvalidArchive.
	r = archive.NewReader(bytes.NewReader(append([]byte(archive.MagicBytes), 0x80)))
	if _, err := r.Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("partial first field: %v, want ErrInvalidArchive", err)
	}
	// Truncated mid-entry header -> ErrInvalidArchive.
	stream2 := append([]byte(archive.MagicBytes), bbArcEncodeEntry("abcdef", 7, 9, 1, 0, nil)...)
	r = archive.NewReader(bytes.NewReader(stream2[:len(stream2)-2]))
	if _, err := r.Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("truncated header: %v, want ErrInvalidArchive", err)
	}
}

// Detail 9: Next field validation — nameLen > 1 MiB, negative decoded
// sizes, or negative sum -> ErrInvalidArchive.
func TestDetail09_NextFieldValidation(t *testing.T) {
	var tmp [binary.MaxVarintLen64]byte
	putU := func(buf *bytes.Buffer, v uint64) { n := binary.PutUvarint(tmp[:], v); buf.Write(tmp[:n]) }
	putV := func(buf *bytes.Buffer, v int64) { n := binary.PutVarint(tmp[:], v); buf.Write(tmp[:n]) }
	build := func(nameLen uint64, ts int64, seq uint64, hsz, psz int64, name string) []byte {
		var buf bytes.Buffer
		buf.WriteString(archive.MagicBytes)
		putU(&buf, nameLen)
		putV(&buf, ts)
		putU(&buf, seq)
		putU(&buf, uint64(hsz))
		putU(&buf, uint64(psz))
		buf.WriteString(name)
		return buf.Bytes()
	}
	// nameLen > 1 MiB.
	if _, err := archive.NewReader(bytes.NewReader(build(1<<20+1, 0, 0, 0, 0, ""))).Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("oversize nameLen: %v, want ErrInvalidArchive", err)
	}
	// Exactly 1 MiB is admitted by the bound (the read itself then fails on
	// short input — still an archive-level error).
	if _, err := archive.NewReader(bytes.NewReader(build(1<<20, 0, 0, 0, 0, ""))).Next(); err == nil {
		t.Fatal("1 MiB nameLen with no data should still error")
	}
	// Negative decoded sizes.
	if _, err := archive.NewReader(bytes.NewReader(build(1, 0, 0, -5, 0, "x"))).Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("negative headerSize: %v, want ErrInvalidArchive", err)
	}
	if _, err := archive.NewReader(bytes.NewReader(build(1, 0, 0, 0, -5, "x"))).Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("negative payloadSize: %v, want ErrInvalidArchive", err)
	}
	// Negative sum: hsz+psz overflows into negative.
	if _, err := archive.NewReader(bytes.NewReader(build(1, 0, 0, math.MaxInt64, math.MaxInt64, "x"))).Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("overflowing sum: %v, want ErrInvalidArchive", err)
	}
}

// Detail 10: Read — sticky error first; empty p -> (0,nil); no/exhausted
// entry -> io.EOF; truncates to remaining; mid-payload EOF ->
// ErrInvalidArchive; completion clears the entry.
func TestDetail10_ReadSemantics(t *testing.T) {
	rng := bbArcRng(t)
	payload := bbArcRandBytes(rng, 20)
	stream := append([]byte(archive.MagicBytes), bbArcEncodeEntry("e", 1, 1, 0, int64(len(payload)), payload)...)
	r := archive.NewReader(bytes.NewReader(stream))
	// Read before Next -> io.EOF (no entry).
	if n, err := r.Read(make([]byte, 4)); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("Read before Next: n=%d err=%v", n, err)
	}
	if _, err := r.Next(); err != nil {
		t.Fatal(err)
	}
	// Empty p -> (0, nil).
	if n, err := r.Read(nil); n != 0 || err != nil {
		t.Fatalf("Read(nil): n=%d err=%v", n, err)
	}
	// Truncates to remaining: ask for more than left.
	buf := make([]byte, 64)
	n, err := r.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Read big: err=%v", err)
	}
	if n != len(payload) || !bytes.Equal(buf[:n], payload) {
		t.Fatalf("Read returned %d bytes, want %d payload", n, len(payload))
	}
	// Exhausted -> io.EOF.
	if n, err := r.Read(buf); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("Read exhausted: n=%d err=%v", n, err)
	}
	// Mid-payload EOF -> ErrInvalidArchive.
	trunc := append([]byte(archive.MagicBytes), bbArcEncodeEntry("e", 1, 1, 0, 10, payload[:4])...)
	r = archive.NewReader(bytes.NewReader(trunc))
	if _, err := r.Next(); err != nil {
		t.Fatal(err)
	}
	got := 0
	for {
		n, err := r.Read(buf)
		got += n
		if err != nil {
			if !errors.Is(err, archive.ErrInvalidArchive) {
				t.Fatalf("mid-payload EOF: %v, want ErrInvalidArchive", err)
			}
			break
		}
	}
	if got != 4 {
		t.Fatalf("read %d payload bytes before error, want 4", got)
	}
	// Sticky: subsequent reads keep returning the stored error.
	if _, err := r.Read(buf); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("sticky error read: %v, want ErrInvalidArchive", err)
	}
	if _, err := r.Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("sticky error Next: %v, want ErrInvalidArchive", err)
	}
}

// Detail 11: all reader errors are sticky via stored u.err.
func TestDetail11_StickyReaderErrors(t *testing.T) {
	// Wrong magic -> sticky invalid.
	r := archive.NewReader(bytes.NewReader([]byte("XXXXXXXXabcdef")))
	if _, err := r.Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("wrong magic: %v", err)
	}
	if _, err := r.Next(); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("second Next after invalid: %v", err)
	}
	if _, err := r.Read(make([]byte, 4)); !errors.Is(err, archive.ErrInvalidArchive) {
		t.Fatalf("Read after invalid: %v", err)
	}
	// io.EOF stickiness on empty stream.
	r = archive.NewReader(bytes.NewReader(nil))
	if _, err := r.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("empty Next: %v", err)
	}
	if _, err := r.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("empty Next 2 (sticky EOF): %v", err)
	}
}
