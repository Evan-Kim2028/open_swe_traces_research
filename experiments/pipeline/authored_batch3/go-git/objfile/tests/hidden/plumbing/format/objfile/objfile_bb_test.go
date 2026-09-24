package objfile

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"io"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	format "example.internal/gitkit/v6/plumbing/format/config"
)

func inflated(t *testing.T, raw []byte) string {
	t.Helper()
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("zlib open: %v", err)
	}
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("zlib read: %v", err)
	}
	return string(body)
}

func zlibed(t *testing.T, raw string) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

// TestDetail01: stream is zlib; header is `<type> SP <size> NUL` then body.
func TestDetail01(t *testing.T) {
	r, err := NewReader(zlibed(t, "blob 5\x00hello"), format.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	typ, size, err := r.Header()
	if err != nil {
		t.Fatal(err)
	}
	if typ != plumbing.BlobObject || size != 5 {
		t.Fatalf("header = %v %d", typ, size)
	}
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" {
		t.Fatalf("body = %q", body)
	}
	r.Close()
}

// TestDetail02: the 32-byte bound is shared across the whole header — a
// 20-digit size blows it even with a short type. "blob " (5) + 30-digit
// size + NUL = 36 > 32.
func TestDetail02(t *testing.T) {
	hdr := "blob " + strings.Repeat("9", 30) + "\x00"
	r, err := NewReader(zlibed(t, hdr+"x"), format.SHA1)
	if err != nil {
		return // failure at construction is an acceptable shape
	}
	_, _, herr := r.Header()
	if herr == nil {
		t.Fatal("36-byte header accepted")
	}
	if !errors.Is(herr, ErrHeaderTooLong) && !errors.Is(herr, ErrHeader) {
		t.Fatalf("header error = %v, want too-long or invalid", herr)
	}
}

// TestDetail03: an unparseable type fails before size is read; a non-numeric
// size is a generic header failure.
func TestDetail03(t *testing.T) {
	for _, raw := range []string{"bogus 5\x00hello", "blob xyz\x00hello"} {
		r, err := NewReader(zlibed(t, raw), format.SHA1)
		if err != nil {
			continue
		}
		if _, _, herr := r.Header(); herr == nil {
			t.Fatalf("raw %q header accepted", raw)
		}
	}
}

// TestDetail04 (shape — Inferable: no): a negative numeric size is not
// validated by the reader the way the writer validates it — whichever way
// the reader treats it, it must not panic; the writer must refuse it.
func TestDetail04(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, format.SHA1)
	if err := w.WriteHeader(plumbing.BlobObject, -3); !errors.Is(err, ErrNegativeSize) {
		t.Fatalf("writer negative size = %v, want ErrNegativeSize", err)
	}

	r, err := NewReader(zlibed(t, "blob -3\x00abc"), format.SHA1)
	if err != nil {
		return // refusal at construction is acceptable
	}
	_, _, _ = r.Header() // outcome unpinned — must not panic
}

// TestDetail05: byte-budget overflow mid-field and end-of-stream mid-header
// are different failures.
func TestDetail05(t *testing.T) {
	long := "blob " + strings.Repeat("1", 40) // never terminates within budget
	r, err := NewReader(zlibed(t, long), format.SHA1)
	if err == nil {
		_, _, err = r.Header()
	}
	if err == nil {
		t.Fatal("over-budget header accepted")
	}
	budgetErr := err

	r2, err := NewReader(zlibed(t, "blob 5"), format.SHA1) // EOF before NUL
	if err == nil {
		_, _, err = r2.Header()
	}
	if err == nil {
		t.Fatal("truncated header accepted")
	}
	eofErr := err
	// The distinguishable commitment: budget-overflow is ErrHeaderTooLong,
	// truncation is not.
	if !errors.Is(budgetErr, ErrHeaderTooLong) {
		t.Fatalf("budget overflow = %v, want ErrHeaderTooLong", budgetErr)
	}
	if errors.Is(eofErr, ErrHeaderTooLong) {
		t.Fatalf("truncated header reported as too-long: %v", eofErr)
	}
}

// TestDetail06 (shape — Inferable: no): Read before a successful Header
// fails with a dedicated error; Hash before it returns a zero hash sized to
// the format, not absent.
func TestDetail06(t *testing.T) {
	r, err := NewReader(zlibed(t, "blob 5\x00hello"), format.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 8)
	if _, err := r.Read(buf); !errors.Is(err, ErrHeaderNotRead) {
		t.Fatalf("Read before Header = %v, want ErrHeaderNotRead", err)
	}
	h := r.Hash()
	if !h.IsZero() {
		t.Fatalf("Hash before header = %v, want zero hash", h)
	}
	if len(h.String()) != 40 { // SHA-1 hex width — sized to the format
		t.Fatalf("Hash hex = %q, want 40 chars", h.String())
	}
}

// TestDetail07: the hash covers `<type> <size>\x00` + body — seeded with the
// declared type and size even though only body bytes flow through Read.
func TestDetail07(t *testing.T) {
	r, err := NewReader(zlibed(t, "blob 5\x00hello"), format.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.Header(); err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, r)
	want := sha1.Sum([]byte("blob 5\x00hello"))
	if r.Hash().String() != hexEncode(want[:]) {
		t.Fatalf("Hash = %v, want sha1 of header+body", r.Hash())
	}
}

func hexEncode(b []byte) string {
	const digits = "0123456789abcdef"
	var sb strings.Builder
	for _, c := range b {
		sb.WriteByte(digits[c>>4])
		sb.WriteByte(digits[c&0xf])
	}
	return sb.String()
}

// TestDetail08: header write rejects an invalid type before emitting bytes
// and refuses a negative size. (The over-bound clause is unreachable through
// the public API: the longest possible header — 9-char type + 19-digit
// int64 size + delimiters — is 30 bytes < 32.)
func TestDetail08(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, format.SHA1)
	if err := w.WriteHeader(plumbing.InvalidObject, 3); err == nil {
		t.Fatal("invalid type accepted")
	}
	if buf.Len() != 0 {
		t.Fatalf("invalid type emitted %d bytes", buf.Len())
	}
	if err := w.WriteHeader(plumbing.BlobObject, -1); !errors.Is(err, ErrNegativeSize) {
		t.Fatalf("negative size = %v", err)
	}
}

// TestDetail09 (shape — Inferable: no): writing more than the declared size
// reports the overflow, and the truncated bytes that fit are committed —
// a partial write is not rolled back.
func TestDetail09(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, format.SHA1)
	if err := w.WriteHeader(plumbing.BlobObject, 3); err != nil {
		t.Fatal(err)
	}
	_, err := w.Write([]byte("abcde"))
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("overflow write = %v, want ErrOverflow", err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if body := inflated(t, buf.Bytes()); !strings.Contains(body, "abc") {
		t.Fatalf("stream body = %q, want the fitting bytes committed", body)
	}
}

// TestDetail10 (shape — Inferable: no): a write after the declared size is
// exhausted reports overflow rather than silently ignoring it; the earlier
// fitting bytes stay committed.
func TestDetail10(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, format.SHA1)
	if err := w.WriteHeader(plumbing.BlobObject, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("d")); !errors.Is(err, ErrOverflow) {
		t.Fatalf("post-exhaust write = %v, want ErrOverflow", err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if body := inflated(t, buf.Bytes()); !strings.Contains(body, "abc") {
		t.Fatalf("stream body = %q, want committed prefix preserved", body)
	}
}

// TestDetail11 (shape — Inferable: no): Write before WriteHeader does not
// produce a clean nil error — it fails loudly (panic or error).
func TestDetail11(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, format.SHA1)
	defer func() {
		_ = recover() // panic is an acceptable loud failure
	}()
	_, err := w.Write([]byte("x"))
	if err == nil {
		t.Fatal("Write before WriteHeader succeeded silently")
	}
}

// TestDetail12: Close is idempotent — same error every repeat; neither
// direction closes the wrapped stream.
func TestDetail12(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, format.SHA1)
	if err := w.WriteHeader(plumbing.BlobObject, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	e1 := w.Close()
	e2 := w.Close()
	if e1 != e2 {
		t.Fatalf("Close errors differ: %v vs %v", e1, e2)
	}
	// Wrapped writer not closed: bytes.Buffer has no Close, so check reader side.
	underlying := &closeTracker{Reader: zlibed(t, "blob 5\x00hello")}
	r, err := NewReader(underlying, format.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if underlying.closed {
		t.Fatal("Reader.Close closed the wrapped reader")
	}
}

type closeTracker struct {
	*bytes.Reader
	closed bool
}

func (c *closeTracker) Close() error { c.closed = true; return nil }
