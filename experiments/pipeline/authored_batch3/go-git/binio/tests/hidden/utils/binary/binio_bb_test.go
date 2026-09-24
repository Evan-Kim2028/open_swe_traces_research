package binary

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

// errAfterBytes fails the write once more than n bytes have been accepted.
// attempted counts every byte ever handed to Write, accepted or not.
type errAfterBytes struct {
	n         int
	got       int64
	attempted int64
}

func (w *errAfterBytes) Write(p []byte) (int, error) {
	atomic.AddInt64(&w.attempted, int64(len(p)))
	if atomic.LoadInt64(&w.got)+int64(len(p)) > int64(w.n) {
		return 0, errors.New("write budget exhausted")
	}
	atomic.AddInt64(&w.got, int64(len(p)))
	return len(p), nil
}

// oneShotReader returns all of data on the first Read together with err.
type oneShotReader struct {
	data []byte
	err  error
	done bool
}

func (r *oneShotReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	return copy(p, r.data), r.err
}

// zeroNilReader returns (0, nil) on every Read.
type zeroNilReader struct{}

func (zeroNilReader) Read(p []byte) (int, error) { return 0, nil }

// countingReader wraps a reader and records total bytes delivered.
type countingReader struct {
	r io.Reader
	n int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	atomic.AddInt64(&r.n, int64(n))
	return n, err
}

// TestDetail01: scalar Read/Write apply BigEndian to each argument in order;
// the first failure stops the sequence mid-way.
func TestDetail01(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, uint16(0x1234), uint32(0xDEADBEEF), uint8(0x09)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	want := []byte{0x12, 0x34, 0xDE, 0xAD, 0xBE, 0xEF, 0x09}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("Write bytes = %x, want %x", buf.Bytes(), want)
	}

	var a uint16
	var b uint32
	var c uint8
	if err := Read(bytes.NewReader(want), &a, &b, &c); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if a != 0x1234 || b != 0xDEADBEEF || c != 0x09 {
		t.Fatalf("Read got a=%x b=%x c=%x", a, b, c)
	}

	// First failure stops mid-way: a writer that accepts only the first
	// argument's bytes must see the error and hold only those bytes.
	w := &errAfterBytes{n: 2}
	if err := Write(w, uint16(0xAAAA), uint32(0xBBBBBBBB)); err == nil {
		t.Fatal("Write to failing writer: want error, got nil")
	}
	if w.got != 2 {
		t.Fatalf("Write continued past first failure: wrote %d bytes, want 2", w.got)
	}

	var x uint16
	var y uint16
	if err := Read(bytes.NewReader([]byte{0x11, 0x22}), &x, &y); err == nil {
		t.Fatal("Read past end: want error, got nil")
	}
	if x != 0x1122 || y != 0 {
		t.Fatalf("Read did not stop at first failure: x=%x y=%x", x, y)
	}
}

// TestDetail02: the Git VLQ is the OFFSET variant — each continuation byte
// increments the accumulator before shifting in the next 7 bits.
func TestDetail02(t *testing.T) {
	cases := []struct {
		enc  []byte
		want int64
	}{
		{[]byte{0x00}, 0},
		{[]byte{0x7f}, 127},
		{[]byte{0x80, 0x00}, 128},
		{[]byte{0xff, 0x7f}, 16511},
		{[]byte{0x80, 0x80, 0x00}, 16512},
		{[]byte{0xff, 0xff, 0x7f}, 2113663},
	}
	for _, c := range cases {
		got, err := ReadVariableWidthInt(bytes.NewReader(c.enc))
		if err != nil {
			t.Fatalf("ReadVariableWidthInt(%x): %v", c.enc, err)
		}
		if got != c.want {
			t.Fatalf("ReadVariableWidthInt(%x) = %d, want %d", c.enc, got, c.want)
		}
	}
}

// TestDetail03: write side emits the low 7 bits first, then prepends higher
// groups as 0x80|group after a pre-decrement — big group first on the wire.
func TestDetail03(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteVariableWidthInt(&buf, 128); err != nil {
		t.Fatalf("WriteVariableWidthInt(128): %v", err)
	}
	if !bytes.Equal(buf.Bytes(), []byte{0x80, 0x00}) {
		t.Fatalf("WriteVariableWidthInt(128) = %x, want 80 00", buf.Bytes())
	}

	for _, v := range []int64{1, 127, 128, 300, 16511, 16512, 2113663, 1 << 40} {
		buf.Reset()
		if err := WriteVariableWidthInt(&buf, v); err != nil {
			t.Fatalf("WriteVariableWidthInt(%d): %v", v, err)
		}
		got, err := ReadVariableWidthInt(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatalf("ReadVariableWidthInt(%x): %v", buf.Bytes(), err)
		}
		if got != v {
			t.Fatalf("roundtrip %d: got %d (enc %x)", v, got, buf.Bytes())
		}
	}
}

// TestDetail04: zero encodes as a single 0x00 byte.
func TestDetail04(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteVariableWidthInt(&buf, 0); err != nil {
		t.Fatalf("WriteVariableWidthInt(0): %v", err)
	}
	if !bytes.Equal(buf.Bytes(), []byte{0x00}) {
		t.Fatalf("WriteVariableWidthInt(0) = %x, want single 00 byte", buf.Bytes())
	}
}

// TestDetail05: a negative input to the writer never terminates — there is no
// sign guard. Shape assertion only: the call must not return normally within
// the window. A write-through implementation keeps handing bytes to the
// writer (observed via attempted); a buffering implementation simply never
// comes back.
func TestDetail05(t *testing.T) {
	bounded := &errAfterBytes{n: 1 << 16}

	type res struct {
		err   error
		panic any
	}
	done := make(chan res, 1)
	go func() {
		var r res
		defer func() {
			if p := recover(); p != nil {
				r.panic = p
			}
			done <- r
		}()
		r.err = WriteVariableWidthInt(bounded, -1)
	}()

	select {
	case r := <-done:
		switch {
		case r.panic != nil:
			t.Fatalf("WriteVariableWidthInt(-1) panicked: %v", r.panic)
		case r.err == nil:
			t.Fatal("WriteVariableWidthInt(-1) returned without error — a sign guard the format does not have")
		case atomic.LoadInt64(&bounded.attempted) <= 16:
			t.Fatalf("WriteVariableWidthInt(-1) returned after attempting only %d bytes — not an unterminated stream", bounded.attempted)
		}
		// Returned an error only after emitting far more than any
		// legitimate encoding: non-terminating shape holds.
	case <-time.After(2 * time.Second):
		// Still running: the arithmetic-shift loop never exits for a
		// negative input.
	}
}

// TestDetail06: the reader bounds the accumulator before increment+shift and
// fails with the overflow error rather than wrapping.
func TestDetail06(t *testing.T) {
	enc := append(bytes.Repeat([]byte{0xFF}, 12), 0x7F)
	got, err := ReadVariableWidthInt(bytes.NewReader(enc))
	if err == nil {
		t.Fatalf("ReadVariableWidthInt overflow: got %d, want error", got)
	}
	if !errors.Is(err, ErrIntegerOverflow) {
		t.Fatalf("ReadVariableWidthInt overflow error = %v, want ErrIntegerOverflow", err)
	}
}

// TestDetail07: ReadUntil drops the delimiter; on end-of-input before the
// delimiter it returns no bytes plus the error; an already-buffered reader is
// used in place (the bytes past the delimiter stay buffered).
func TestDetail07(t *testing.T) {
	got, err := ReadUntil(bytes.NewReader([]byte("abc\nrest")), '\n')
	if err != nil {
		t.Fatalf("ReadUntil: %v", err)
	}
	if string(got) != "abc" {
		t.Fatalf("ReadUntil = %q, want %q (delimiter dropped)", got, "abc")
	}

	got, err = ReadUntil(bytes.NewReader([]byte("abc-no-delim")), '\n')
	if err == nil {
		t.Fatalf("ReadUntil EOF before delimiter: got %q, want error", got)
	}
	if len(got) != 0 {
		t.Fatalf("ReadUntil EOF before delimiter returned %d bytes; collected bytes must be discarded", len(got))
	}

	br := bufio.NewReader(bytes.NewReader([]byte("abc\nrest")))
	got, err = ReadUntil(br, '\n')
	if err != nil {
		t.Fatalf("ReadUntil on *bufio.Reader: %v", err)
	}
	if string(got) != "abc" {
		t.Fatalf("ReadUntil on *bufio.Reader = %q, want %q", got, "abc")
	}
	rest, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		t.Fatalf("buffered remainder: %v", err)
	}
	if rest != "rest" {
		t.Fatalf("ReadUntil did not use the buffered reader in place: remainder %q, want %q", rest, "rest")
	}
}

// TestDetail08: the buffered variant returns no bytes plus the error whenever
// the underlying read reports ANY error; a delivered value is never paired
// with an error.
func TestDetail08(t *testing.T) {
	sentinel := errors.New("sentinel mid-stream failure")
	r := &oneShotReader{data: []byte("ab"), err: sentinel}
	got, err := ReadUntilFromBufioReader(bufio.NewReader(r), '\n')
	if !errors.Is(err, sentinel) {
		t.Fatalf("ReadUntilFromBufioReader error = %v, want %v", err, sentinel)
	}
	if len(got) != 0 {
		t.Fatalf("ReadUntilFromBufioReader returned %d bytes alongside the error", len(got))
	}

	// Value plus end-of-input in a single read: either a clean (value, nil)
	// or an empty (no bytes, err) result — never both data and error.
	r2 := &oneShotReader{data: []byte("abc\n"), err: io.EOF}
	got, err = ReadUntilFromBufioReader(bufio.NewReader(r2), '\n')
	if err == nil {
		if string(got) != "abc" {
			t.Fatalf("ReadUntilFromBufioReader = %q, want %q", got, "abc")
		}
	} else if len(got) != 0 {
		t.Fatalf("ReadUntilFromBufioReader returned data %q AND error %v", got, err)
	}
}

// TestDetail09: binary detection reads at most the sniff-window size, stops at
// the first NUL without draining, reports false on short clean input, and
// leaves the reader positioned after the sniff window.
func TestDetail09(t *testing.T) {
	clean := bytes.Repeat([]byte("a"), 16000)
	cr := &countingReader{r: bytes.NewReader(clean)}
	isBin, err := IsBinary(cr)
	if err != nil {
		t.Fatalf("IsBinary: %v", err)
	}
	if isBin {
		t.Fatal("IsBinary(clean input) = true, want false")
	}
	if cr.n > 8000 {
		t.Fatalf("IsBinary read %d bytes, exceeds the 8000-byte sniff window", cr.n)
	}

	withNul := append(bytes.Repeat([]byte("a"), 100), append([]byte{0x00}, bytes.Repeat([]byte("a"), 16000)...)...)
	cr = &countingReader{r: bytes.NewReader(withNul)}
	isBin, err = IsBinary(cr)
	if err != nil {
		t.Fatalf("IsBinary: %v", err)
	}
	if !isBin {
		t.Fatal("IsBinary(NUL at offset 100) = false, want true")
	}
	if cr.n > 8000 {
		t.Fatalf("IsBinary read %d bytes on NUL input, exceeds sniff window", cr.n)
	}

	isBin, err = IsBinary(bytes.NewReader([]byte("short")))
	if err != nil {
		t.Fatalf("IsBinary(short): %v", err)
	}
	if isBin {
		t.Fatal("IsBinary(short clean input) = true, want false")
	}
}

// TestDetail10: a reader returning (0, nil) mid-stream is treated as
// end-of-data, not spun on.
func TestDetail10(t *testing.T) {
	done := make(chan struct {
		b   bool
		err error
	}, 1)
	go func() {
		b, err := IsBinary(zeroNilReader{})
		done <- struct {
			b   bool
			err error
		}{b, err}
	}()
	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("IsBinary over (0,nil) reader: %v", res.err)
		}
		if res.b {
			t.Fatal("IsBinary over (0,nil) reader = true, want false")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("IsBinary spun on a (0, nil) reader")
	}
}

// TestDetail11: the sniff buffer is pooled and reused — content beyond the
// bytes just read is never inspected, so consecutive calls stay independent.
func TestDetail11(t *testing.T) {
	if b, err := IsBinary(bytes.NewReader(bytes.Repeat([]byte{0x00}, 8000))); err != nil || !b {
		t.Fatalf("IsBinary(NUL-filled window) = %v, %v", b, err)
	}
	if b, err := IsBinary(bytes.NewReader([]byte("clean text"))); err != nil || b {
		t.Fatalf("IsBinary(clean) after binary call = %v, %v — stale buffer content inspected", b, err)
	}
	if b, err := IsBinary(bytes.NewReader(bytes.Repeat([]byte("z"), 16000))); err != nil || b {
		t.Fatalf("IsBinary(long clean) = %v, %v", b, err)
	}
}

// TestDetail12: ReadUntil has no length bound — a missing delimiter reads to
// end-of-input and never fails on size.
func TestDetail12(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 256*1024)
	cr := &countingReader{r: bytes.NewReader(payload)}
	got, err := ReadUntil(cr, '\n')
	if err == nil {
		t.Fatalf("ReadUntil without delimiter: got %d bytes, want error", len(got))
	}
	if len(got) != 0 {
		t.Fatalf("ReadUntil without delimiter returned %d bytes; want them discarded", len(got))
	}
	if cr.n < int64(len(payload)) {
		t.Fatalf("ReadUntil stopped after %d of %d bytes — bounded like a header reader", cr.n, len(payload))
	}
}
