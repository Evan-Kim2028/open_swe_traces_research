// Hidden black-box property suite for the proxyproto unit.
// Drives only the API in api.md: detect/read/parse entry points, the
// proxyProtoAddr/proxyConn types and the documented error values, fed with
// hand-crafted wire bytes through net.Pipe.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package server

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const bbPpHiddenSeed = 20260919

func bbPpSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbPpHiddenSeed
}

func bbPpRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbPpSeed()))
}

// bbPpFeed writes data to one end of a net.Pipe in the background and
// returns the reading end.
func bbPpFeed(t *testing.T, data []byte) net.Conn {
	t.Helper()
	w, r := net.Pipe()
	go func() {
		w.Write(data)
		w.Close()
	}()
	return r
}

// bbPpReadAll drains a conn (for reconstructing post-header bytes).
func bbPpReadAll(c net.Conn) []byte {
	c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var out []byte
	buf := make([]byte, 4096)
	for {
		n, err := c.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			return out
		}
	}
}

// bbPpRecConn records SetReadDeadline calls.
type bbPpRecConn struct {
	net.Conn
	mu        sync.Mutex
	deadlines []time.Time
}

func (c *bbPpRecConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	c.deadlines = append(c.deadlines, t)
	c.mu.Unlock()
	return c.Conn.SetReadDeadline(t)
}

func (c *bbPpRecConn) Recorded() []time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Time(nil), c.deadlines...)
}

// bbPpTimeoutErr implements net.Error with Timeout()==true.
type bbPpTimeoutErr struct{}

func (bbPpTimeoutErr) Error() string   { return "i/o timeout (test)" }
func (bbPpTimeoutErr) Timeout() bool   { return true }
func (bbPpTimeoutErr) Temporary() bool { return true }

type bbPpTimeoutConn struct{ net.Conn }

func (c *bbPpTimeoutConn) Read(b []byte) (int, error) { return 0, bbPpTimeoutErr{} }

// bbPpV2Header builds a v2 header: sig + ver/cmd + fam/proto + addrlen + addrs.
func bbPpV2Header(vercmd, famproto byte, addrs []byte) []byte {
	h := append([]byte(proxyProtoV2Sig), vercmd, famproto, 0, 0)
	binary.BigEndian.PutUint16(h[len(h)-2:], uint16(len(addrs)))
	return append(h, addrs...)
}

func bbPpIPv4Addrs(src net.IP, dst net.IP, sport, dport uint16, extra int) []byte {
	b := make([]byte, 12+extra)
	copy(b[0:4], src.To4())
	copy(b[4:8], dst.To4())
	binary.BigEndian.PutUint16(b[8:10], sport)
	binary.BigEndian.PutUint16(b[10:12], dport)
	return b
}

func bbPpIPv6Addrs(src, dst net.IP, sport, dport uint16, extra int) []byte {
	b := make([]byte, 36+extra)
	copy(b[0:16], src.To16())
	copy(b[16:32], dst.To16())
	binary.BigEndian.PutUint16(b[32:34], sport)
	binary.BigEndian.PutUint16(b[34:36], dport)
	return b
}

// Detail 1: detection reads exactly 6 bytes — "PROXY " -> v1, v2 sig
// prefix -> v2, anything else -> unrecognized error + consumed bytes for
// replay.
func TestDetail01_DetectSixBytes(t *testing.T) {
	// v1.
	conn := bbPpFeed(t, []byte("PROXY TCP4 1.2.3.4 5.6.7.8 1 2\r\n"))
	v, hdr, err := detectProxyProtoVersion(conn)
	if err != nil || v != 1 {
		t.Fatalf("v1 detect=(%d,%v) err=%v", v, hdr, err)
	}
	if string(hdr) != "PROXY " {
		t.Fatalf("v1 header=%q want \"PROXY \"", hdr)
	}
	// v2.
	full := bbPpV2Header(proxyProtoV2Ver|proxyProtoCmdProxy,
		proxyProtoFamilyInet|proxyProtoProtoStream,
		bbPpIPv4Addrs(net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8), 1, 2, 0))
	conn = bbPpFeed(t, full)
	v, hdr, err = detectProxyProtoVersion(conn)
	if err != nil || v != 2 {
		t.Fatalf("v2 detect=(%d,%v) err=%v", v, hdr, err)
	}
	if string(hdr) != proxyProtoV2Sig[:6] {
		t.Fatalf("v2 header=%x want sig prefix", hdr)
	}
	// Unrecognized: error + 6 consumed bytes returned for replay.
	rng := bbPpRng(t)
	garbage := make([]byte, 6)
	for i := range garbage {
		garbage[i] = byte(33 + rng.Intn(90)) // printable, non-"PROXY "
	}
	if string(garbage) == "PROXY " {
		garbage[0] = 'X'
	}
	conn = bbPpFeed(t, append(garbage, []byte("REST")...))
	v, hdr, err = detectProxyProtoVersion(conn)
	if !errors.Is(err, errProxyProtoUnrecognized) {
		t.Fatalf("garbage detect err=%v want errProxyProtoUnrecognized", err)
	}
	if string(hdr) != string(garbage) {
		t.Fatalf("replay bytes=%q want %q", hdr, garbage)
	}
	// Exactly 6 consumed: the remaining bytes are still in the stream.
	rest := bbPpReadAll(conn)
	if string(rest) != "REST" {
		t.Fatalf("stream after detect=%q want REST (only 6 consumed)", rest)
	}
}

// Detail 2: v1 line bound — total 107 bytes incl. prefix (101 after it);
// exceeding without CRLF -> invalid error.
func TestDetail02_V1LineBound(t *testing.T) {
	// A maximal valid line well under the bound.
	line := "TCP4 255.255.255.255 255.255.255.255 65535 65535\r\n"
	if len("PROXY "+line) > proxyProtoV1MaxLineLen {
		t.Fatalf("test line itself too long: %d", len("PROXY "+line))
	}
	conn := bbPpFeed(t, []byte(line))
	addr, extra, err := readProxyProtoV1Header(conn)
	if err != nil || addr == nil {
		t.Fatalf("valid line: addr=%v extra=%q err=%v", addr, extra, err)
	}
	// A line of exactly proxyProtoV1MaxLineLen-6 post-prefix bytes (i.e.
	// total 107 incl. the consumed "PROXY " prefix and CRLF) reaches the
	// parser — invalid field content yields a parse-level error, not a
	// length rejection. Valid fields can't fill 101 bytes, so the parse
	// error itself proves the bound accepted the length.
	pad := proxyProtoV1MaxLineLen - len("PROXY ") - len("TCP4 " + " 5.6.7.8 1111 2222\r\n")
	if pad < 1 {
		t.Fatalf("pad=%d", pad)
	}
	atBound := "TCP4 " + strings.Repeat("1", pad) + " 5.6.7.8 1111 2222\r\n"
	conn = bbPpFeed(t, []byte(atBound))
	addr, _, err = readProxyProtoV1Header(conn)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("line at exactly %d bytes: err=%v want parse-level error", len(atBound)+6, err)
	}
	// Over the bound without CRLF -> invalid.
	over := "TCP4 " + strings.Repeat("1", proxyProtoV1MaxLineLen) + "\r\n"
	conn = bbPpFeed(t, []byte(over))
	_, _, err = readProxyProtoV1Header(conn)
	if !errors.Is(err, errProxyProtoInvalid) {
		t.Fatalf("over-bound line err=%v want invalid", err)
	}
}

// Detail 3: v1 returns bytes read past the CRLF for replay.
func TestDetail03_V1CoalescedBytes(t *testing.T) {
	payload := "PING\r\nCONNECT {}\r\n"
	conn := bbPpFeed(t, []byte("TCP4 1.2.3.4 5.6.7.8 1111 2222\r\n"+payload))
	addr, extra, err := readProxyProtoV1Header(conn)
	if err != nil || addr == nil {
		t.Fatalf("addr=%v err=%v", addr, err)
	}
	// extra + remaining stream must reconstruct the payload.
	rest := bbPpReadAll(conn)
	got := string(append(append([]byte(nil), extra...), rest...))
	if got != payload {
		t.Fatalf("replayed payload=%q want %q", got, payload)
	}
}

// Detail 4: v1 "UNKNOWN" -> nil address, nil error (health check).
func TestDetail04_V1Unknown(t *testing.T) {
	conn := bbPpFeed(t, []byte("UNKNOWN\r\n"))
	addr, _, err := readProxyProtoV1Header(conn)
	if err != nil {
		t.Fatalf("UNKNOWN err=%v", err)
	}
	if addr != nil {
		t.Fatalf("UNKNOWN addr=%v want nil", addr)
	}
	// Through the auto-detect entry as well.
	conn = bbPpFeed(t, []byte("PROXY UNKNOWN\r\n"))
	addr, _, err = readProxyProtoHeader(conn)
	if err != nil || addr != nil {
		t.Fatalf("auto UNKNOWN addr=%v err=%v", addr, err)
	}
}

// Detail 5: v1 non-UNKNOWN must have EXACTLY 5 fields.
func TestDetail05_V1FieldCount(t *testing.T) {
	for _, line := range []string{
		"TCP4 1.2.3.4 5.6.7.8 1111\r\n",           // 4 fields
		"TCP4 1.2.3.4 5.6.7.8 1111 2222 3333\r\n", // 6
		"TCP4\r\n",                               // 1
		"TCP4 1.2.3.4\r\n",                       // 2
		"\r\n",                                   // 0
	} {
		conn := bbPpFeed(t, []byte(line))
		_, _, err := readProxyProtoV1Header(conn)
		if !errors.Is(err, errProxyProtoInvalid) {
			t.Fatalf("line %q err=%v want invalid", line, err)
		}
	}
	conn := bbPpFeed(t, []byte("TCP4 1.2.3.4 5.6.7.8 1111 2222\r\n"))
	addr, _, err := readProxyProtoV1Header(conn)
	if err != nil || addr == nil {
		t.Fatalf("5-field line: addr=%v err=%v", addr, err)
	}
}

// Detail 6: v1 family validation is colon-based — TCP4 rejects any ':'
// address; TCP6 requires ':' in both; ::ffff: mapped literals are legal
// TCP6.
func TestDetail06_V1FamilyColonRule(t *testing.T) {
	cases := []struct {
		line string
		ok   bool
	}{
		{"TCP4 1.2.3.4 5.6.7.8 1 2\r\n", true},
		{"TCP4 ::1 ::1 1 2\r\n", false},              // ':' src
		{"TCP4 1.2.3.4 ::1 1 2\r\n", false},          // ':' dst
		{"TCP6 ::1 ::2 1 2\r\n", true},
		{"TCP6 ::ffff:1.2.3.4 ::ffff:5.6.7.8 1 2\r\n", true}, // mapped ok
		{"TCP6 1.2.3.4 5.6.7.8 1 2\r\n", false},      // no ':'
		{"TCP6 ::1 5.6.7.8 1 2\r\n", false},          // dst lacks ':'
		{"TCP6 1.2.3.4 ::2 1 2\r\n", false},          // src lacks ':'
	}
	for _, c := range cases {
		conn := bbPpFeed(t, []byte(c.line))
		addr, _, err := readProxyProtoV1Header(conn)
		if c.ok {
			if err != nil || addr == nil {
				t.Fatalf("%q should parse: addr=%v err=%v", c.line, addr, err)
			}
		} else if !errors.Is(err, errProxyProtoInvalid) {
			t.Fatalf("%q err=%v want invalid", c.line, err)
		}
	}
}

// Detail 7: v1 ports must parse as uint16 decimals; errors propagate as
// port errors.
func TestDetail07_V1Ports(t *testing.T) {
	for _, line := range []string{
		"TCP4 1.2.3.4 5.6.7.8 abc 1\r\n",
		"TCP4 1.2.3.4 5.6.7.8 1 xyz\r\n",
		"TCP4 1.2.3.4 5.6.7.8 65536 1\r\n", // > uint16
		"TCP4 1.2.3.4 5.6.7.8 -1 1\r\n",
		"TCP4 1.2.3.4 5.6.7.8 1 999999\r\n",
	} {
		conn := bbPpFeed(t, []byte(line))
		_, _, err := readProxyProtoV1Header(conn)
		if err == nil {
			t.Fatalf("%q should error", line)
		}
	}
	// Boundary ports parse.
	conn := bbPpFeed(t, []byte("TCP4 1.2.3.4 5.6.7.8 0 65535\r\n"))
	addr, _, err := readProxyProtoV1Header(conn)
	if err != nil || addr == nil {
		t.Fatalf("boundary ports: addr=%v err=%v", addr, err)
	}
	if addr.srcPort != 0 || addr.dstPort != 65535 {
		t.Fatalf("ports=%d/%d", addr.srcPort, addr.dstPort)
	}
}

// Detail 8: the auto-detect entry sets a 5s read deadline and clears it
// afterwards.
func TestDetail08_DeadlineSetAndCleared(t *testing.T) {
	inner := bbPpFeed(t, []byte("PROXY TCP4 1.2.3.4 5.6.7.8 1 2\r\n"))
	rec := &bbPpRecConn{Conn: inner}
	addr, _, err := readProxyProtoHeader(rec)
	if err != nil || addr == nil {
		t.Fatalf("addr=%v err=%v", addr, err)
	}
	dls := rec.Recorded()
	if len(dls) == 0 {
		t.Fatal("no SetReadDeadline recorded")
	}
	// First deadline ~ now+5s; last clears (zero time).
	now := time.Now()
	first := dls[0]
	if first.Before(now.Add(4*time.Second)) || first.After(now.Add(7*time.Second)) {
		t.Fatalf("deadline=%v not ~5s out (const=%v)", first, proxyProtoReadTimeout)
	}
	last := dls[len(dls)-1]
	if !last.IsZero() {
		t.Fatalf("last deadline=%v want zero (cleared)", last)
	}
	// Conn usable after return without a deadline-induced timeout.
	inner.SetReadDeadline(time.Time{})
}

// Detail 9: v2 dispatch — auto-detect reads remaining 6 sig bytes,
// validates full 12-byte sig, then 4-byte header; standalone reader takes
// all 16 at once and maps a read timeout to the timeout error.
func TestDetail09_V2DispatchAndTimeout(t *testing.T) {
	full := bbPpV2Header(proxyProtoV2Ver|proxyProtoCmdProxy,
		proxyProtoFamilyInet|proxyProtoProtoStream,
		bbPpIPv4Addrs(net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8), 1111, 2222, 0))
	// Auto-detect path.
	conn := bbPpFeed(t, full)
	addr, _, err := readProxyProtoHeader(conn)
	if err != nil || addr == nil {
		t.Fatalf("auto v2: addr=%v err=%v", addr, err)
	}
	if !addr.srcIP.Equal(net.IPv4(1, 2, 3, 4)) || addr.srcPort != 1111 {
		t.Fatalf("v2 addr=%v", addr)
	}
	// Standalone reader takes all 16 at once.
	conn = bbPpFeed(t, full)
	addr, err = readProxyProtoV2Header(conn)
	if err != nil || addr == nil {
		t.Fatalf("standalone v2: addr=%v err=%v", addr, err)
	}
	// Bad signature -> invalid.
	bad := append([]byte(nil), full...)
	bad[0] = 'X'
	conn = bbPpFeed(t, bad)
	if _, _, err = readProxyProtoHeader(conn); err == nil {
		t.Fatal("bad sig should error")
	}
	conn = bbPpFeed(t, bad)
	if _, err = readProxyProtoV2Header(conn); err == nil {
		t.Fatal("standalone bad sig should error")
	}
	// Truncated header -> error.
	conn = bbPpFeed(t, full[:10])
	if _, err = readProxyProtoV2Header(conn); err == nil {
		t.Fatal("truncated header should error")
	}
	// Read timeout maps to errProxyProtoTimeout.
	tc := &bbPpTimeoutConn{Conn: bbPpFeed(t, nil)}
	if _, err = readProxyProtoV2Header(tc); !errors.Is(err, errProxyProtoTimeout) {
		t.Fatalf("timeout err=%v want errProxyProtoTimeout", err)
	}
}

// Detail 10: v2 version nibble must be 0x20.
func TestDetail10_V2VersionNibble(t *testing.T) {
	for _, vercmd := range []byte{0x11, 0x30 | proxyProtoCmdProxy, 0x00, 0x41} {
		full := bbPpV2Header(vercmd, proxyProtoFamilyInet|proxyProtoProtoStream,
			bbPpIPv4Addrs(net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8), 1, 2, 0))
		conn := bbPpFeed(t, full)
		if _, _, err := readProxyProtoHeader(conn); err == nil {
			t.Fatalf("ver/cmd=%#x should error", vercmd)
		}
	}
	// 0x20|PROXY is the only accepted combo.
	full := bbPpV2Header(0x20|proxyProtoCmdProxy, proxyProtoFamilyInet|proxyProtoProtoStream,
		bbPpIPv4Addrs(net.IPv4(9, 9, 9, 9), net.IPv4(8, 8, 8, 8), 3, 4, 0))
	conn := bbPpFeed(t, full)
	if _, _, err := readProxyProtoHeader(conn); err != nil {
		t.Fatalf("ver 0x20: %v", err)
	}
}

// Detail 11: LOCAL command — consumes and discards announced address
// bytes, returns nil,nil.
func TestDetail11_V2LocalCommand(t *testing.T) {
	full := bbPpV2Header(proxyProtoV2Ver|proxyProtoCmdLocal,
		proxyProtoFamilyUnspec|proxyProtoProtoUnspec, nil)
	conn := bbPpFeed(t, full)
	addr, err := readProxyProtoV2Header(conn)
	if err != nil || addr != nil {
		t.Fatalf("LOCAL addr=%v err=%v want nil,nil", addr, err)
	}
	// LOCAL with announced bytes: consumed and discarded.
	full = bbPpV2Header(proxyProtoV2Ver|proxyProtoCmdLocal,
		proxyProtoFamilyInet|proxyProtoProtoStream,
		bbPpIPv4Addrs(net.IPv4(1, 1, 1, 1), net.IPv4(2, 2, 2, 2), 1, 1, 0))
	conn = bbPpFeed(t, full)
	addr, err = readProxyProtoV2Header(conn)
	if err != nil || addr != nil {
		t.Fatalf("LOCAL+addrs addr=%v err=%v want nil,nil", addr, err)
	}
	// Through auto-detect too.
	conn = bbPpFeed(t, append(append([]byte(nil), full...), []byte("NEXT")...))
	addr, _, err = readProxyProtoHeader(conn)
	if err != nil || addr != nil {
		t.Fatalf("auto LOCAL addr=%v err=%v", addr, err)
	}
}

// Detail 12: non-LOCAL non-PROXY commands -> "unknown command"; non-STREAM
// transport -> unsupported; unknown family -> unsupported.
func TestDetail12_V2CommandFamilyValidation(t *testing.T) {
	// Unknown command (ver 0x20, cmd 0x02).
	full := bbPpV2Header(0x20|0x02, proxyProtoFamilyInet|proxyProtoProtoStream,
		bbPpIPv4Addrs(net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8), 1, 2, 0))
	conn := bbPpFeed(t, full)
	if _, _, err := readProxyProtoHeader(conn); err == nil {
		t.Fatal("cmd=2 should error (unknown command)")
	}
	// DGRAM transport -> unsupported.
	full = bbPpV2Header(0x20|proxyProtoCmdProxy, proxyProtoFamilyInet|proxyProtoProtoDatagram,
		bbPpIPv4Addrs(net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8), 1, 2, 0))
	conn = bbPpFeed(t, full)
	if _, _, err := readProxyProtoHeader(conn); !errors.Is(err, errProxyProtoUnsupported) {
		t.Fatalf("DGRAM err=%v want unsupported", err)
	}
	// UNIX family -> unsupported.
	full = bbPpV2Header(0x20|proxyProtoCmdProxy, proxyProtoFamilyUnix|proxyProtoProtoStream, nil)
	conn = bbPpFeed(t, full)
	if _, _, err := readProxyProtoHeader(conn); !errors.Is(err, errProxyProtoUnsupported) {
		t.Fatalf("UNIX family err=%v want unsupported", err)
	}
	// Bogus family nibble.
	full = bbPpV2Header(0x20|proxyProtoCmdProxy, 0xF0|proxyProtoProtoStream, nil)
	conn = bbPpFeed(t, full)
	if _, _, err := readProxyProtoHeader(conn); !errors.Is(err, errProxyProtoUnsupported) {
		t.Fatalf("family 0xF0 err=%v want unsupported", err)
	}
}

// Detail 13: UNSPEC family with PROXY command — discard announced bytes,
// return nil,nil.
func TestDetail13_V2UnspecFamily(t *testing.T) {
	// UNSPEC family still requires the STREAM transport nibble; the
	// announced address bytes are discarded.
	full := bbPpV2Header(proxyProtoV2Ver|proxyProtoCmdProxy,
		proxyProtoFamilyUnspec|proxyProtoProtoStream,
		[]byte{0xDE, 0xAD, 0xBE, 0xEF})
	conn := bbPpFeed(t, full)
	addr, err := readProxyProtoV2Header(conn)
	if err != nil || addr != nil {
		t.Fatalf("UNSPEC addr=%v err=%v want nil,nil", addr, err)
	}
	// Zero announced bytes too.
	full = bbPpV2Header(proxyProtoV2Ver|proxyProtoCmdProxy,
		proxyProtoFamilyUnspec|proxyProtoProtoStream, nil)
	conn = bbPpFeed(t, full)
	addr, _, err = readProxyProtoHeader(conn)
	if err != nil || addr != nil {
		t.Fatalf("UNSPEC empty addr=%v err=%v", addr, err)
	}
}

// Detail 14: address parsing — INET needs >=12 bytes, INET6 >=36;
// announced length may EXCEED minimum (extra bytes read and ignored =
// TLVs).
func TestDetail14_AddrLengthAndTLVs(t *testing.T) {
	src, dst := net.IPv4(10, 0, 0, 1), net.IPv4(10, 0, 0, 2)
	// Exact 12.
	conn := bbPpFeed(t, bbPpIPv4Addrs(src, dst, 1234, 5678, 0))
	addr, err := parseIPv4Addr(conn, proxyProtoAddrSizeIPv4)
	if err != nil || addr == nil {
		t.Fatalf("v4 exact: %v", err)
	}
	if !addr.srcIP.Equal(src) || addr.srcPort != 1234 || !addr.dstIP.Equal(dst) || addr.dstPort != 5678 {
		t.Fatalf("v4 addr=%+v", addr)
	}
	// Exceeding length: extra bytes (TLVs) read and ignored.
	rng := bbPpRng(t)
	extra := 1 + rng.Intn(20)
	conn = bbPpFeed(t, bbPpIPv4Addrs(src, dst, 4321, 8765, extra))
	addr, err = parseIPv4Addr(conn, uint16(proxyProtoAddrSizeIPv4+extra))
	if err != nil || addr == nil {
		t.Fatalf("v4+%d TLVs: %v", extra, err)
	}
	if addr.srcPort != 4321 {
		t.Fatalf("v4 TLV addr=%+v", addr)
	}
	// Short -> error.
	conn = bbPpFeed(t, bbPpIPv4Addrs(src, dst, 1, 2, 0))
	if _, err = parseIPv4Addr(conn, proxyProtoAddrSizeIPv4-1); err == nil {
		t.Fatal("v4 len 11 should error")
	}
	// IPv6 exact 36.
	s6 := net.ParseIP("2001:db8::1")
	d6 := net.ParseIP("2001:db8::2")
	conn = bbPpFeed(t, bbPpIPv6Addrs(s6, d6, 1111, 2222, 0))
	addr, err = parseIPv6Addr(conn, proxyProtoAddrSizeIPv6)
	if err != nil || addr == nil {
		t.Fatalf("v6 exact: %v", err)
	}
	if !addr.srcIP.Equal(s6) || addr.srcPort != 1111 {
		t.Fatalf("v6 addr=%+v", addr)
	}
	// IPv6 + TLVs.
	conn = bbPpFeed(t, bbPpIPv6Addrs(s6, d6, 9, 8, extra))
	addr, err = parseIPv6Addr(conn, uint16(proxyProtoAddrSizeIPv6+extra))
	if err != nil || addr == nil || addr.srcPort != 9 {
		t.Fatalf("v6+%d: addr=%v err=%v", extra, addr, err)
	}
	// IPv6 short -> error.
	conn = bbPpFeed(t, bbPpIPv6Addrs(s6, d6, 1, 2, 0))
	if _, err = parseIPv6Addr(conn, proxyProtoAddrSizeIPv6-1); err == nil {
		t.Fatal("v6 len 35 should error")
	}
	// End-to-end via header: announced len exceeding actual addr bytes.
	full := bbPpV2Header(proxyProtoV2Ver|proxyProtoCmdProxy,
		proxyProtoFamilyInet|proxyProtoProtoStream,
		bbPpIPv4Addrs(src, dst, 7777, 8888, 7))
	conn = bbPpFeed(t, full)
	addr, _, err = readProxyProtoHeader(conn)
	if err != nil || addr == nil || addr.srcPort != 7777 {
		t.Fatalf("e2e TLV: addr=%v err=%v", addr, err)
	}
}

// Detail 15: proxyProtoAddr.String = JoinHostPort(srcIP,srcPort);
// Network() = "tcp4" iff srcIP v4-compatible else "tcp6".
func TestDetail15_AddrFormatting(t *testing.T) {
	a := &proxyProtoAddr{srcIP: net.IPv4(1, 2, 3, 4), srcPort: 5678}
	if a.String() != "1.2.3.4:5678" {
		t.Fatalf("String=%q want 1.2.3.4:5678", a.String())
	}
	if a.Network() != "tcp4" {
		t.Fatalf("Network=%q want tcp4", a.Network())
	}
	b := &proxyProtoAddr{srcIP: net.ParseIP("2001:db8::1"), srcPort: 443}
	if b.String() != "[2001:db8::1]:443" {
		t.Fatalf("v6 String=%q", b.String())
	}
	if b.Network() != "tcp6" {
		t.Fatalf("v6 Network=%q want tcp6", b.Network())
	}
	// IPv4-mapped v6 src is v4-compatible.
	c := &proxyProtoAddr{srcIP: net.ParseIP("::ffff:1.2.3.4"), srcPort: 1}
	if c.Network() != "tcp4" {
		t.Fatalf("mapped Network=%q want tcp4", c.Network())
	}
}

// Detail 16: proxyConn.RemoteAddr returns the extracted address.
func TestDetail16_ProxyConnRemoteAddr(t *testing.T) {
	w, r := net.Pipe()
	defer w.Close()
	defer r.Close()
	a := &proxyProtoAddr{srcIP: net.IPv4(9, 8, 7, 6), srcPort: 4321}
	pc := &proxyConn{Conn: r, remoteAddr: a}
	got := pc.RemoteAddr()
	if got != a {
		t.Fatalf("RemoteAddr=%v want the extracted %v", got, a)
	}
	if got.String() != "9.8.7.6:4321" {
		t.Fatalf("RemoteAddr.String=%q", got)
	}
	// The wrapped conn still works for I/O.
	go w.Write([]byte("x"))
	buf := make([]byte, 1)
	if _, err := pc.Read(buf); err != nil || buf[0] != 'x' {
		t.Fatalf("wrapped Read err=%v", err)
	}
}
