package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	"github.com/minio/highwayhash"
)

func mrBbDigest() *highwayhash.Digest64 {
	key := sha256.Sum256([]byte("mr-bb"))
	hh, _ := highwayhash.NewDigest64(key[:])
	return hh
}

// mrBbRec builds one on-disk record: rl(4) seq(8) ts(8) slen(2) subj
// [hlen(4) hdr] msg checksum(8).
func mrBbRec(hh *highwayhash.Digest64, seq uint64, ts int64, subj string, hdr, msg []byte) []byte {
	le := binary.LittleEndian
	rl := uint32(msgHdrSize + len(subj) + len(msg) + checksumSize)
	if len(hdr) > 0 {
		rl += uint32(4 + len(hdr))
	}
	l := rl
	if len(hdr) > 0 {
		l |= hbit
	}
	var hb [msgHdrSize]byte
	le.PutUint32(hb[0:], l)
	le.PutUint64(hb[4:], seq)
	le.PutUint64(hb[12:], uint64(ts))
	le.PutUint16(hb[20:], uint16(len(subj)))
	var b []byte
	b = append(b, hb[:]...)
	b = append(b, subj...)
	if len(hdr) > 0 {
		var hl [4]byte
		le.PutUint32(hl[0:], uint32(len(hdr)))
		b = append(b, hl[:]...)
		b = append(b, hdr...)
	}
	b = append(b, msg...)
	if hh != nil {
		hh.Reset()
		hh.Write(hb[4:20])
		hh.Write([]byte(subj))
		if len(hdr) > 0 {
			hh.Write(hdr)
		}
		hh.Write(msg)
		b = append(b, hh.Sum(nil)...)
	} else {
		b = append(b, make([]byte, checksumSize)...)
	}
	return b
}

// TestDetail01: record layout — a well-formed record decodes all fields and
// the record fits the documented size accounting.
func TestDetail01(t *testing.T) {
	hh := mrBbDigest()
	mb := &msgBlock{}
	rec := mrBbRec(hh, 42, 777, "foo", nil, []byte("hello"))
	if uint64(len(rec)) != fileStoreMsgSizeRaw(3, 0, 5) {
		t.Fatalf("record len %d != size formula %d", len(rec), fileStoreMsgSizeRaw(3, 0, 5))
	}
	sm, err := mb.msgFromBuf(rec, nil, hh)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if sm.subj != "foo" || string(sm.msg) != "hello" || sm.seq != 42 || sm.ts != 777 {
		t.Fatalf("decoded = %+v", sm)
	}
}

// TestDetail02: sanity gate — undersized rl, oversized rl, truncated buffer,
// and impossible subject length all yield errBadMsg.
func TestDetail02(t *testing.T) {
	hh := mrBbDigest()
	mb := &msgBlock{}
	good := mrBbRec(hh, 1, 1, "foo", nil, []byte("hello"))

	var bm errBadMsg
	check := func(b []byte, what string) {
		_, err := mb.msgFromBuf(b, nil, nil)
		if err == nil || !errors.As(err, &bm) {
			t.Fatalf("%s: want errBadMsg, got %v", what, err)
		}
	}
	check(good[:len(good)-2], "truncated buffer (rl > len(buf))")
	big := append([]byte{}, good...)
	binary.LittleEndian.PutUint32(big[0:], rlBadThresh+1)
	check(big, "rl > 32MB")
	small := append([]byte{}, good...)
	binary.LittleEndian.PutUint32(small[0:], 10)
	check(small, "rl < header (dlen < 0)")
	badslen := append([]byte{}, good...)
	binary.LittleEndian.PutUint16(badslen[20:], 500)
	check(badslen, "slen beyond record")
}

// TestDetail03: checksum covers header[4:20]+subject+payload; a mismatch is an
// errBadMsg identifying the checksum; a correct hash decodes.
func TestDetail03(t *testing.T) {
	hh := mrBbDigest()
	mb := &msgBlock{}
	rec := mrBbRec(hh, 9, 9, "s", nil, []byte("m"))
	if _, err := mb.msgFromBuf(rec, nil, hh); err != nil {
		t.Fatalf("valid record with checksum: %v", err)
	}
	bad := append([]byte{}, rec...)
	bad[len(bad)-1] ^= 0xff
	_, err := mb.msgFromBuf(bad, nil, hh)
	var bm errBadMsg
	if err == nil || !errors.As(err, &bm) || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("checksum mismatch should be an errBadMsg naming the checksum, got %v", err)
	}
	// hh == nil skips checksum verification entirely.
	zeroed := append([]byte{}, rec...)
	for i := len(zeroed) - 8; i < len(zeroed); i++ {
		zeroed[i] = 0
	}
	if _, err := mb.msgFromBuf(zeroed, nil, nil); err != nil {
		t.Fatalf("nil hash should skip checksum: %v", err)
	}
}

// TestDetail04: ebit on seq decodes as an erased record (seq=0) while ts is
// still decoded.
func TestDetail04(t *testing.T) {
	hh := mrBbDigest()
	mb := &msgBlock{}
	rec := mrBbRec(hh, 42|ebit, 777, "foo", nil, []byte("x"))
	sm, err := mb.msgFromBuf(rec, nil, hh)
	if err != nil {
		t.Fatal(err)
	}
	if sm.seq != 0 {
		t.Fatalf("erased record seq = %d, want 0", sm.seq)
	}
	if sm.ts != 777 {
		t.Fatalf("erased record ts = %d, want 777", sm.ts)
	}
}

// TestDetail05: headered slicing — sm.buf covers hdr+msg, sm.hdr is
// capacity-limited, sm.msg is the remainder; unheadered sm.msg is the data
// after the subject.
func TestDetail05(t *testing.T) {
	hh := mrBbDigest()
	mb := &msgBlock{}
	hdrBytes := []byte("NATS/1.0\r\nK: v\r\n\r\n")
	rec := mrBbRec(hh, 7, 1, "s", hdrBytes, []byte("m"))
	sm, err := mb.msgFromBuf(rec, nil, hh)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sm.hdr, hdrBytes) {
		t.Fatalf("sm.hdr = %q", sm.hdr)
	}
	if cap(sm.hdr) != len(sm.hdr) {
		t.Fatalf("sm.hdr must be capacity-limited: cap=%d len=%d", cap(sm.hdr), len(sm.hdr))
	}
	if string(sm.msg) != "m" {
		t.Fatalf("sm.msg = %q", sm.msg)
	}
	if !bytes.Equal(sm.buf, append(append([]byte{}, hdrBytes...), 'm')) {
		t.Fatalf("sm.buf should cover hdr+msg, got %q", sm.buf)
	}
}

// TestDetail06: doCopy=false aliases the record buffer; doCopy=true detaches;
// a passed StoreMsg is reused.
func TestDetail06(t *testing.T) {
	hh := mrBbDigest()
	mb := &msgBlock{}

	recN := mrBbRec(hh, 1, 1, "s", nil, []byte("mm"))
	smN, err := mb.msgFromBufNoCopy(recN, nil, hh)
	if err != nil {
		t.Fatal(err)
	}
	recN[len(recN)-9] = 'Z' // last byte of "mm" payload
	if string(smN.msg) != "mZ" {
		t.Fatalf("no-copy decode must alias the buffer, msg=%q", smN.msg)
	}

	recC := mrBbRec(hh, 1, 1, "s", nil, []byte("mm"))
	smC, err := mb.msgFromBuf(recC, nil, hh)
	if err != nil {
		t.Fatal(err)
	}
	recC[len(recC)-9] = 'Z'
	if string(smC.msg) != "mm" {
		t.Fatalf("copy decode must detach, msg=%q", smC.msg)
	}

	old := &StoreMsg{subj: "x"}
	sm2, err := mb.msgFromBuf(mrBbRec(hh, 2, 2, "t", nil, []byte("y")), old, hh)
	if err != nil {
		t.Fatal(err)
	}
	if sm2 != old || sm2.subj != "t" {
		t.Fatalf("passed StoreMsg should be reused, ptr-eq=%v subj=%q", sm2 == old, sm2.subj)
	}
}

// TestDetail07: size accounting — raw = 22+slen+mlen+8 (+4+hlen when headers);
// isFileStoreMsgTooLarge rejects an rl carrying hbit or exceeding 32MB.
func TestDetail07(t *testing.T) {
	if fileStoreMsgSizeRaw(3, 0, 5) != 22+3+5+8 {
		t.Fatalf("sizeRaw = %d", fileStoreMsgSizeRaw(3, 0, 5))
	}
	if fileStoreMsgSizeRaw(3, 4, 5) != 22+3+5+8+4+4 {
		t.Fatalf("headered sizeRaw = %d", fileStoreMsgSizeRaw(3, 4, 5))
	}
	if fileStoreMsgSize("abc", nil, []byte("12345")) != 38 {
		t.Fatalf("fileStoreMsgSize = %d", fileStoreMsgSize("abc", nil, []byte("12345")))
	}
	if fileStoreMsgSize("abc", []byte("hh"), []byte("12345")) != 44 {
		t.Fatalf("headered fileStoreMsgSize = %d", fileStoreMsgSize("abc", []byte("hh"), []byte("12345")))
	}
	if fileStoreMsgSizeEstimate(3, 100) < fileStoreMsgSizeRaw(3, 0, 100) {
		t.Fatal("estimate must cover the raw size")
	}
	if !isFileStoreMsgTooLarge(hbit | 50) {
		t.Fatal("rl with hbit set must be too large")
	}
	if !isFileStoreMsgTooLarge(rlBadThresh + 1) {
		t.Fatal("rl > 32MB must be too large")
	}
	if isFileStoreMsgTooLarge(rlBadThresh) || isFileStoreMsgTooLarge(64) {
		t.Fatal("in-range rl must be accepted")
	}
}

// TestDetail08: errBadMsg.Error() prints the basename plus an optional
// ': detail' — shape only.
func TestDetail08(t *testing.T) {
	e1 := errBadMsg{fn: "/path/to/file.blk", detail: "invalid checksum"}
	s := e1.Error()
	if !strings.Contains(s, "file.blk") || strings.Contains(s, "/path/to") {
		t.Fatalf("error should print the basename only: %q", s)
	}
	if !strings.Contains(s, "invalid checksum") {
		t.Fatalf("error should include the detail: %q", s)
	}
	e2 := errBadMsg{fn: "/path/to/file.blk"}
	if s2 := e2.Error(); !strings.Contains(s2, "file.blk") || strings.HasSuffix(s2, ":") {
		t.Fatalf("no-detail error = %q", s2)
	}
}
