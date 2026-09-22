package server

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func ppBb(c *client, in string) error {
	return c.parse([]byte(in))
}

func ppBbDummy(kind int) *client {
	return &client{srv: New(&defaultServerOptions), kind: kind, msubs: -1, mpay: -1, mcl: MAX_CONTROL_LINE_SIZE}
}

// TestDetail01: op dispatch is case-insensitive; R*/A* ops are protocol
// errors from CLIENT; L* ops are protocol errors unless LEAF or ROUTER
// (GATEWAY may send A*/RS*/RMSG but never L*).
func TestDetail01(t *testing.T) {
	c := dummyClient()
	if err := ppBb(c, "pInG\r\n"); err != nil {
		t.Fatalf("case-insensitive PING: %v", err)
	}
	for _, op := range []string{
		"RMSG acc foo 5\r\nhello\r\n",
		"LMSG acc foo 5\r\nhello\r\n",
		"A+ acc\r\n",
		"RS+ acc foo\r\n",
		"LS+ acc foo\r\n",
	} {
		c := dummyClient()
		if err := ppBb(c, op); err == nil {
			t.Fatalf("CLIENT must reject %q", op)
		}
	}
	gc := ppBbDummy(GATEWAY)
	if err := ppBb(gc, "LS+ acc foo\r\n"); err == nil {
		t.Fatal("GATEWAY must never accept L* ops")
	}
	rc := dummyRouteClient()
	if err := ppBb(rc, "A+ acc\r\n"); err != nil {
		t.Fatalf("ROUTER may send A* ops: %v", err)
	}
	lc := ppBbDummy(LEAF)
	if err := ppBb(lc, "LS- acc foo\r\n"); err != nil {
		t.Fatalf("LEAF may send L* ops: %v", err)
	}
}

// TestDetail02: PUB/HPUB/SUB/UNSUB/MSG require a space or tab after the
// op token; CONNECT/INFO do not; -ERR requires one.
func TestDetail02(t *testing.T) {
	c := dummyClient()
	if err := ppBb(c, "PUBfoo 5\r\nhello\r\n"); err == nil {
		t.Fatal("PUB without a space must be a protocol error")
	}
	c = dummyClient()
	if err := ppBb(c, "PUB\tfoo\t5\r\nhello\r\n"); err != nil {
		t.Fatalf("tabs are valid separators: %v", err)
	}
	c = dummyClient()
	if err := ppBb(c, "PUB   foo   5\r\nhello\r\n"); err != nil {
		t.Fatalf("extra spaces must be skipped: %v", err)
	}
	c = dummyClient()
	if err := ppBb(c, "SUBfoo 1\r\n"); err == nil {
		t.Fatal("SUB without a space must be a protocol error")
	}
	c = dummyClient()
	if err := ppBb(c, "CONNECT{}\r\n"); err != nil {
		t.Fatalf("CONNECT does not require a space: %v", err)
	}
	c = dummyClient()
	if err := ppBb(c, "INFO{}\r\n"); err != nil {
		t.Fatalf("INFO does not require a space: %v", err)
	}
	c = dummyClient()
	if err := ppBb(c, "-ERR'x'\r\n"); err == nil {
		t.Fatal("-ERR requires a space or tab")
	}
	c = dummyClient()
	if err := ppBb(c, "-ERR 'x'\r\n"); err != nil {
		t.Fatalf("-ERR with a space: %v", err)
	}
}

// TestDetail03: PING/PONG/+OK terminate on '\n' only and absorb other
// bytes before it.
func TestDetail03(t *testing.T) {
	for _, line := range []string{"PING garbage\r\n", "PONG junk junk\n", "+OK whatever\r\n"} {
		c := dummyClient()
		if err := ppBb(c, line); err != nil {
			t.Fatalf("%q should be absorbed, got %v", line, err)
		}
	}
}

// TestDetail04: inside arg states a '\r' mid-arg stays in the assembled
// arg; a dangling trailing '\r' at buffer end is excluded.
func TestDetail04(t *testing.T) {
	c := dummyClient()
	if err := ppBb(c, "PUB f\roo "); err != nil {
		t.Fatal(err)
	}
	if string(c.argBuf) != "f\roo" {
		t.Fatalf("mid-arg CR must stay in the arg, argBuf=%q", c.argBuf)
	}
	c = dummyClient()
	if err := ppBb(c, "PUB foo 5\r"); err != nil {
		t.Fatal(err)
	}
	if string(c.argBuf) != "foo 5" {
		t.Fatalf("dangling trailing CR must be excluded, argBuf=%q", c.argBuf)
	}
}

// TestDetail05: an arg spanning buffers lands in argBuf and parses;
// a '\r'/'\n' split across buffers still terminates cleanly.
func TestDetail05(t *testing.T) {
	c := dummyClient()
	if err := ppBb(c, "PUB foo "); err != nil {
		t.Fatal(err)
	}
	if err := ppBb(c, "5\r\nhello\r\n"); err != nil {
		t.Fatalf("arg spanning buffers: %v", err)
	}
	c = dummyClient()
	if err := ppBb(c, "PUB foo 5\r"); err != nil {
		t.Fatal(err)
	}
	if err := ppBb(c, "\nhello\r\n"); err != nil {
		t.Fatalf("CR/LF split across buffers: %v", err)
	}
}

// TestDetail06: overMaxControlLineLimit compares len(arg) to mcl for
// CLIENT but mcl*16 for route/gateway/leaf; on excess it returns
// ErrMaxControlLine.
func TestDetail06(t *testing.T) {
	c := dummyClient()
	if err := c.overMaxControlLineLimit(make([]byte, 10), 5); !errors.Is(err, ErrMaxControlLine) {
		t.Fatalf("CLIENT over mcl: %v", err)
	}
	c = dummyClient()
	if err := c.overMaxControlLineLimit(make([]byte, 5), 5); err != nil {
		t.Fatalf("CLIENT at mcl should pass: %v", err)
	}
	rc := dummyRouteClient()
	rc.route = &route{}
	rc.subs = map[string]*subscription{}
	if err := rc.overMaxControlLineLimit(make([]byte, 70), 5); err != nil {
		t.Fatalf("ROUTER gets mcl*16 headroom: %v", err)
	}
	rc = dummyRouteClient()
	rc.route = &route{}
	rc.subs = map[string]*subscription{}
	if err := rc.overMaxControlLineLimit(make([]byte, 90), 5); !errors.Is(err, ErrMaxControlLine) {
		t.Fatalf("ROUTER over mcl*16: %v", err)
	}
}

// TestDetail07: pa.size is the payload length without the trailing CRLF;
// MSG_END demands exactly '\r' then '\n' — a payload byte where '\r'
// belongs is a protocol error.
func TestDetail07(t *testing.T) {
	c := dummyClient()
	if err := ppBb(c, "PUB foo 5\r\nhelloX\n"); err == nil {
		t.Fatal("byte after size-exact payload must be CR")
	}
	c = dummyClient()
	if err := ppBb(c, "PUB foo 5\r\nhello\n"); err == nil {
		t.Fatal("LF without CR must fail")
	}
	c = dummyClient()
	if err := ppBb(c, "PUB foo 5\r\nhello\r\n"); err != nil {
		t.Fatalf("exact payload + CRLF must parse: %v", err)
	}
}

// TestDetail08: with msgBuf still nil at a payload boundary the parser
// resumes correctly — a split payload followed by another op completes
// both.
func TestDetail08(t *testing.T) {
	c := dummyClient()
	if err := ppBb(c, "PUB foo 5\r\nhel"); err != nil {
		t.Fatal(err)
	}
	if err := ppBb(c, "lo\r\nPING\r\n"); err != nil {
		t.Fatalf("split payload then PING: %v", err)
	}
	if c.state != OP_START {
		t.Fatalf("parser should be back at OP_START, state=%d", c.state)
	}
}

// TestDetail09: when both the arg and the payload span buffers the saved
// arg is re-parsed from scratch and the message completes.
func TestDetail09(t *testing.T) {
	c := dummyClient()
	for _, chunk := range []string{"PUB f", "oo 5\r\nhe", "llo\r\n"} {
		if err := ppBb(c, chunk); err != nil {
			t.Fatalf("chunk %q: %v", chunk, err)
		}
	}
	if c.state != OP_START {
		t.Fatalf("state=%d", c.state)
	}
}

// TestDetail10: clonePubArg re-dispatches by kind — a LEAF connection
// completes a split LMSG.
func TestDetail10(t *testing.T) {
	lc := ppBbDummy(LEAF)
	if err := ppBb(lc, "LMSG acc foo 5\r\nhe"); err != nil {
		t.Fatal(err)
	}
	if err := ppBb(lc, "llo\r\n"); err != nil {
		t.Fatalf("split LMSG on LEAF: %v", err)
	}
}

// TestDetail11: while the auth gate is set, only CONNECT may proceed for
// CLIENT; any other first op is an authentication error. After CONNECT,
// the gate lifts.
func TestDetail11(t *testing.T) {
	c := dummyClient()
	c.flags.set(expectConnect)
	if err := ppBb(c, "PING\r\n"); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("non-CONNECT under auth gate: %v", err)
	}
	c = dummyClient()
	c.flags.set(expectConnect)
	if err := ppBb(c, "CONNECT{}\r\n"); err != nil {
		t.Fatalf("CONNECT under auth gate: %v", err)
	}
	if err := ppBb(c, "PING\r\n"); err != nil {
		t.Fatalf("gate should lift after CONNECT: %v", err)
	}
}

// TestDetail12: MSG_ARG dispatch is per-kind — a CLIENT receiving MSG
// gets a protocol error at dispatch.
func TestDetail12(t *testing.T) {
	c := dummyClient()
	if err := ppBb(c, "MSG foo 1 5\r\nhello\r\n"); err == nil {
		t.Fatal("CLIENT must reject MSG")
	}
}

// TestDetail13: protoSnippet returns a %q-quoted buf[start:stop] where
// stop=start+max clamped to len(buf)-1, and start>=len yields "".
func TestDetail13(t *testing.T) {
	if s := protoSnippet(0, 3, []byte("hello")); s != fmt.Sprintf("%q", "hel") {
		t.Fatalf("snippet = %q", s)
	}
	// start+max past the end clamps one byte short of the end.
	if s := protoSnippet(2, 50, []byte("hello")); s != fmt.Sprintf("%q", "ll") {
		t.Fatalf("clamped snippet = %q, want %q", s, "\"ll\"")
	}
	if s := protoSnippet(10, 5, []byte("hi")); s != `""` {
		t.Fatalf("out-of-range snippet = %q", s)
	}
}

// TestDetail14: getHeader MIME-parses msgBuf[:pa.hdr] lazily only when
// pa.hdr > 0, skips the status line, caches the result, and leaves the
// cache nil on parse error.
func TestDetail14(t *testing.T) {
	ps := &parseState{}
	ps.pa.hdr = 18
	ps.msgBuf = []byte("NATS/1.0\r\nK: v\r\n\r\n")
	h := ps.getHeader()
	if len(h["K"]) != 1 || h["K"][0] != "v" {
		t.Fatalf("header = %v", h)
	}
	if ps.header == nil {
		t.Fatal("result should be cached")
	}
	ps = &parseState{}
	if h := ps.getHeader(); h != nil {
		t.Fatalf("hdr<=0 should not parse, got %v", h)
	}
	ps = &parseState{}
	ps.pa.hdr = 13
	ps.msgBuf = []byte("NATS/1.0\r\nbad")
	if h := ps.getHeader(); h != nil || ps.header != nil {
		t.Fatalf("bad MIME should yield nil and leave cache nil: %v", h)
	}
}

// TestDetail15: on message completion the parser resets argBuf/msgBuf/
// header, clears pa fields (hdr back to -1), and returns to OP_START.
func TestDetail15(t *testing.T) {
	c := dummyClient()
	if err := ppBb(c, "PUB foo 5\r\nhello\r\n"); err != nil {
		t.Fatal(err)
	}
	if c.state != OP_START {
		t.Fatalf("state=%d", c.state)
	}
	if c.pa.hdr != -1 || c.pa.size != 0 || c.pa.delivered || c.pa.trace != nil {
		t.Fatalf("pa not reset: %+v", c.pa)
	}
	if len(c.argBuf) != 0 || len(c.msgBuf) != 0 || c.header != nil {
		t.Fatal("argBuf/msgBuf/header not reset")
	}
}

// TestDetail16: parse errors carry the connection kind string, the
// numeric state and index, and a bounded %q snippet of the buffer.
func TestDetail16(t *testing.T) {
	c := dummyClient()
	err := ppBb(c, "XYZZY\r\n")
	if err == nil {
		t.Fatal("bad op must error")
	}
	s := err.Error()
	if !strings.Contains(s, "Client") || !strings.Contains(s, "state=") || !strings.Contains(s, "i=") {
		t.Fatalf("error should carry kind/state/index: %q", s)
	}
	if !strings.Contains(s, `"XYZZY`) {
		t.Fatalf("error should quote the proto excerpt: %q", s)
	}
	// Snippet is bounded at PROTO_SNIPPET_SIZE regardless of input size.
	big := strings.Repeat("Z", 5000) + "\r\n"
	c = dummyClient()
	err = ppBb(c, big)
	if err == nil || len(err.Error()) > 300 {
		t.Fatalf("error snippet must be bounded: %q", err)
	}
}
