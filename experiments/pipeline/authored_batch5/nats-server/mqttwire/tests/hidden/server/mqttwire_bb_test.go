package server

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/nats-io/jwt/v2"
)

// TestDetail01: readVarInt — 7 bits per byte LSB-first, 0x80 continuation;
// (v,true,nil) complete, (0,false,nil) mid-value, errMQTTMalformedVarInt past
// the 4-byte cap.
func TestDetail01(t *testing.T) {
	cases := []struct {
		b    []byte
		v    int
		comp bool
	}{
		{[]byte{0x00}, 0, true},
		{[]byte{0x7f}, 127, true},
		{[]byte{0x80, 0x01}, 128, true},
		{[]byte{0xac, 0x02}, 300, true},
		{[]byte{0xff, 0xff, 0xff, 0x7f}, 268435455, true},
		{[]byte{0x01, 0x99}, 1, true}, // stops at first terminating byte
	}
	for _, tc := range cases {
		r := &mqttReader{}
		r.reset(tc.b)
		v, comp, err := r.readVarInt()
		if err != nil || v != tc.v || comp != tc.comp {
			t.Fatalf("varint %x = %d,%v,%v; want %d,%v", tc.b, v, comp, err, tc.v, tc.comp)
		}
	}
	// Exhausted mid-value → incomplete, not an error.
	r := &mqttReader{}
	r.reset([]byte{0x80})
	if v, comp, err := r.readVarInt(); v != 0 || comp || err != nil {
		t.Fatalf("mid-value = %d,%v,%v", v, comp, err)
	}
	// Past the 4-byte cap → malformed.
	for _, b := range [][]byte{{0xff, 0xff, 0xff, 0xff, 0x7f}, {0x80, 0x80, 0x80, 0x80}} {
		r := &mqttReader{}
		r.reset(b)
		if _, _, err := r.readVarInt(); !errors.Is(err, errMQTTMalformedVarInt) {
			t.Fatalf("varint %x: want malformed-varint error, got %v", b, err)
		}
	}
}

// TestDetail02: readPacketLen — complete → (v,true,nil); incomplete stashes
// buf[pstart:] into pbuf; the maxLen check covers the whole packet and on
// violation returns ErrMaxPayload without touching pbuf.
func TestDetail02(t *testing.T) {
	r := &mqttReader{}
	r.reset([]byte{0x03, 'a', 'b', 'c'})
	if v, comp, err := r.readPacketLen(100); v != 3 || !comp || err != nil {
		t.Fatalf("complete packet = %d,%v,%v", v, comp, err)
	}

	// Incomplete: payload shorter than declared → stash whole buffer.
	r = &mqttReader{}
	r.reset([]byte{0x05, 'a', 'b', 'c'})
	if v, comp, err := r.readPacketLen(100); v != 0 || comp || err != nil {
		t.Fatalf("incomplete packet = %d,%v,%v", v, comp, err)
	}
	if !bytes.Equal(r.pbuf, []byte{0x05, 'a', 'b', 'c'}) {
		t.Fatalf("pbuf = %x", r.pbuf)
	}

	// Incomplete varint also stashes.
	r = &mqttReader{}
	r.reset([]byte{0x80})
	if _, comp, err := r.readPacketLen(100); comp || err != nil {
		t.Fatalf("incomplete varint = %v,%v", comp, err)
	}
	if !bytes.Equal(r.pbuf, []byte{0x80}) {
		t.Fatalf("pbuf = %x", r.pbuf)
	}

	// maxLen covers the whole packet (fixed header + payload).
	r = &mqttReader{}
	r.reset([]byte{0x03, 'a', 'b', 'c'})
	if _, _, err := r.readPacketLen(2); !errors.Is(err, ErrMaxPayload) {
		t.Fatalf("whole-packet max violation: %v", err)
	}
	if len(r.pbuf) != 0 {
		t.Fatalf("pbuf must be untouched on max violation, got %x", r.pbuf)
	}
	r = &mqttReader{}
	r.reset([]byte{0x03, 'a', 'b', 'c'})
	if _, comp, err := r.readPacketLen(4); !comp || err != nil {
		t.Fatalf("4-byte packet under maxLen 4 = %v,%v", comp, err)
	}
	r = &mqttReader{}
	r.reset([]byte{0x03, 'a', 'b', 'c'})
	if _, comp, err := r.readPacketLen(jwt.NoLimit); !comp || err != nil {
		t.Fatalf("NoLimit: %v,%v", comp, err)
	}
}

// TestDetail03: reset prepends pending pbuf, clears it, zeroes pos/pstart.
func TestDetail03(t *testing.T) {
	r := &mqttReader{}
	r.reset([]byte{0x80})
	if _, _, err := r.readPacketLen(100); err != nil {
		t.Fatal(err)
	}
	r.reset([]byte{0x01, 'Z'})
	if !bytes.Equal(r.buf, []byte{0x80, 0x01, 'Z'}) {
		t.Fatalf("reset must prepend pbuf, buf=%x", r.buf)
	}
	if len(r.pbuf) != 0 || r.pos != 0 || r.pstart != 0 {
		t.Fatalf("reset state: pbuf=%x pos=%d pstart=%d", r.pbuf, r.pos, r.pstart)
	}
	// The combined buffer now decodes: varint 0x80,0x01 → 128... too long for
	// the 1-byte payload — just verify the varint consumed correctly.
	if v, comp, err := r.readVarInt(); err != nil || !comp || v != 128 {
		t.Fatalf("varint over carried buffer = %d,%v,%v", v, comp, err)
	}
}

// TestDetail04: readBytes/readString are uint16-length-prefixed; zero length
// yields nil/""; cp=false aliases the buffer while cp=true copies; short
// buffers produce field-named unexpected-EOF/EOF errors.
func TestDetail04(t *testing.T) {
	r := &mqttReader{}
	r.reset([]byte{0, 0, 'x'})
	bs, err := r.readBytes("f", false)
	if err != nil || len(bs) != 0 || r.pos != 2 {
		t.Fatalf("zero-len bytes = %v,%v pos=%d", bs, err, r.pos)
	}
	r.reset([]byte{0, 0})
	s, err := r.readString("f")
	if err != nil || s != "" || r.pos != 2 {
		t.Fatalf("zero-len string = %q,%v pos=%d", s, err, r.pos)
	}

	// Alias vs copy.
	r.reset([]byte{0, 2, 'a', 'b'})
	alias, _ := r.readBytes("f", false)
	if &alias[0] != &r.buf[2] {
		t.Fatal("cp=false must alias the read buffer")
	}
	r.reset([]byte{0, 2, 'a', 'b'})
	cp, _ := r.readBytes("f", true)
	cp[0] = 'Z'
	if r.buf[2] != 'a' {
		t.Fatal("cp=true must copy")
	}

	// Error shapes: field name plus the EOF family text.
	r.reset([]byte{0x05})
	if _, err := r.readUint16("myfield"); err == nil || !strings.Contains(err.Error(), "myfield") ||
		!strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("short uint16 err = %v", err)
	}
	r.reset(nil)
	if _, err := r.readByte("mybyte"); err == nil || !strings.Contains(err.Error(), "mybyte") ||
		!strings.Contains(err.Error(), "EOF") {
		t.Fatalf("empty byte err = %v", err)
	}
	r.reset([]byte{0, 5, 'a'})
	if _, err := r.readBytes("mybytes", false); err == nil || !strings.Contains(err.Error(), "mybytes") ||
		!strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("short bytes err = %v", err)
	}
}

// TestDetail05: mqttCheckFixedHeaderFlags — CONNECT/PUBACK/PUBREC/PUBCOMP/
// PINGREQ/DISCONNECT need 0; PUBREL/SUBSCRIBE/UNSUBSCRIBE need 0x2; PUBLISH
// and unknown types accept anything.
func TestDetail05(t *testing.T) {
	for _, pt := range []byte{mqttPacketConnect, mqttPacketPubAck, mqttPacketPubRec, mqttPacketPubComp, mqttPacketPing, mqttPacketDisconnect} {
		if err := mqttCheckFixedHeaderFlags(pt, 0); err != nil {
			t.Fatalf("pt=%02x flags=0: %v", pt, err)
		}
		if err := mqttCheckFixedHeaderFlags(pt, 1); err == nil {
			t.Fatalf("pt=%02x flags=1 must fail", pt)
		}
	}
	for _, pt := range []byte{mqttPacketPubRel, mqttPacketSub, mqttPacketUnsub} {
		if err := mqttCheckFixedHeaderFlags(pt, 2); err != nil {
			t.Fatalf("pt=%02x flags=2: %v", pt, err)
		}
		if err := mqttCheckFixedHeaderFlags(pt, 0); err == nil {
			t.Fatalf("pt=%02x flags=0 must fail", pt)
		}
	}
	for _, f := range []byte{0, 0xf, 0x8} {
		if err := mqttCheckFixedHeaderFlags(mqttPacketPub, f); err != nil {
			t.Fatalf("PUBLISH flags=%x: %v", f, err)
		}
	}
	// Unknown packet types pass.
	if err := mqttCheckFixedHeaderFlags(0xf0, 7); err != nil {
		t.Fatalf("unknown type: %v", err)
	}
	if err := mqttCheckFixedHeaderFlags(mqttPacketSubAck, 9); err != nil {
		t.Fatalf("SUBACK passes any flags: %v", err)
	}
}

// TestDetail06: mqttCheckRemainingLength — CONNECT/PUBLISH/SUBSCRIBE/
// UNSUBSCRIBE accept any; PUBACK/PUBREC/PUBREL/PUBCOMP need 2; PINGREQ/
// DISCONNECT need 0; unknown types pass.
func TestDetail06(t *testing.T) {
	for _, pt := range []byte{mqttPacketConnect, mqttPacketPub, mqttPacketSub, mqttPacketUnsub} {
		if err := mqttCheckRemainingLength(pt, 99); err != nil {
			t.Fatalf("pt=%02x any-length: %v", pt, err)
		}
	}
	for _, pt := range []byte{mqttPacketPubAck, mqttPacketPubRec, mqttPacketPubRel, mqttPacketPubComp} {
		if err := mqttCheckRemainingLength(pt, 2); err != nil {
			t.Fatalf("pt=%02x pl=2: %v", pt, err)
		}
		if err := mqttCheckRemainingLength(pt, 3); err == nil {
			t.Fatalf("pt=%02x pl=3 must fail", pt)
		}
	}
	for _, pt := range []byte{mqttPacketPing, mqttPacketDisconnect} {
		if err := mqttCheckRemainingLength(pt, 0); err != nil {
			t.Fatalf("pt=%02x pl=0: %v", pt, err)
		}
		if err := mqttCheckRemainingLength(pt, 1); err == nil {
			t.Fatalf("pt=%02x pl=1 must fail", pt)
		}
	}
	if err := mqttCheckRemainingLength(0xf0, 3); err != nil {
		t.Fatalf("unknown type: %v", err)
	}
}

// TestDetail07: mqttParsePIPacket reads a big-endian uint16 and rejects 0.
func TestDetail07(t *testing.T) {
	r := &mqttReader{}
	r.reset([]byte{0x12, 0x34})
	if pi, err := mqttParsePIPacket(r); pi != 0x1234 || err != nil {
		t.Fatalf("pi = %d, %v", pi, err)
	}
	r.reset([]byte{0, 0})
	if _, err := mqttParsePIPacket(r); !errors.Is(err, errMQTTPacketIdentifierIsZero) {
		t.Fatalf("PI=0: %v", err)
	}
}

// TestDetail08: mqttGetQoS = (flags & 0x6) >> 1; mqttIsRetained = flags & 0x1.
func TestDetail08(t *testing.T) {
	for f, want := range map[byte]byte{0: 0, 1: 0, 2: 1, 4: 2, 6: 3, 8: 0, 0xf: 3} {
		if q := mqttGetQoS(f); q != want {
			t.Fatalf("qos(%02x) = %d, want %d", f, q, want)
		}
	}
	for _, f := range []byte{1, 3, 0xf} {
		if !mqttIsRetained(f) {
			t.Fatalf("flags %02x should be retained", f)
		}
	}
	for _, f := range []byte{0, 2, 0xe} {
		if mqttIsRetained(f) {
			t.Fatalf("flags %02x should not be retained", f)
		}
	}
}

// TestDetail09: writer primitives — big-endian uint16, length-prefixed
// string/bytes, LSB-first varint with at least one byte; newMQTTWriter
// pre-grows the buffer.
func TestDetail09(t *testing.T) {
	w := newMQTTWriter(0)
	w.WriteUint16(0x1234)
	if !bytes.Equal(w.Bytes(), []byte{0x12, 0x34}) {
		t.Fatalf("WriteUint16 = %x", w.Bytes())
	}
	w.Reset()
	w.WriteString("ab")
	if !bytes.Equal(w.Bytes(), []byte{0, 2, 'a', 'b'}) {
		t.Fatalf("WriteString = %x", w.Bytes())
	}
	w.Reset()
	w.WriteBytes([]byte{9, 8})
	if !bytes.Equal(w.Bytes(), []byte{0, 2, 9, 8}) {
		t.Fatalf("WriteBytes = %x", w.Bytes())
	}
	for v, want := range map[int][]byte{0: {0x00}, 127: {0x7f}, 128: {0x80, 0x01}, 16384: {0x80, 0x80, 0x01}} {
		w.Reset()
		w.WriteVarInt(v)
		if !bytes.Equal(w.Bytes(), want) {
			t.Fatalf("WriteVarInt(%d) = %x, want %x", v, w.Bytes(), want)
		}
	}
	if w2 := newMQTTWriter(8); cap(w2.Bytes()) < 8 {
		t.Fatalf("newMQTTWriter(8) cap = %d", cap(w2.Bytes()))
	}
}
