package server

import (
	"testing"
)

func mtBbHdr(lines string) []byte {
	return []byte("NATS/1.0\r\n" + lines)
}

// TestDetail01: requires the NATS/1.0 status line; key:value lines are
// trimmed of spaces and tabs; empty keys/values are skipped entirely; a
// trailing line without CRLF is not parsed.
func TestDetail01(t *testing.T) {
	if m, _ := genHeaderMapIfTraceHeadersPresent(nil); m != nil {
		t.Fatalf("nil hdr should yield nil map, got %v", m)
	}
	if m, _ := genHeaderMapIfTraceHeadersPresent(
		[]byte("not-a-status-line\r\nMsgkit-Trace-Dest: d\r\n\r\n")); m != nil {
		t.Fatalf("missing status line should yield nil map, got %v", m)
	}
	m, _ := genHeaderMapIfTraceHeadersPresent(
		mtBbHdr("Msgkit-Trace-Dest:\t foo \t\r\n\r\n"))
	if got := m["Msgkit-Trace-Dest"]; len(got) != 1 || got[0] != "foo" {
		t.Fatalf("value should be trimmed of spaces and tabs, got %v", m)
	}
	m, _ = genHeaderMapIfTraceHeadersPresent(
		mtBbHdr("Empty:\r\n: nokey\r\nMsgkit-Trace-Dest: a\r\n\r\n"))
	if _, ok := m["Empty"]; ok {
		t.Fatal("empty value should be skipped entirely")
	}
	if _, ok := m[""]; ok {
		t.Fatal("empty key should be skipped entirely")
	}
	if len(m) != 1 {
		t.Fatalf("only Msgkit-Trace-Dest should remain, got %v", m)
	}
	m, _ = genHeaderMapIfTraceHeadersPresent(mtBbHdr("Msgkit-Trace-Dest: a"))
	if m != nil {
		t.Fatalf("line lacking CRLF should not parse, got %v", m)
	}
}

// TestDetail02: MsgTraceDest matches case-sensitively; traceparent matches
// case-insensitively and preserves the original key case in the output.
func TestDetail02(t *testing.T) {
	if m, _ := genHeaderMapIfTraceHeadersPresent(
		mtBbHdr("msgkit-trace-dest: foo\r\n\r\n")); m != nil {
		t.Fatalf("lowercase dest must not match, got %v", m)
	}
	m, _ := genHeaderMapIfTraceHeadersPresent(
		mtBbHdr("TraceParent: 00-a-b-01\r\n\r\n"))
	if got := m["TraceParent"]; len(got) != 1 {
		t.Fatalf("original key case should be preserved, got %v", m)
	}
}

// TestDetail03: a MsgTraceDest equal to the disabled sentinel returns
// (nil, false).
func TestDetail03(t *testing.T) {
	m, ext := genHeaderMapIfTraceHeadersPresent(
		mtBbHdr("Msgkit-Trace-Dest: " + MsgTraceDestDisabled + "\r\n\r\n"))
	if m != nil || ext {
		t.Fatalf("disabled sentinel should yield (nil,false), got %v %v", m, ext)
	}
}

// TestDetail04: traceparent counts only with exactly 4 dash-separated
// tokens whose 4th is a 2-char hex value with the 0x1 bit set.
func TestDetail04(t *testing.T) {
	sampled := "traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01\r\n\r\n"
	m, ext := genHeaderMapIfTraceHeadersPresent(mtBbHdr(sampled))
	if m == nil || !ext {
		t.Fatalf("sampled traceparent should lift headers, got %v %v", m, ext)
	}
	for _, tp := range []string{
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00", // unsampled
		"not-valid",
		"00-a-b",       // 3 tokens
		"00-a-b-c-d-e", // 5 tokens
		"00-a-b-zz",    // non-hex
		"00-a-b-011",   // 3-char flags field
	} {
		if m, _ := genHeaderMapIfTraceHeadersPresent(
			mtBbHdr("traceparent: " + tp + "\r\n\r\n")); m != nil {
			t.Fatalf("traceparent %q should not lift headers, got %v", tp, m)
		}
	}
	// Any flags value with bit 0x1 set counts.
	if m, _ := genHeaderMapIfTraceHeadersPresent(
		mtBbHdr("traceparent: 00-a-b-03\r\n\r\n")); m == nil {
		t.Fatal("flags 0x03 has the sampled bit and should count")
	}
}

// TestDetail05: the map is nil unless a dest or sampled traceparent is
// found; the bool is true only when traceparent alone lifted it; duplicate
// keys accumulate values in order.
func TestDetail05(t *testing.T) {
	m, ext := genHeaderMapIfTraceHeadersPresent(
		mtBbHdr("traceparent: 00-a-b-01\r\nMsgkit-Trace-Dest: d\r\n\r\n"))
	if ext {
		t.Fatal("dest + traceparent should not report external-only")
	}
	if len(m["traceparent"]) != 1 || len(m["Msgkit-Trace-Dest"]) != 1 {
		t.Fatalf("both keys should be recorded, got %v", m)
	}
	m, ext = genHeaderMapIfTraceHeadersPresent(
		mtBbHdr("Msgkit-Trace-Dest: a\r\nMsgkit-Trace-Dest: b\r\n\r\n"))
	if ext {
		t.Fatal("dest-only should not report external")
	}
	got := m["Msgkit-Trace-Dest"]
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("duplicate keys should accumulate in order, got %v", got)
	}
	// Uninteresting headers alone lift nothing.
	if m, _ := genHeaderMapIfTraceHeadersPresent(
		mtBbHdr("Other: x\r\n\r\n")); m != nil {
		t.Fatalf("uninteresting headers should yield nil, got %v", m)
	}
}

// TestDetail06: getConnName picks the kind-specific remote name, falling
// back to opts.Name when empty or for other kinds.
func TestDetail06(t *testing.T) {
	c := &client{kind: ROUTER, route: &route{remoteName: "rn"}}
	if getConnName(c) != "rn" {
		t.Fatalf("router = %q", getConnName(c))
	}
	c = &client{kind: ROUTER, route: &route{}}
	c.opts.Name = "optname"
	if getConnName(c) != "optname" {
		t.Fatalf("router empty remote = %q", getConnName(c))
	}
	c = &client{kind: GATEWAY, gw: &gateway{remoteName: "gn"}}
	if getConnName(c) != "gn" {
		t.Fatalf("gateway = %q", getConnName(c))
	}
	c = &client{kind: LEAF, leaf: &leaf{remoteServer: "ln"}}
	if getConnName(c) != "ln" {
		t.Fatalf("leaf = %q", getConnName(c))
	}
	for _, k := range []int{CLIENT, SYSTEM} {
		c = &client{kind: k}
		c.opts.Name = "optname"
		if getConnName(c) != "optname" {
			t.Fatalf("kind %d = %q", k, getConnName(c))
		}
	}
}

// TestDetail07: getCompressionType — empty → noCompression; case-insensitive
// substring match for snappy/s2 → snappy, gzip → gzip; else unsupported.
func TestDetail07(t *testing.T) {
	if getCompressionType("") != noCompression {
		t.Fatal("empty should be noCompression")
	}
	for _, s := range []string{"snappy", "SNAPPY", "s2", "S2"} {
		if getCompressionType(s) != snappyCompression {
			t.Fatalf("%q should be snappy", s)
		}
	}
	for _, s := range []string{"gzip", "GZIP", "xgzipx"} {
		if getCompressionType(s) != gzipCompression {
			t.Fatalf("%q should be gzip", s)
		}
	}
	for _, s := range []string{"br", "xsnap", "deflate"} {
		if getCompressionType(s) != unsupportedCompression {
			t.Fatalf("%q should be unsupported", s)
		}
	}
}

// TestDetail08: sample treats out-of-range [1..99] as 100%; in-range values
// sample at roughly their percentage.
func TestDetail08(t *testing.T) {
	for _, n := range []int{-5, 0, 100, 200} {
		for i := 0; i < 50; i++ {
			if !sample(n) {
				t.Fatalf("sample(%d) should always be true", n)
			}
		}
	}
	hits := 0
	const trials = 20000
	for i := 0; i < trials; i++ {
		if sample(50) {
			hits++
		}
	}
	if hits < trials*35/100 || hits > trials*65/100 {
		t.Fatalf("sample(50) hit %d/%d, far outside 50%%", hits, trials)
	}
}

// TestDetail09: msgTraceSupport is always true for CLIENT; other kinds
// require opts.Protocol >= MsgTraceProto.
func TestDetail09(t *testing.T) {
	c := &client{kind: CLIENT}
	if !c.msgTraceSupport() {
		t.Fatal("CLIENT should always support tracing")
	}
	for _, k := range []int{ROUTER, GATEWAY, LEAF} {
		c = &client{kind: k}
		if c.msgTraceSupport() {
			t.Fatalf("kind %d with proto 0 should not support tracing", k)
		}
		c.opts.Protocol = MsgTraceProto
		if !c.msgTraceSupport() {
			t.Fatalf("kind %d with MsgTraceProto should support tracing", k)
		}
	}
}
