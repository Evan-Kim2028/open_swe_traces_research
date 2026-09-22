package server

import (
	"bytes"
	"testing"
)

// TestDetail01: wsFillFrameHeader — byte0 = type|final|rsv1 (type bits only
// when first); byte1 mask bit; length tiers 125/126/127.
func TestDetail01(t *testing.T) {
	fh := make([]byte, wsMaxFrameHeaderSize)
	n, key := wsFillFrameHeader(fh, false, true, true, false, wsTextMessage, 3)
	if n != 2 || fh[0] != wsFinalBit|byte(wsTextMessage) || fh[1] != 3 || key != nil {
		t.Fatalf("small frame: n=%d %x", n, fh[:n])
	}
	n, _ = wsFillFrameHeader(fh, false, true, false, true, wsBinaryMessage, 200)
	if n != 4 || fh[0] != byte(wsBinaryMessage)|wsRsv1Bit || fh[1] != 126 || fh[2] != 0 || fh[3] != 200 {
		t.Fatalf("126-marker frame: n=%d %x", n, fh[:n])
	}
	n, _ = wsFillFrameHeader(fh, false, true, true, false, wsBinaryMessage, 70000)
	if n != 10 || fh[1] != 127 || fh[9] != 0x70 {
		t.Fatalf("127-marker frame: n=%d %x", n, fh[:n])
	}
	// Continuation frame: first=false leaves the type bits at 0.
	n, _ = wsFillFrameHeader(fh, false, false, true, false, wsBinaryMessage, 5)
	if n != 2 || fh[0] != wsFinalBit {
		t.Fatalf("continuation frame: n=%d %x", n, fh[:n])
	}
}

// TestDetail02: with masking, byte1 gains the mask bit and a 4-byte key is
// generated and returned, written right after the length field.
func TestDetail02(t *testing.T) {
	fh := make([]byte, wsMaxFrameHeaderSize)
	n, key := wsFillFrameHeader(fh, true, true, true, false, wsBinaryMessage, 200)
	if n != 8 || fh[1]&wsMaskBit == 0 {
		t.Fatalf("masked frame: n=%d %x", n, fh[:n])
	}
	if len(key) != 4 || !bytes.Equal(fh[4:8], key) {
		t.Fatalf("key should occupy header[4:8]: %x vs %x", fh[4:8], key)
	}
}

// TestDetail03: wsCreateFrameHeader returns a pooled wsMaxFrameHeaderSize
// buffer sliced to the header length.
func TestDetail03(t *testing.T) {
	h, key := wsCreateFrameHeader(true, false, wsTextMessage, 5)
	if len(h) != 6 || cap(h) < wsMaxFrameHeaderSize {
		t.Fatalf("pooled header: len=%d cap=%d", len(h), cap(h))
	}
	if len(key) != 4 {
		t.Fatalf("masking key len=%d", len(key))
	}
	nbPoolPut(h)
}

// TestDetail04: wsMaskBufs continues the key position across buffers.
func TestDetail04(t *testing.T) {
	key := []byte{1, 2, 3, 4}
	b1 := []byte("abc")
	b2 := []byte("defg")
	wsMaskBufs(key, [][]byte{b1, b2})
	// Contiguous masking: b2[0] uses key[3], not key[0].
	if b2[0] != 'd'^key[3] || b2[1] != 'e'^key[0] {
		t.Fatalf("mask position must continue across buffers: %x %x", b1, b2)
	}
	if b1[0] != 'a'^key[0] {
		t.Fatalf("b1 = %x", b1)
	}
	// Masking is symmetric.
	wsMaskBuf(key, b1)
	if string(b1) != "abc" {
		t.Fatalf("unmask failed: %q", b1)
	}
}

// TestDetail05: unmask honours the running mkpos across calls.
func TestDetail05(t *testing.T) {
	r := &wsReadInfo{mkpos: 2, mkey: [4]byte{1, 2, 3, 4}}
	buf := []byte("012345")
	r.unmask(buf)
	if buf[0] != '0'^3 || buf[1] != '1'^4 || buf[2] != '2'^1 {
		t.Fatalf("unmask should start at key pos 2: %x", buf)
	}
	if r.mkpos != 0 {
		t.Fatalf("mkpos should wrap mod 4: %d", r.mkpos)
	}
	// A frame split across calls continues where it left off.
	r2 := &wsReadInfo{mkey: [4]byte{1, 2, 3, 4}}
	a := []byte("01234")
	b := []byte("567")
	r2.unmask(a)
	if r2.mkpos != 1 {
		t.Fatalf("mkpos after 5 bytes = %d", r2.mkpos)
	}
	r2.unmask(b)
	if b[0] != '5'^2 || r2.mkpos != 0 {
		t.Fatalf("cross-call unmask: %x pos=%d", b, r2.mkpos)
	}
}

// TestDetail06: wsIsControlFrame is opcode >= wsCloseMessage — unknown
// opcodes >= 8 count too.
func TestDetail06(t *testing.T) {
	for _, op := range []wsOpCode{wsTextMessage, wsBinaryMessage, wsOpCode(7)} {
		if wsIsControlFrame(op) {
			t.Fatalf("op %d is not control", op)
		}
	}
	for _, op := range []wsOpCode{wsCloseMessage, wsPingMessage, wsPongMessage, wsOpCode(11), wsOpCode(15)} {
		if !wsIsControlFrame(op) {
			t.Fatalf("op %d should be control", op)
		}
	}
}

// TestDetail07: wsIsValidCloseStatus rejects <1000, >=5000, the named
// reserved codes 1004/1005/1006/1015, and the reserved range 1016-2999.
func TestDetail07(t *testing.T) {
	valid := []int{1000, 1001, 1002, 1003, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 3000, 4000, 4999}
	for _, c := range valid {
		if !wsIsValidCloseStatus(c) {
			t.Fatalf("%d should be valid", c)
		}
	}
	invalid := []int{999, 1004, 1005, 1006, 1015, 1016, 2000, 2999, 5000, 5001}
	for _, c := range invalid {
		if wsIsValidCloseStatus(c) {
			t.Fatalf("%d should be invalid", c)
		}
	}
}

// TestDetail08: wsCreateCloseMessage = u16 BE status + body, truncated to
// wsMaxControlPayloadSize-2 with a "..." tail.
func TestDetail08(t *testing.T) {
	cm := wsCreateCloseMessage(1000, "bye")
	if len(cm) != 5 || cm[0] != 0x03 || cm[1] != 0xe8 || string(cm[2:]) != "bye" {
		t.Fatalf("close message = %x", cm)
	}
	nbPoolPut(cm)
	long := string(make([]byte, 200))
	cm2 := wsCreateCloseMessage(1000, long)
	if len(cm2) != wsMaxControlPayloadSize {
		t.Fatalf("truncated close message len=%d", len(cm2))
	}
	if string(cm2[len(cm2)-3:]) != "..." {
		t.Fatalf("truncation should append ellipsis: %x", cm2[len(cm2)-6:])
	}
	nbPoolPut(cm2)
}

// TestDetail09: wsGet is zero-copy when the buffer suffices; otherwise it
// copies the tail and reads the rest from r, advancing pos only past
// consumed buffer bytes.
func TestDetail09(t *testing.T) {
	src := bytes.NewReader([]byte("REST"))
	b, pos, err := wsGet(src, []byte("xy"), 2, 4)
	if err != nil || string(b) != "REST" || pos != 2 {
		t.Fatalf("read-through: %q pos=%d err=%v", b, pos, err)
	}
	b2, pos2, err := wsGet(src, []byte("yz12"), 1, 2)
	if err != nil || string(b2) != "z1" || pos2 != 3 {
		t.Fatalf("zero-copy: %q pos=%d err=%v", b2, pos2, err)
	}
	// Mixed: tail of buffer plus reader bytes.
	b3, pos3, err := wsGet(bytes.NewReader([]byte("34")), []byte("yz12"), 2, 4)
	if err != nil || string(b3) != "1234" || pos3 != 4 {
		t.Fatalf("mixed: %q pos=%d err=%v", b3, pos3, err)
	}
}

// TestDetail10: wsMaxMessageSize = mpay*8 capped at 64MB; mpay<=0 uses
// MAX_PAYLOAD_SIZE.
func TestDetail10(t *testing.T) {
	if got := wsMaxMessageSize(100); got != 100*wsMaxMsgPayloadMultiple {
		t.Fatalf("maxMessageSize(100) = %d", got)
	}
	if got := wsMaxMessageSize(1 << 30); got != wsMaxMsgPayloadLimit {
		t.Fatalf("cap = %d", got)
	}
	for _, mp := range []int{-1, 0} {
		if got := wsMaxMessageSize(mp); got != MAX_PAYLOAD_SIZE*wsMaxMsgPayloadMultiple {
			t.Fatalf("mpay=%d: %d", mp, got)
		}
	}
}
