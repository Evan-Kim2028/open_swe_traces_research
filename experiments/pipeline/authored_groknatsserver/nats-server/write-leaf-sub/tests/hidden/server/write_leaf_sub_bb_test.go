package server

import (
	"bytes"
	"strings"
	"testing"
)

func wlsClient(traceOn bool) (*client, *DummyLogger) {
	c := &client{trace: traceOn}
	c.srv = &Server{opts: &Options{}}
	dl := &DummyLogger{}
	c.srv.SetLogger(dl, false, true)
	return c, dl
}

// TestDetail01 (yes): an empty key writes nothing — no bytes, no CRLF.
func TestDetail01(t *testing.T) {
	var buf bytes.Buffer
	c, _ := wlsClient(false)
	c.writeLeafSub(&buf, _EMPTY_, 5)
	if buf.Len() != 0 {
		t.Fatalf("empty key wrote %q", buf.String())
	}
	c.writeLeafSub(&buf, _EMPTY_, -1)
	if buf.Len() != 0 {
		t.Fatalf("empty key with n<0 wrote %q", buf.String())
	}
}

// TestDetail02 (yes): when n > 0 the line starts with "LS+ " followed by the
// key.
func TestDetail02(t *testing.T) {
	var buf bytes.Buffer
	c, _ := wlsClient(false)
	c.writeLeafSub(&buf, "foo", 1)
	if got := buf.String(); !strings.HasPrefix(got, "LS+ foo") {
		t.Fatalf("n=1 wrote %q, want prefix %q", got, "LS+ foo")
	}
}

// TestDetail03 (doc): when n > 0 and the key contains a space (queue group),
// a decimal encoding of n is appended after another space.
func TestDetail03(t *testing.T) {
	var buf bytes.Buffer
	c, _ := wlsClient(false)
	c.writeLeafSub(&buf, "foo bar", 5)
	if got := buf.String(); got != "LS+ foo bar 5\r\n" {
		t.Fatalf("queue add wrote %q, want %q", got, "LS+ foo bar 5\r\n")
	}
}

// TestDetail04 (yes): the decimal encoding has no leading zeros — most
// significant digit first.
func TestDetail04(t *testing.T) {
	c, _ := wlsClient(false)
	for n, want := range map[int32]string{
		1:          "LS+ k g 1\r\n",
		10:         "LS+ k g 10\r\n",
		999:        "LS+ k g 999\r\n",
		2147483647: "LS+ k g 2147483647\r\n",
	} {
		var buf bytes.Buffer
		c.writeLeafSub(&buf, "k g", n)
		if got := buf.String(); got != want {
			t.Fatalf("n=%d wrote %q, want %q", n, got, want)
		}
	}
}

// TestDetail05 (doc): the queue count is omitted when the key has no space,
// even if n > 1.
func TestDetail05(t *testing.T) {
	var buf bytes.Buffer
	c, _ := wlsClient(false)
	c.writeLeafSub(&buf, "foo", 42)
	if got := buf.String(); got != "LS+ foo\r\n" {
		t.Fatalf("non-queue add wrote %q, want %q", got, "LS+ foo\r\n")
	}
}

// TestDetail06 (yes): when n <= 0 the line is "LS- " plus the key, with no
// count — including for queue keys.
func TestDetail06(t *testing.T) {
	c, _ := wlsClient(false)
	for _, n := range []int32{0, -1, -100} {
		var buf bytes.Buffer
		c.writeLeafSub(&buf, "foo", n)
		if got := buf.String(); got != "LS- foo\r\n" {
			t.Fatalf("n=%d wrote %q, want %q", n, got, "LS- foo\r\n")
		}
	}
	var buf bytes.Buffer
	c.writeLeafSub(&buf, "foo bar", 0)
	if got := buf.String(); got != "LS- foo bar\r\n" {
		t.Fatalf("queue remove wrote %q, want %q", got, "LS- foo bar\r\n")
	}
}

// TestDetail07 (yes): every successful write (non-empty key) ends with CRLF.
func TestDetail07(t *testing.T) {
	c, _ := wlsClient(false)
	for _, tc := range []struct {
		key string
		n   int32
	}{
		{"foo", 1},
		{"foo bar", 9},
		{"foo", 0},
	} {
		var buf bytes.Buffer
		c.writeLeafSub(&buf, tc.key, tc.n)
		if got := buf.String(); !strings.HasSuffix(got, "\r\n") {
			t.Fatalf("key=%q n=%d wrote %q, want CRLF terminator", tc.key, tc.n, got)
		}
	}
}

// TestDetail08 (yes): when tracing is on, LS+ traces "key n" for queue keys
// and just the key otherwise; LS- traces the key.
func TestDetail08(t *testing.T) {
	var buf bytes.Buffer

	c, dl := wlsClient(true)
	c.writeLeafSub(&buf, "foo bar", 7)
	if got := dl.Msg; !strings.Contains(got, "foo bar 7") {
		t.Fatalf("queue add traced %q, want it to contain %q", got, "foo bar 7")
	}

	c, dl = wlsClient(true)
	c.writeLeafSub(&buf, "baz", 7)
	if got := dl.Msg; !strings.Contains(got, "baz") || strings.Contains(got, "baz 7") {
		t.Fatalf("non-queue add traced %q, want the bare key", got)
	}

	c, dl = wlsClient(true)
	c.writeLeafSub(&buf, "baz", 0)
	if got := dl.Msg; !strings.Contains(got, "baz") || strings.Contains(got, "baz 0") {
		t.Fatalf("remove traced %q, want the bare key", got)
	}
}
