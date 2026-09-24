package ioutil

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

// blockingWriter parks inside Write until release is closed, then writes.
type blockingWriter struct {
	release chan struct{}
	wrote   chan struct{}
}

func newBlockingWriter() *blockingWriter {
	return &blockingWriter{release: make(chan struct{}), wrote: make(chan struct{}, 1)}
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	<-w.release
	w.wrote <- struct{}{}
	return len(p), nil
}

// blockingReader parks inside Read until release is closed.
type blockingReader struct {
	release chan struct{}
	data    []byte
}

func (r *blockingReader) Read(p []byte) (int, error) {
	<-r.release
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

// panicWriter always panics inside Write.
type panicWriter struct{}

func (panicWriter) Write(p []byte) (int, error) { panic("boom") }

// errReader fails with err after delivering data once.
type errReader struct {
	data []byte
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.data != nil {
		n := copy(p, r.data)
		r.data = r.data[n:]
		if len(r.data) == 0 && r.err != nil {
			return n, r.err
		}
		return n, nil
	}
	return 0, r.err
}

// seqCloser records close order into seq.
type seqCloser struct {
	seq *[]string
	id  string
	err error
}

func (c *seqCloser) Close() error {
	*c.seq = append(*c.seq, c.id)
	return c.err
}

// TestDetail01: NonEmptyReader returns ErrEmptyReader on empty input and a
// ReadPeeker with the first byte still unread on non-empty input.
func TestDetail01(t *testing.T) {
	if _, err := NonEmptyReader(bytes.NewReader(nil)); !errors.Is(err, ErrEmptyReader) {
		t.Fatalf("NonEmptyReader(empty) = %v, want ErrEmptyReader", err)
	}

	sentinel := errors.New("read failure")
	if _, err := NonEmptyReader(&errReader{err: sentinel}); !errors.Is(err, sentinel) {
		t.Fatalf("NonEmptyReader(failing) = %v, want %v", err, sentinel)
	}

	r, err := NonEmptyReader(bytes.NewReader([]byte("hello")))
	if err != nil {
		t.Fatalf("NonEmptyReader: %v", err)
	}
	rp, ok := r.(ReadPeeker)
	if !ok {
		t.Fatalf("NonEmptyReader did not return a ReadPeeker: %T", r)
	}
	b, err := rp.Peek(1)
	if err != nil || len(b) != 1 || b[0] != 'h' {
		t.Fatalf("Peek(1) = %v, %v; want 'h'", b, err)
	}
	all, err := io.ReadAll(rp)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(all) != "hello" {
		t.Fatalf("peeked byte was consumed: got %q, want %q", all, "hello")
	}
}

// TestDetail02: the context writer waits for an in-flight write before
// reporting ctx.Err() — the underlying writer is quiescent when Write returns.
func TestDetail02(t *testing.T) {
	w := newBlockingWriter()
	ctx, cancel := context.WithCancel(context.Background())
	cw := NewContextWriter(ctx, w)

	done := make(chan error, 1)
	go func() { _, err := cw.Write([]byte("x")); done <- err }()

	// Let the write get in flight, then cancel while it is still blocked.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		t.Fatalf("Write returned %v before the in-flight write completed", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(w.release)
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Write after cancel = %v, want ctx.Err()", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Write did not return after the in-flight write completed")
	}

	select {
	case <-w.wrote:
	default:
		t.Fatal("underlying write never ran")
	}
}

// TestDetail03: a panic in the underlying writer surfaces as a normal error,
// not a panic through the wrapper. Shape: error non-nil, no panic escapes.
func TestDetail03(t *testing.T) {
	w := NewContextWriter(context.Background(), panicWriter{})
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("panic escaped the context writer: %v", p)
		}
	}()
	n, err := w.Write([]byte("x"))
	if err == nil {
		t.Fatalf("Write over panicking writer: n=%d err=nil, want error", n)
	}
}

// TestDetail04: the context reader copies out of its internal window — bytes
// a still-blocked underlying read later delivers never land in the caller's
// buffer.
func TestDetail04(t *testing.T) {
	r := &blockingReader{release: make(chan struct{}), data: []byte("late")}
	ctx, cancel := context.WithCancel(context.Background())
	cr := NewContextReader(ctx, r)

	buf := []byte("________")
	done := make(chan error, 1)
	go func() { _, err := cr.Read(buf); done <- err }()

	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Read after cancel = %v, want ctx.Err()", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not return after cancellation")
	}

	close(r.release)
	time.Sleep(100 * time.Millisecond)
	if string(buf) != "________" {
		t.Fatalf("caller buffer written by the post-cancel read: %q", buf)
	}
}

// TestDetail05: on ctx.Done the context reader closes its closer before
// returning, when one was attached.
func TestDetail05(t *testing.T) {
	r := &blockingReader{release: make(chan struct{})}
	defer close(r.release)
	ctx, cancel := context.WithCancel(context.Background())

	var closed int32
	cr := NewContextReaderWithCloser(ctx, r, CloserFunc(func() error {
		atomic.StoreInt32(&closed, 1)
		return nil
	}))

	done := make(chan error, 1)
	go func() { _, err := cr.Read(make([]byte, 8)); done <- err }()
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Read after cancel = %v, want ctx.Err()", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not return after cancellation")
	}
	if atomic.LoadInt32(&closed) != 1 {
		t.Fatal("attached closer was not closed on cancellation")
	}
}

// TestDetail06: ReadFinished answers false only when BOTH this context is
// done AND the read's error is a cancellation error.
func TestDetail06(t *testing.T) {
	live := context.Background()
	dead, cancel := context.WithCancel(context.Background())
	cancel()

	ioErr := errors.New("io failure")
	for _, c := range []struct {
		ctx  context.Context
		err  error
		want bool
	}{
		{live, nil, true},
		{live, ioErr, true},
		{live, context.Canceled, true},
		{live, context.DeadlineExceeded, true},
		{dead, nil, true},
		{dead, ioErr, true},
		{dead, context.Canceled, false},
		{dead, context.DeadlineExceeded, false},
	} {
		if got := ReadFinished(c.ctx, c.err); got != c.want {
			t.Fatalf("ReadFinished(ctxDone=%v, err=%v) = %v, want %v",
				c.ctx.Err() != nil, c.err, got, c.want)
		}
	}
}

// TestDetail07: CheckClose writes the Close error only into a nil *err.
func TestDetail07(t *testing.T) {
	closeErr := errors.New("close failed")
	c := CloserFunc(func() error { return closeErr })

	var err error
	CheckClose(c, &err)
	if !errors.Is(err, closeErr) {
		t.Fatalf("CheckClose into nil *err: %v, want %v", err, closeErr)
	}

	prior := errors.New("prior error")
	err = prior
	CheckClose(c, &err)
	if !errors.Is(err, prior) {
		t.Fatalf("CheckClose over existing error: %v, want %v", err, prior)
	}

	err = nil
	CheckClose(CloserFunc(func() error { return nil }), &err)
	if err != nil {
		t.Fatalf("CheckClose clean close set err = %v", err)
	}
}

// TestDetail08: OnError wrappers notify on any non-EOF error; EOF is silent.
func TestDetail08(t *testing.T) {
	sentinel := errors.New("boom")
	var got []error
	notify := func(e error) { got = append(got, e) }

	r := NewReaderOnError(&errReader{data: []byte("ab"), err: sentinel}, notify)
	buf := make([]byte, 8)
	_, _ = r.Read(buf)
	if len(got) != 1 || !errors.Is(got[0], sentinel) {
		t.Fatalf("reader notify = %v, want [%v]", got, sentinel)
	}

	got = nil
	r = NewReaderOnError(&errReader{err: io.EOF}, notify)
	_, _ = r.Read(buf)
	if len(got) != 0 {
		t.Fatalf("EOF notified: %v", got)
	}

	got = nil
	rc := NewReadCloserOnError(io.NopCloser(&errReader{err: sentinel}), notify)
	_, _ = rc.Read(buf)
	if len(got) != 1 {
		t.Fatalf("readcloser notify = %v", got)
	}

	got = nil
	w := NewWriterOnError(&errWriter{err: sentinel}, notify)
	_, _ = w.Write([]byte("x"))
	if len(got) != 1 || !errors.Is(got[0], sentinel) {
		t.Fatalf("writer notify = %v, want [%v]", got, sentinel)
	}

	got = nil
	wc := NewWriteCloserOnError(&nopErrWriteCloser{err: sentinel}, notify)
	_, _ = wc.Write([]byte("x"))
	if len(got) != 1 {
		t.Fatalf("writecloser notify = %v", got)
	}
}

type errWriter struct{ err error }

func (w *errWriter) Write(p []byte) (int, error) { return 0, w.err }

type nopErrWriteCloser struct{ err error }

func (w *nopErrWriteCloser) Write(p []byte) (int, error) { return 0, w.err }
func (w *nopErrWriteCloser) Close() error                { return nil }

// TestDetail09: NewReaderUsingReaderAt is a stateful sequential reader —
// each Read advances an internal offset starting at the given one.
func TestDetail09(t *testing.T) {
	r := NewReaderUsingReaderAt(bytes.NewReader([]byte("0123456789")), 2)
	buf := make([]byte, 3)
	if n, err := r.Read(buf); err != nil || string(buf[:n]) != "234" {
		t.Fatalf("first Read = %q, %v; want \"234\"", buf[:n], err)
	}
	if n, err := r.Read(buf); err != nil || string(buf[:n]) != "567" {
		t.Fatalf("second Read = %q, %v; want \"567\" (offset advanced)", buf[:n], err)
	}
	var rest []byte
	for {
		n, err := r.Read(buf)
		rest = append(rest, buf[:n]...)
		if err != nil {
			if err != io.EOF {
				t.Fatalf("tail Read: %v", err)
			}
			break
		}
	}
	if string(rest) != "89" {
		t.Fatalf("tail = %q, want \"89\"", rest)
	}
}

// TestDetail10: the closer combiners chain both closes and surface the first
// error; the WithCloser variant runs its function after the underlying close.
func TestDetail10(t *testing.T) {
	seq := []string{}
	errA := errors.New("a")
	errB := errors.New("b")

	// Plain combiner: the attached closer runs and its error surfaces.
	rc := NewReadCloser(io.NopCloser(nil), &seqCloser{seq: &seq, id: "c", err: errA})
	if err := rc.Close(); !errors.Is(err, errA) {
		t.Fatalf("NewReadCloser.Close = %v, want %v", err, errA)
	}
	if len(seq) != 1 || seq[0] != "c" {
		t.Fatalf("attached closer not run: %v", seq)
	}

	wc := NewWriteCloser(io.Discard, &seqCloser{seq: &seq, id: "c", err: errA})
	if err := wc.Close(); !errors.Is(err, errA) {
		t.Fatalf("NewWriteCloser.Close = %v, want %v", err, errA)
	}
	if len(seq) != 2 || seq[1] != "c" {
		t.Fatalf("write closer not run: %v", seq)
	}

	// WithCloser: underlying close runs first, then the func; both run even
	// when the first fails, and the first error is the one returned.
	order := []string{}
	rc = NewReadCloserWithCloser(&markReadCloser{order: &order, id: "inner"}, func() error {
		order = append(order, "func")
		return nil
	})
	if err := rc.Close(); err != nil {
		t.Fatalf("WithCloser.Close: %v", err)
	}
	if len(order) != 2 || order[0] != "inner" || order[1] != "func" {
		t.Fatalf("closer func did not run after underlying close: %v", order)
	}

	rc = NewReadCloserWithCloser(&nopErrReadCloser{err: errA}, func() error { return errB })
	if err := rc.Close(); !errors.Is(err, errA) {
		t.Fatalf("two failing closes returned %v, want the first error", err)
	}

	rc = NewReadCloserWithCloser(io.NopCloser(nil), func() error { return errB })
	if err := rc.Close(); !errors.Is(err, errB) {
		t.Fatalf("only func fails: Close = %v, want %v", err, errB)
	}
}

type nopErrReadCloser struct{ err error }

func (r *nopErrReadCloser) Read(p []byte) (int, error) { return 0, io.EOF }
func (r *nopErrReadCloser) Close() error               { return r.err }

type markReadCloser struct {
	order *[]string
	id    string
}

func (r *markReadCloser) Read(p []byte) (int, error) { return 0, io.EOF }
func (r *markReadCloser) Close() error {
	*r.order = append(*r.order, r.id)
	return nil
}

// TestDetail11: CopyBufferPool copies correctly end to end (pooling itself
// is internal; the contract is a correct bounded copy).
func TestDetail11(t *testing.T) {
	src := bytes.Repeat([]byte("data"), 4096)
	var dst bytes.Buffer
	n, err := CopyBufferPool(&dst, bytes.NewReader(src))
	if err != nil {
		t.Fatalf("CopyBufferPool: %v", err)
	}
	if n != int64(len(src)) || !bytes.Equal(dst.Bytes(), src) {
		t.Fatalf("CopyBufferPool n=%d len=%d, want %d", n, len(dst.Bytes()), len(src))
	}

	n, err = CopyBufferPool(&bytes.Buffer{}, bytes.NewReader(nil))
	if err != nil || n != 0 {
		t.Fatalf("CopyBufferPool(empty) = %d, %v", n, err)
	}
}

// TestDetail12: WriteNopCloser's Close is a no-op — nil error, writer untouched.
func TestDetail12(t *testing.T) {
	w := &bytes.Buffer{}
	wc := WriteNopCloser(w)
	if err := wc.Close(); err != nil {
		t.Fatalf("Close = %v, want nil", err)
	}
	if _, err := wc.Write([]byte("x")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if w.String() != "x" {
		t.Fatalf("underlying writer not written through: %q", w.String())
	}
	if err := wc.Close(); err != nil {
		t.Fatalf("second Close = %v, want nil", err)
	}
}
