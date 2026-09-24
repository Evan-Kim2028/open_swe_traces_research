package server

import (
	"errors"
	"strings"
	"testing"

	"github.com/nats-io/jwt/v2"
)

func lmBbClient() *client {
	c := dummyClient()
	c.kind = LEAF
	c.leaf = &leaf{}
	return c
}

func lmBbQueues(c *client) []string {
	var out []string
	for _, q := range c.pa.queues {
		out = append(out, string(q))
	}
	return out
}

// TestDetail01: inline arg tokenization splits on space, tab, CR, LF; c.pa.arg
// is the raw arg and is populated even when the parse fails.
func TestDetail01(t *testing.T) {
	c := lmBbClient()
	if err := c.processLeafMsgArgs([]byte("foo\t_INBOX.x 12")); err != nil {
		t.Fatalf("mixed-whitespace parse: %v", err)
	}
	if string(c.pa.subject) != "foo" || string(c.pa.reply) != "_INBOX.x" || c.pa.size != 12 {
		t.Fatalf("mixed whitespace: subj=%q reply=%q size=%d", c.pa.subject, c.pa.reply, c.pa.size)
	}

	c = lmBbClient()
	err := c.processLeafMsgArgs([]byte("foo"))
	if err == nil {
		t.Fatal("expected arity error")
	}
	if string(c.pa.arg) != "foo" {
		t.Fatalf("pa.arg should hold the raw arg on error, got %q", c.pa.arg)
	}
}

// TestDetail02: processLeafMsgArgs layout is `subject [reply] size`; with 4+
// args args[1] is a single-byte reply indicator ('+' reply follows, '|' none),
// and queues are the tokens between reply position and size.
func TestDetail02(t *testing.T) {
	c := lmBbClient()
	if err := c.processLeafMsgArgs([]byte("foo 12")); err != nil {
		t.Fatal(err)
	}
	if string(c.pa.subject) != "foo" || len(c.pa.reply) != 0 || c.pa.size != 12 {
		t.Fatalf("2-arg: subj=%q reply=%q size=%d", c.pa.subject, c.pa.reply, c.pa.size)
	}

	c = lmBbClient()
	if err := c.processLeafMsgArgs([]byte("foo _INBOX.x 12")); err != nil {
		t.Fatal(err)
	}
	if string(c.pa.reply) != "_INBOX.x" {
		t.Fatalf("3-arg reply = %q", c.pa.reply)
	}

	// 4+ args: indicator required.
	c = lmBbClient()
	if err := c.processLeafMsgArgs([]byte("foo + _INBOX.x q1 q2 12")); err != nil {
		t.Fatal(err)
	}
	if string(c.pa.reply) != "_INBOX.x" {
		t.Fatalf("indicator reply = %q", c.pa.reply)
	}
	if qs := lmBbQueues(c); len(qs) != 2 || qs[0] != "q1" || qs[1] != "q2" {
		t.Fatalf("queues = %v", qs)
	}

	c = lmBbClient()
	if err := c.processLeafMsgArgs([]byte("foo | q1 q2 12")); err != nil {
		t.Fatal(err)
	}
	if len(c.pa.reply) != 0 {
		t.Fatalf("'|' indicator must yield no reply, got %q", c.pa.reply)
	}
	if qs := lmBbQueues(c); len(qs) != 2 || qs[0] != "q1" || qs[1] != "q2" {
		t.Fatalf("queues = %v", qs)
	}

	// Bad indicator: wrong byte or multi-byte token; and a 4-arg line with
	// no indicator at all.
	for _, arg := range []string{"foo x _INBOX.x 12", "foo xx _INBOX.x 12", "foo _INBOX.x q1 12"} {
		c = lmBbClient()
		err := c.processLeafMsgArgs([]byte(arg))
		if err == nil || !strings.Contains(err.Error(), "Bad or Missing Reply Indicator") {
			t.Fatalf("%q: want reply-indicator error, got %v", arg, err)
		}
	}
}

// TestDetail03: processLeafHeaderMsgArgs adds hdr/hdb — total size is the last
// token and header size the second-to-last; queues shift accordingly.
func TestDetail03(t *testing.T) {
	c := lmBbClient()
	if err := c.processLeafHeaderMsgArgs([]byte("foo 4 17")); err != nil {
		t.Fatal(err)
	}
	if string(c.pa.subject) != "foo" || c.pa.hdr != 4 || string(c.pa.hdb) != "4" || c.pa.size != 17 || string(c.pa.szb) != "17" {
		t.Fatalf("3-arg: subj=%q hdr=%d hdb=%q size=%d szb=%q", c.pa.subject, c.pa.hdr, c.pa.hdb, c.pa.size, c.pa.szb)
	}

	c = lmBbClient()
	if err := c.processLeafHeaderMsgArgs([]byte("foo + _INBOX.x q1 q2 4 17")); err != nil {
		t.Fatal(err)
	}
	if string(c.pa.reply) != "_INBOX.x" || c.pa.hdr != 4 || c.pa.size != 17 {
		t.Fatalf("indicator form: reply=%q hdr=%d size=%d", c.pa.reply, c.pa.hdr, c.pa.size)
	}
	if qs := lmBbQueues(c); len(qs) != 2 || qs[0] != "q1" || qs[1] != "q2" {
		t.Fatalf("header queues = %v", qs)
	}

	c = lmBbClient()
	if err := c.processLeafHeaderMsgArgs([]byte("foo | q1 q2 4 17")); err != nil {
		t.Fatal(err)
	}
	if len(c.pa.reply) != 0 || c.pa.hdr != 4 || c.pa.size != 17 {
		t.Fatalf("'|' header form: reply=%q hdr=%d size=%d", c.pa.reply, c.pa.hdr, c.pa.size)
	}
	if qs := lmBbQueues(c); len(qs) != 2 || qs[0] != "q1" || qs[1] != "q2" {
		t.Fatalf("header '|' queues = %v", qs)
	}

	c = lmBbClient()
	if err := c.processLeafHeaderMsgArgs([]byte("foo x _INBOX.x 4 17")); err == nil ||
		!strings.Contains(err.Error(), "Bad or Missing Reply Indicator") {
		t.Fatalf("bad indicator: %v", err)
	}
	// Header variant needs >= 3 args.
	c = lmBbClient()
	if err := c.processLeafHeaderMsgArgs([]byte("foo 17")); err == nil {
		t.Fatal("2-arg header parse should fail")
	}
}

// TestDetail04: size and header size come via parseSize; a negative result
// errors, and the subject is only assigned after validation passes.
func TestDetail04(t *testing.T) {
	for _, arg := range []string{"foo abc", "foo -5"} {
		c := lmBbClient()
		err := c.processLeafMsgArgs([]byte(arg))
		if err == nil || !strings.Contains(err.Error(), "Bad or Missing Size") {
			t.Fatalf("%q: want size error, got %v", arg, err)
		}
		if len(c.pa.subject) != 0 {
			t.Fatalf("%q: subject must not be assigned on failure, got %q", arg, c.pa.subject)
		}
		if len(c.pa.szb) == 0 {
			t.Fatalf("%q: szb should carry the raw size token", arg)
		}
	}
	c := lmBbClient()
	if err := c.processLeafHeaderMsgArgs([]byte("foo -1 17")); err == nil ||
		!strings.Contains(err.Error(), "Bad or Missing Header Size") {
		t.Fatalf("bad header size: %v", err)
	}
	if len(c.pa.subject) != 0 {
		t.Fatalf("subject must not be assigned on header-size failure, got %q", c.pa.subject)
	}
	c = lmBbClient()
	if err := c.processLeafHeaderMsgArgs([]byte("foo 4 abc")); err == nil ||
		!strings.Contains(err.Error(), "Bad or Missing Size") {
		t.Fatalf("bad size in header parse: %v", err)
	}
	if string(c.pa.hdb) != "4" {
		t.Fatalf("hdb should carry the raw header-size token, got %q", c.pa.hdb)
	}
}

// TestDetail05: a parsed size above the client's mpay returns the ErrMaxPayload
// sentinel (maxPayloadViolation path), not a parse error.
func TestDetail05(t *testing.T) {
	c := lmBbClient()
	c.mpay = 16
	if err := c.processLeafMsgArgs([]byte("foo 17")); !errors.Is(err, ErrMaxPayload) {
		t.Fatalf("oversize LMSG: %v", err)
	}
	c = lmBbClient()
	c.mpay = 16
	if err := c.processLeafHeaderMsgArgs([]byte("foo 4 17")); !errors.Is(err, ErrMaxPayload) {
		t.Fatalf("oversize LHMSG: %v", err)
	}
	// Unlimited payload accepts any size.
	c = lmBbClient()
	c.mpay = jwt.NoLimit
	if err := c.processLeafMsgArgs([]byte("foo 999999")); err != nil {
		t.Fatalf("NoLimit should accept any size: %v", err)
	}
}

// TestDetail06: keyFromSub emits "subject" or "subject queue".
func TestDetail06(t *testing.T) {
	if k := keyFromSub(&subscription{subject: []byte("foo")}); k != "foo" {
		t.Fatalf("keyFromSub = %q", k)
	}
	if k := keyFromSub(&subscription{subject: []byte("foo"), queue: []byte("bar")}); k != "foo bar" {
		t.Fatalf("keyFromSub queue = %q", k)
	}
	if k := keyFromSub(&subscription{subject: []byte("foo"), leaf: true, origin: []byte("o")}); k != "foo" {
		t.Fatalf("keyFromSub ignores leaf/origin = %q", k)
	}
}

// TestDetail07: keyFromSubWithOrigin prefixes a one-byte kind — 'L' when an
// origin is set, 'N' for leaf subs without origin, 'R' otherwise — then
// subject, optional queue, optional origin.
func TestDetail07(t *testing.T) {
	cases := []struct {
		sub  *subscription
		want string
	}{
		{&subscription{subject: []byte("foo")}, "R foo"},
		{&subscription{subject: []byte("foo"), queue: []byte("bar")}, "R foo bar"},
		{&subscription{subject: []byte("foo"), leaf: true}, "N foo"},
		{&subscription{subject: []byte("foo"), queue: []byte("bar"), leaf: true}, "N foo bar"},
		{&subscription{subject: []byte("foo"), leaf: true, origin: []byte("baz")}, "L foo baz"},
		{&subscription{subject: []byte("foo"), queue: []byte("bar"), leaf: true, origin: []byte("baz")}, "L foo bar baz"},
		{&subscription{subject: []byte("foo"), origin: []byte("baz")}, "L foo baz"},
	}
	for _, tc := range cases {
		if k := keyFromSubWithOrigin(tc.sub); k != tc.want {
			t.Fatalf("keyFromSubWithOrigin(%+v) = %q, want %q", tc.sub.subject, k, tc.want)
		}
	}
	// All kinds are distinct for the same subject+queue.
	ks := map[string]bool{}
	for _, s := range []*subscription{
		{subject: []byte("foo"), queue: []byte("bar")},
		{subject: []byte("foo"), queue: []byte("bar"), leaf: true},
		{subject: []byte("foo"), queue: []byte("bar"), leaf: true, origin: []byte("o")},
	} {
		k := keyFromSubWithOrigin(s)
		if ks[k] {
			t.Fatalf("duplicate key %q", k)
		}
		ks[k] = true
	}
}
