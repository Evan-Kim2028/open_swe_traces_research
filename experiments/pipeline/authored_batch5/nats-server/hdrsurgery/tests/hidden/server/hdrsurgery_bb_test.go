package server

import (
	"bytes"
	"testing"
)

// TestDetail01: getHeaderKeyIndex only accepts a key preceded by a full CRLF
// and immediately followed by ':'; rejected matches retry past the key.
func TestDetail01(t *testing.T) {
	hdr := []byte("NATS/1.0\r\nKey: v1\r\nOther: x\r\n\r\n")
	if i := getHeaderKeyIndex("Key", hdr); i != 10 {
		t.Fatalf("Key index = %d, want 10", i)
	}
	if i := getHeaderKeyIndex("Other", hdr); i != 19 {
		t.Fatalf("Other index = %d, want 19", i)
	}
	if i := getHeaderKeyIndex("Missing", hdr); i != -1 {
		t.Fatalf("missing key = %d, want -1", i)
	}

	// A longer key that merely starts with the searched key must not shadow.
	shadow := []byte("NATS/1.0\r\nKeyFoo: a\r\nKey: b\r\n\r\n")
	i := getHeaderKeyIndex("Key", shadow)
	if i < 0 || !bytes.HasPrefix(shadow[i:], []byte("Key: b")) {
		t.Fatalf("shadowed key resolved to %d (%q), want the real 'Key:' line", i, shadow[i:])
	}

	// A key appearing inside a value (not at a line start) is not a match.
	emb := []byte("NATS/1.0\r\nV: has Key: inside\r\n\r\n")
	if i := getHeaderKeyIndex("Key", emb); i != -1 {
		t.Fatalf("mid-value key occurrence must be rejected, got %d", i)
	}
}

// TestDetail02: sliceHeader borrows into hdr with capacity limited to the
// value; getHeader returns an exact-size copy.
func TestDetail02(t *testing.T) {
	hdr := []byte("NATS/1.0\r\nKey: v1\r\nOther: x\r\n\r\n")

	s := sliceHeader("Key", hdr)
	if string(s) != "v1" {
		t.Fatalf("sliceHeader = %q, want %q", s, "v1")
	}
	if cap(s) != 2 {
		t.Fatalf("sliceHeader cap = %d, want 2 (no copy, capped at value end)", cap(s))
	}
	// Borrowed: mutating hdr reflects in the slice.
	hdr[15] = 'X'
	if string(s) != "X1" {
		t.Fatalf("sliceHeader should borrow from hdr, got %q after mutation", s)
	}

	g := getHeader("Key", []byte("NATS/1.0\r\nKey: v1\r\n\r\n"))
	if string(g) != "v1" || cap(g) != 2 {
		t.Fatalf("getHeader = %q cap %d, want exact-size copy", g, cap(g))
	}

	// Leading spaces after ':' are skipped; value ends at first CRLF.
	sp := sliceHeader("K", []byte("NATS/1.0\r\nK:   spaced  \r\n\r\n"))
	if string(sp) != "spaced  " {
		t.Fatalf("leading-space skip = %q", sp)
	}
	if sliceHeader("Missing", hdr) != nil {
		t.Fatal("missing key must return nil")
	}
	if getHeader("Missing", hdr) != nil {
		t.Fatal("getHeader missing key must return nil")
	}
}

// TestDetail03: removeHeaderIfPresent removes ALL exact-key lines; when only
// the empty header line remains the result is nil.
func TestDetail03(t *testing.T) {
	dup := []byte("NATS/1.0\r\nK: 1\r\nX: 2\r\nK: 3\r\n\r\n")
	got := removeHeaderIfPresent(dup, "K")
	if string(got) != "NATS/1.0\r\nX: 2\r\n\r\n" {
		t.Fatalf("duplicate removal = %q", got)
	}

	only := []byte("NATS/1.0\r\nK: 1\r\n\r\n")
	if r := removeHeaderIfPresent(only, "K"); r != nil {
		t.Fatalf("removing the only header line must return nil, got %q", r)
	}

	// Longer keys that start with the key are not removed.
	prefixed := []byte("NATS/1.0\r\nKLong: 1\r\nK: 2\r\n\r\n")
	if got := removeHeaderIfPresent(prefixed, "K"); string(got) != "NATS/1.0\r\nKLong: 1\r\n\r\n" {
		t.Fatalf("exact-key removal = %q", got)
	}

	miss := removeHeaderIfPresent(dup, "ZZZ")
	if !bytes.Equal(miss, dup) {
		t.Fatalf("absent key should leave header unchanged, got %q", miss)
	}
}

// TestDetail04: removeHeaderIfPrefixPresent removes lines whose key begins
// with the prefix; the occurrence must start a line.
func TestDetail04(t *testing.T) {
	in := []byte("NATS/1.0\r\nNats-A: 1\r\nKeep: 2\r\nNats-B: 3\r\n\r\n")
	if got := removeHeaderIfPrefixPresent(in, "Nats-"); string(got) != "NATS/1.0\r\nKeep: 2\r\n\r\n" {
		t.Fatalf("prefix removal = %q", got)
	}
	// A prefix occurring inside a value is skipped; scanning continues.
	mid := []byte("NATS/1.0\r\nV: hasNats-here\r\nNats-X: 9\r\n\r\n")
	if got := removeHeaderIfPrefixPresent(mid, "Nats-"); string(got) != "NATS/1.0\r\nV: hasNats-here\r\n\r\n" {
		t.Fatalf("non-line-start occurrence must be skipped, got %q", got)
	}
	if got := removeHeaderIfPrefixPresent(in, "Nope-"); !bytes.Equal(got, in) {
		t.Fatalf("absent prefix should leave header unchanged, got %q", got)
	}
}

// TestDetail05: removeHeaderStatusIfPresent only strips a NATS/1.0 status
// line; a non-NATS/1.0 header is returned unchanged. Shape-only assertions.
func TestDetail05(t *testing.T) {
	withDesc := []byte("NATS/1.0 200 OK\r\nK: 1\r\n\r\n")
	got := removeHeaderStatusIfPresent(withDesc)
	if string(got) != "NATS/1.0\r\nK: 1\r\n\r\n" {
		t.Fatalf("status strip = %q", got)
	}
	// Removing a status line that leaves only the empty header line → nil.
	if r := removeHeaderStatusIfPresent([]byte("NATS/1.0 200\r\n\r\n")); r != nil {
		t.Fatalf("status-only header should reduce to nil, got %q", r)
	}
	nonNATS := []byte("HTTP/1.1 200\r\nK: 1\r\n\r\n")
	if r := removeHeaderStatusIfPresent(nonNATS); !bytes.Equal(r, nonNATS) {
		t.Fatalf("non-NATS/1.0 header must be unchanged, got %q", r)
	}
	// A bare status line with no extra CR after the marker is unchanged.
	bare := []byte("NATS/1.0\r\n\r\n")
	if r := removeHeaderStatusIfPresent(bare); !bytes.Equal(r, bare) {
		t.Fatalf("empty header must be unchanged, got %q", r)
	}
}

// TestDetail06: isServiceReply is the "_R_." prefix; isJSAckSubject is the
// "$JS.ACK." prefix with at least one more byte.
func TestDetail06(t *testing.T) {
	if !isServiceReply([]byte("_R_.x")) || !isServiceReply([]byte("_R_.")) {
		t.Fatal("service reply prefix not detected")
	}
	if isServiceReply([]byte("R_.x")) || isServiceReply([]byte("_R_")) {
		t.Fatal("non-service reply misclassified")
	}
	if !isJSAckSubject([]byte("$JS.ACK.x")) {
		t.Fatal("JS ack subject not detected")
	}
	if isJSAckSubject([]byte("$JS.ACK.")) || isJSAckSubject([]byte("$JS.ACK")) {
		t.Fatal("prefix-length-only subject must not be a JS ack")
	}
}

// TestDetail07: jsAckDeliverIdx returns the first '@' after at least eight
// '.' characters; non-JS-ack replies return -1.
func TestDetail07(t *testing.T) {
	ack := []byte("$JS.ACK.dom.acc.stream.cons.1.2.3.4@5")
	if i := jsAckDeliverIdx(ack); i != 35 {
		t.Fatalf("deliver idx = %d, want 35", i)
	}
	// '@' inside an early token does not count.
	atTok := []byte("$JS.ACK.st@am.acc.stream.cons.1.2.3.4@5")
	if i := jsAckDeliverIdx(atTok); i != 37 {
		t.Fatalf("embedded-@ reply idx = %d, want 37", i)
	}
	// Fewer than eight dots before '@' → -1.
	if i := jsAckDeliverIdx([]byte("$JS.ACK.a.b.c.d.e@9")); i != -1 {
		t.Fatalf("insufficient dots should give -1, got %d", i)
	}
	// No '@' after the dots → -1.
	if i := jsAckDeliverIdx([]byte("$JS.ACK.a.b.c.d.e.f.g")); i != -1 {
		t.Fatalf("no deliver marker should give -1, got %d", i)
	}
	// Non-JS-ack reply → -1.
	if i := jsAckDeliverIdx([]byte("a.b.c.d.e.f.g.h@x")); i != -1 {
		t.Fatalf("non-ack reply should give -1, got %d", i)
	}
}

// TestDetail08: isReservedReply covers service replies, JS ack subjects, and
// gateway routed reply prefixes.
func TestDetail08(t *testing.T) {
	for _, r := range []string{"_R_.abc", "$JS.ACK.x.y", "_GR_.gw.reply", "$GR.gw.reply"} {
		if !isReservedReply([]byte(r)) {
			t.Fatalf("%q should be a reserved reply", r)
		}
	}
	for _, r := range []string{"normal.reply", "_R", "$JS.AC", ""} {
		if isReservedReply([]byte(r)) {
			t.Fatalf("%q should not be a reserved reply", r)
		}
	}
}

// TestDetail09: splitSubjectQueue splits on whitespace; 1 field → subject,
// 2 → subject+queue, 0 or >2 → error; both tokens must be valid subjects.
func TestDetail09(t *testing.T) {
	s, q, err := splitSubjectQueue("foo")
	if err != nil || string(s) != "foo" || len(q) != 0 {
		t.Fatalf("single field: %q %q %v", s, q, err)
	}
	s, q, err = splitSubjectQueue("  foo   bar  ")
	if err != nil || string(s) != "foo" || string(q) != "bar" {
		t.Fatalf("two fields: %q %q %v", s, q, err)
	}
	for _, in := range []string{"", "a b c", "a..b", "a q..x", "a..b q"} {
		if _, _, err := splitSubjectQueue(in); err == nil {
			t.Fatalf("%q should error", in)
		}
	}
	// Tab is whitespace too.
	s, q, err = splitSubjectQueue("foo\tbar")
	if err != nil || string(s) != "foo" || string(q) != "bar" {
		t.Fatalf("tab split: %q %q %v", s, q, err)
	}
}

// TestDetail10: splitArg tokenizes on space, tab, CR, LF with runs collapsed
// and no empty tokens.
func TestDetail10(t *testing.T) {
	if got := splitArg([]byte("")); len(got) != 0 {
		t.Fatalf("empty arg = %q", got)
	}
	got := splitArg([]byte("a  b\tc\rd\ne"))
	want := [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d"), []byte("e")}
	if len(got) != len(want) {
		t.Fatalf("splitArg len = %d, want %d (%q)", len(got), len(want), got)
	}
	for i := range want {
		if !bytes.Equal(got[i], want[i]) {
			t.Fatalf("token %d = %q, want %q", i, got[i], want[i])
		}
	}
	// More than MAX_MSG_ARGS args still all returned.
	got = splitArg([]byte("a b c d e f g"))
	if len(got) != 7 {
		t.Fatalf("7 args = %d tokens", len(got))
	}
	if got := splitArg([]byte("   ")); len(got) != 0 {
		t.Fatalf("all-whitespace arg = %q", got)
	}
}
