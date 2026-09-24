package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/minio/highwayhash"
)

// TestDetail01: appendEntry layout — leader[idLen], u64 term/commit/pterm/
// pindex, u16 count, per entry u32(1+len(data))+type+data, then a uvarint
// lterm tail.
func TestDetail01(t *testing.T) {
	ae := &appendEntry{leader: "LEADERID", term: 2, commit: 3, pterm: 1, pindex: 4,
		entries: []*Entry{
			{Type: EntryNormal, Data: []byte("hello")},
			{Type: EntryAddPeer, Data: []byte("12345678")},
		}}
	b, err := ae.encode(nil)
	if err != nil {
		t.Fatal(err)
	}
	le := binary.LittleEndian
	if string(b[:8]) != "LEADERID" {
		t.Fatalf("leader id at head: %q", b[:8])
	}
	if le.Uint64(b[8:]) != 2 || le.Uint64(b[16:]) != 3 || le.Uint64(b[24:]) != 1 || le.Uint64(b[32:]) != 4 {
		t.Fatalf("u64 header fields wrong: %x", b[:40])
	}
	if le.Uint16(b[40:]) != 2 {
		t.Fatalf("entry count: %x", b[40:42])
	}
	// Entry 0: u32(1+5)=6, type byte 0, "hello".
	if le.Uint32(b[42:]) != 6 || b[46] != byte(EntryNormal) || string(b[47:52]) != "hello" {
		t.Fatalf("entry0 framing: %x", b[42:52])
	}
	// Entry 1: u32(1+8)=9, type byte AddPeer, "12345678".
	if le.Uint32(b[52:]) != 9 || b[56] != byte(EntryAddPeer) || string(b[57:65]) != "12345678" {
		t.Fatalf("entry1 framing: %x", b[52:65])
	}
	// Trailing uvarint lterm: zero here encodes as a single 0 byte.
	if len(b) != 66 || b[65] != 0 {
		t.Fatalf("lterm tail: %x", b[64:])
	}
}

// TestDetail02: encode rejects a leader of the wrong length, more than
// 65535 entries, and over-large entry data.
func TestDetail02(t *testing.T) {
	if _, err := (&appendEntry{leader: "X"}).encode(nil); err != errLeaderLen {
		t.Fatalf("bad leader len: %v", err)
	}
	many := &appendEntry{leader: "LEADERID", entries: make([]*Entry, 65536)}
	for i := range many.entries {
		many.entries[i] = &Entry{Type: EntryNormal}
	}
	if _, err := many.encode(nil); err != errTooManyEntries {
		t.Fatalf("65536 entries: %v", err)
	}
}

// TestDetail03: encode reuses a caller buffer with sufficient capacity.
func TestDetail03(t *testing.T) {
	ae := &appendEntry{leader: "LEADERID", term: 1, entries: []*Entry{{Type: EntryNormal, Data: []byte("x")}}}
	buf := make([]byte, 0, 1024)
	out, err := ae.encode(buf)
	if err != nil {
		t.Fatal(err)
	}
	if &out[:1][0] != &buf[:1][0] {
		t.Fatal("encode should reuse a sufficiently-large caller buffer")
	}
	exact, err := ae.encode(nil)
	if err != nil || len(exact) != len(out) {
		t.Fatalf("encoded size mismatch: %d vs %d", len(exact), len(out))
	}
}

// TestDetail04: decodeAppendEntry requires the base length; each entry
// needs its 4 length bytes, a positive length, and enough buffer; Data
// slices msg and ae.buf retains it.
func TestDetail04(t *testing.T) {
	if _, err := decodeAppendEntry(make([]byte, appendEntryBaseLen-1), nil, ""); err != errBadAppendEntry {
		t.Fatalf("short msg: %v", err)
	}
	ae := &appendEntry{leader: "LEADERID", entries: []*Entry{{Type: EntryNormal, Data: []byte("hi")}}}
	b, _ := ae.encode(nil)
	got, err := decodeAppendEntry(b, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if string(got.entries[0].Data) != "hi" {
		t.Fatalf("entry data = %q", got.entries[0].Data)
	}
	// Entry data aliases the wire buffer.
	dataOff := appendEntryBaseLen + 4 + 1
	b[dataOff] = 'X'
	if string(got.entries[0].Data) != "Xi" {
		t.Fatalf("entry Data must alias msg, got %q", got.entries[0].Data)
	}
	if &got.buf[:1][0] != &b[:1][0] {
		t.Fatal("ae.buf must retain msg")
	}
	// Zero-length entry is corrupt.
	zl := append([]byte{}, b...)
	binary.LittleEndian.PutUint32(zl[appendEntryBaseLen:], 0)
	if _, err := decodeAppendEntry(zl, nil, ""); err != errBadAppendEntry {
		t.Fatalf("zero-length entry: %v", err)
	}
	// Declared length overruns the buffer.
	ov := append([]byte{}, b...)
	binary.LittleEndian.PutUint32(ov[appendEntryBaseLen:], 999)
	if _, err := decodeAppendEntry(ov, nil, ""); err != errBadAppendEntry {
		t.Fatalf("overrun entry: %v", err)
	}
}

// TestDetail05: a trailing uvarint after the last entry decodes as lterm;
// a truncated tail leaves lterm 0.
func TestDetail05(t *testing.T) {
	ae := &appendEntry{leader: "LEADERID", term: 2, lterm: 300,
		entries: []*Entry{{Type: EntryNormal, Data: []byte("x")}}}
	b, _ := ae.encode(nil)
	got, err := decodeAppendEntry(b, nil, "")
	if err != nil || got.lterm != 300 {
		t.Fatalf("lterm = %d err=%v", got.lterm, err)
	}
	// Truncate the uvarint tail: 300 encodes as 2 bytes (0xAC 0x02); cut the
	// second so the tail is an incomplete uvarint.
	trunc := b[:len(b)-1]
	got2, err := decodeAppendEntry(trunc, nil, "")
	if err != nil || got2.lterm != 0 {
		t.Fatalf("truncated tail: lterm=%d err=%v", got2.lterm, err)
	}
}

// TestDetail06: appendEntryResponse is fixed 25 bytes; peer occupies
// 16..24 (truncated or zero-padded), byte 24 is the success flag; decode
// returns nil unless len == 25.
func TestDetail06(t *testing.T) {
	ar := &appendEntryResponse{term: 9, index: 8, peer: "PEERID12", success: true}
	b := ar.encode(nil)
	if len(b) != appendEntryResponseLen || appendEntryResponseLen != 25 {
		t.Fatalf("len = %d", len(b))
	}
	if string(b[16:24]) != "PEERID12" || b[24] != 1 {
		t.Fatalf("peer/success bytes: %x", b[16:25])
	}
	got := decodeAppendEntryResponse(b)
	if got == nil || got.term != 9 || got.index != 8 || got.peer != "PEERID12" || !got.success {
		t.Fatalf("decoded = %+v", got)
	}
	// Over-long peer truncates; short peer zero-pads; failure byte is 0.
	ar2 := &appendEntryResponse{term: 1, index: 2, peer: "TOOLONGPEERID99", success: false}
	b2 := ar2.encode(nil)
	if string(b2[16:24]) != "TOOLONGP" || b2[24] != 0 {
		t.Fatalf("truncated peer / failure byte: %x", b2[16:25])
	}
	ar3 := &appendEntryResponse{peer: "AB", success: true}
	b3 := ar3.encode(nil)
	if string(b3[16:18]) != "AB" || !bytes.Equal(b3[18:24], make([]byte, 6)) {
		t.Fatalf("short peer should zero-pad: %x", b3[16:24])
	}
	if decodeAppendEntryResponse(b[:24]) != nil || decodeAppendEntryResponse(append(b, 0)) != nil {
		t.Fatal("decode must return nil unless len == 25")
	}
}

// TestDetail07: peerState wire form — u32 clusterSize, u32 count, count*8
// id bytes, optional u16 domainExt; decode errors on short buffers or
// missing ids.
func TestDetail07(t *testing.T) {
	ps := &peerState{knownPeers: []string{"ABCDEFGH", "IJKLMNOP"}, clusterSize: 3}
	b := encodePeerState(ps)
	if len(b) != peerStateBufSize(ps) || len(b) < 8+2*idLen {
		t.Fatalf("len=%d bufsize=%d", len(b), peerStateBufSize(ps))
	}
	got, err := decodePeerState(b)
	if err != nil || got.clusterSize != 3 || len(got.knownPeers) != 2 ||
		got.knownPeers[0] != "ABCDEFGH" || got.knownPeers[1] != "IJKLMNOP" {
		t.Fatalf("decoded = %+v err=%v", got, err)
	}
	// With a domain extension the trailing u16 round-trips.
	psx := &peerState{knownPeers: []string{"ABCDEFGH"}, clusterSize: 3, domainExt: extExtended}
	bx := encodePeerState(psx)
	gx, err := decodePeerState(bx)
	if err != nil || gx.domainExt != extExtended {
		t.Fatalf("domainExt = %v err=%v", gx.domainExt, err)
	}
	if _, err := decodePeerState([]byte{1, 2, 3}); err != errCorruptPeers {
		t.Fatalf("short buf: %v", err)
	}
	// Declared peer count without the id bytes.
	bad := []byte{3, 0, 0, 0, 5, 0, 0, 0}
	if _, err := decodePeerState(bad); err != errCorruptPeers {
		t.Fatalf("missing ids: %v", err)
	}
}

// TestDetail08: voteRequest is a fixed 32-byte record; decode returns nil
// unless len == 32 and copies the reply subject.
func TestDetail08(t *testing.T) {
	vr := &voteRequest{term: 4, lastTerm: 5, lastIndex: 6, candidate: "CANDID88"}
	b := vr.encode()
	if len(b) != voteRequestLen || voteRequestLen != 32 {
		t.Fatalf("len = %d", len(b))
	}
	got := decodeVoteRequest(b, "reply.subj")
	if got == nil || got.term != 4 || got.lastTerm != 5 || got.lastIndex != 6 ||
		got.candidate != "CANDID88" || got.reply != "reply.subj" {
		t.Fatalf("decoded = %+v", got)
	}
	if decodeVoteRequest(b[:31], "") != nil || decodeVoteRequest(append(b, 0), "") != nil {
		t.Fatal("decode must return nil unless len == 32")
	}
}

// TestDetail09: encodeSnapshot(nil) → nil; layout is lastTerm, lastIndex,
// u32 peerstate-len, peerstate, data, then an 8-byte highwayhash-64
// checksum over all preceding bytes.
func TestDetail09(t *testing.T) {
	n := &raft{}
	key := sha256.Sum256([]byte("grp"))
	n.hh, _ = highwayhash.NewDigest64(key[:])
	if out := n.encodeSnapshot(nil); out != nil {
		t.Fatalf("nil snapshot should encode to nil, got %x", out)
	}
	out := n.encodeSnapshot(&snapshot{lastTerm: 3, lastIndex: 9, peerstate: []byte{1, 2}, data: []byte("d")})
	le := binary.LittleEndian
	if len(out) != 8+8+4+2+1+8 {
		t.Fatalf("len = %d", len(out))
	}
	if le.Uint64(out[0:]) != 3 || le.Uint64(out[8:]) != 9 || le.Uint32(out[16:]) != 2 {
		t.Fatalf("header fields: %x", out)
	}
	if !bytes.Equal(out[20:22], []byte{1, 2}) || out[22] != 'd' {
		t.Fatalf("peerstate/data: %x", out[20:23])
	}
	n.hh.Reset()
	n.hh.Write(out[:len(out)-8])
	var hb [highwayhash.Size64]byte
	if !bytes.Equal(out[len(out)-8:], n.hh.Sum(hb[:0])) {
		t.Fatal("trailing checksum does not match a highwayhash-64 of the preceding bytes")
	}
}

// TestDetail10: termAndIndexFromSnapFile parses the basename strictly —
// the canonical round-trip must reproduce the basename, so padded or
// extended names fail.
func TestDetail10(t *testing.T) {
	term, index, err := termAndIndexFromSnapFile("/x/snap.1.2")
	if err != nil || term != 1 || index != 2 {
		t.Fatalf("snap.1.2: %d %d %v", term, index, err)
	}
	for _, s := range []string{"snap.01.2", "snap.1.2x", "snap.1.2.extra", "", "snap.1"} {
		if _, _, err := termAndIndexFromSnapFile(s); err == nil {
			t.Fatalf("%q should be rejected", s)
		}
	}
}
