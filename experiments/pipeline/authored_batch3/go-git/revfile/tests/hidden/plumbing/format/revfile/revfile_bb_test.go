package revfile

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/idxfile"
)

var (
	rvH1, _   = plumbing.FromHex("1111111111111111111111111111111111111111")
	rvH2, _   = plumbing.FromHex("2222222222222222222222222222222222222222")
	rvPack, _ = plumbing.FromHex("dddddddddddddddddddddddddddddddddddddddd")
)

// buildIdx makes a 2-entry MemoryIndex: h1 at pack offset 200 (index pos 0),
// h2 at offset 50 (index pos 1).
func buildIdx() *idxfile.MemoryIndex {
	idx := idxfile.NewMemoryIndex(20)
	idx.Names = [][]byte{rvH1.Bytes(), rvH2.Bytes()}
	for i := range idx.FanoutMapping {
		idx.FanoutMapping[i] = -1 // noMapping
	}
	idx.FanoutMapping[0x11] = 0
	idx.FanoutMapping[0x22] = 1
	for b := 0x11; b <= 0xff; b++ {
		idx.Fanout[b] = 1
		if b >= 0x22 {
			idx.Fanout[b] = 2
		}
	}
	be := func(v uint64) []byte {
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(v))
		return b[:]
	}
	idx.Offset32 = [][]byte{be(200), be(50)}
	idx.CRC32 = [][]byte{be(0), be(0)}
	idx.PackfileChecksum = rvPack
	return idx
}

// craftRev builds a .rev file by hand for the sha1 hash function.
func craftRev(entries []uint32, pack []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("RIDX")
	binary.Write(&buf, binary.BigEndian, uint32(1)) // version
	binary.Write(&buf, binary.BigEndian, uint32(1)) // sha1
	for _, e := range entries {
		binary.Write(&buf, binary.BigEndian, e)
	}
	buf.Write(pack)
	sum := sha1.Sum(buf.Bytes())
	buf.Write(sum[:])
	return buf.Bytes()
}

// drain collects channel values until close, bounded so a decoder that
// fails to close the channel fails fast instead of hanging the suite.
func drain(t *testing.T, ch <-chan uint32) []uint32 {
	t.Helper()
	done := make(chan []uint32, 1)
	go func() {
		var out []uint32
		for v := range ch {
			out = append(out, v)
		}
		done <- out
	}()
	select {
	case got := <-done:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("output channel never closed")
		return nil
	}
}

// TestDetail01: layout RIDX|version|hashfn|entries|packsum|checksum; only
// version 1 and hash ids 1/2 accepted.
func TestDetail01(t *testing.T) {
	good := craftRev([]uint32{1, 0}, rvPack.Bytes())
	out := make(chan uint32, 8)
	if err := Decode(bytes.NewReader(good), 2, rvPack, out); err != nil {
		t.Fatalf("valid rev file: %v", err)
	}
	if got := drain(t, out); len(got) != 2 {
		t.Fatalf("entries = %v", got)
	}

	// Bad version.
	bad := bytes.Clone(good)
	bad[7] = 9
	out = make(chan uint32, 8)
	if err := Decode(bytes.NewReader(bad), 2, rvPack, out); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("version 9 = %v", err)
	}
	drain(t, out)

	// Unsupported hash function id.
	bad = bytes.Clone(good)
	binary.BigEndian.PutUint32(bad[8:12], 7)
	out = make(chan uint32, 8)
	if err := Decode(bytes.NewReader(bad), 2, rvPack, out); !errors.Is(err, ErrUnsupportedHashFunction) {
		t.Fatalf("hash id 7 = %v", err)
	}
	drain(t, out)
}

// TestDetail02: entries stream in file order; the channel is closed when
// decode finishes — including on failure.
func TestDetail02(t *testing.T) {
	good := craftRev([]uint32{3, 1, 2}, rvPack.Bytes())
	out := make(chan uint32, 8)
	if err := Decode(bytes.NewReader(good), 3, rvPack, out); err != nil {
		t.Fatal(err)
	}
	got := drain(t, out) // drains until close
	want := []uint32{3, 1, 2}
	if len(got) != 3 || got[0] != 3 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("order = %v, want %v", got, want)
	}

	// Failure still closes the channel.
	out = make(chan uint32, 8)
	_ = Decode(bytes.NewReader([]byte("junk")), 3, rvPack, out)
	select {
	case _, ok := <-out:
		if ok {
			drain(t, out)
		}
	default:
		t.Fatal("channel left open on failure")
	}
}

// TestDetail03: the hash-function id drives checksum width and algorithm —
// a sha256 rev file carries 32-byte checksums.
func TestDetail03(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("RIDX")
	binary.Write(&buf, binary.BigEndian, uint32(1))
	binary.Write(&buf, binary.BigEndian, uint32(2)) // sha256
	binary.Write(&buf, binary.BigEndian, uint32(0))
	pack256 := bytes.Repeat([]byte{0xab}, 32)
	buf.Write(pack256)
	sum := sha256.Sum256(buf.Bytes())
	buf.Write(sum[:])

	packID, _ := plumbing.FromHex("abababababababababababababababababababababababababababababababab")
	out := make(chan uint32, 4)
	if err := Decode(bytes.NewReader(buf.Bytes()), 1, packID, out); err != nil {
		t.Fatalf("sha256 rev file: %v", err)
	}
	drain(t, out)
}

// TestDetail04 (shape — Inferable: no): objCount=0 is a dedicated failure
// before any entry read — an error surfaces immediately.
func TestDetail04(t *testing.T) {
	good := craftRev(nil, rvPack.Bytes())
	out := make(chan uint32, 4)
	err := Decode(bytes.NewReader(good), 0, rvPack, out)
	if !errors.Is(err, ErrEmptyReverseIndex) {
		t.Fatalf("objCount=0 = %v, want ErrEmptyReverseIndex", err)
	}
	drain(t, out)
}

// TestDetail05: the stored pack checksum must equal the caller's, exactly
// one hash-length.
func TestDetail05(t *testing.T) {
	// Mismatched checksum value.
	other, _ := plumbing.FromHex("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	good := craftRev([]uint32{0}, rvPack.Bytes())
	out := make(chan uint32, 4)
	if err := Decode(bytes.NewReader(good), 1, other, out); !errors.Is(err, ErrMalformedRevFile) {
		t.Fatalf("checksum mismatch = %v", err)
	}
	drain(t, out)
}

// TestDetail06 (shape — Inferable: no): bytes after the trailing checksum
// are malformed — the reader demands EOF.
func TestDetail06(t *testing.T) {
	good := craftRev([]uint32{0}, rvPack.Bytes())
	trailing := append(bytes.Clone(good), 0x00, 0x01)
	out := make(chan uint32, 4)
	if err := Decode(bytes.NewReader(trailing), 1, rvPack, out); err == nil {
		t.Fatal("trailing bytes accepted")
	}
	drain(t, out)
}

// TestDetail07: the running checksum is verified — a bit-flip anywhere is
// caught.
func TestDetail07(t *testing.T) {
	good := craftRev([]uint32{0}, rvPack.Bytes())
	for _, pos := range []int{0, 10, 13, 20} {
		bad := bytes.Clone(good)
		bad[pos] ^= 0xff
		out := make(chan uint32, 4)
		err := Decode(bytes.NewReader(bad), 1, rvPack, out)
		if err == nil {
			t.Fatalf("bit-flip at %d not caught", pos)
		}
		drain(t, out)
	}
}

// TestDetail08 (shape — Inferable: no): Encode rejects nil and typed-nil
// writers — an error, never a panic or silent write.
func TestDetail08(t *testing.T) {
	idx := buildIdx()
	if err := Encode(nil, sha1.New(), idx); err == nil {
		t.Fatal("nil writer accepted")
	}
	var nilBuf *bytes.Buffer
	var typedNil io.Writer = nilBuf
	if err := Encode(typedNil, sha1.New(), idx); err == nil {
		t.Fatal("typed-nil writer accepted")
	}
}

// TestDetail09 (shape — Inferable: no): the hash-function id comes from the
// hasher's size — a 32-byte hasher selects SHA-256 (id 2), others SHA-1 (1).
func TestDetail09(t *testing.T) {
	idx := buildIdx()
	var buf bytes.Buffer
	if err := Encode(&buf, sha1.New(), idx); err != nil {
		t.Fatal(err)
	}
	if got := binary.BigEndian.Uint32(buf.Bytes()[8:12]); got != 1 {
		t.Fatalf("sha1 hasher produced hash-fn id %d", got)
	}
	buf.Reset()
	if err := Encode(&buf, sha256.New(), idx); err != nil {
		t.Fatal(err)
	}
	if got := binary.BigEndian.Uint32(buf.Bytes()[8:12]); got != 2 {
		t.Fatalf("sha256 hasher produced hash-fn id %d", got)
	}
}

// TestDetail10 (shape — Inferable: no): a dirty hasher is reset — output is
// decodable regardless of carried-in state.
func TestDetail10(t *testing.T) {
	idx := buildIdx()
	h := sha1.New()
	h.Write([]byte("dirty state"))
	var buf bytes.Buffer
	if err := Encode(&buf, h, idx); err != nil {
		t.Fatal(err)
	}
	out := make(chan uint32, 4)
	if err := Decode(bytes.NewReader(buf.Bytes()), 2, rvPack, out); err != nil {
		t.Fatalf("encode with dirty hasher produced undecodable file: %v", err)
	}
	drain(t, out)
}

// TestDetail11: the reverse index maps pack-offset order to index position.
func TestDetail11(t *testing.T) {
	idx := buildIdx()
	var buf bytes.Buffer
	if err := Encode(&buf, sha1.New(), idx); err != nil {
		t.Fatal(err)
	}
	out := make(chan uint32, 4)
	if err := Decode(bytes.NewReader(buf.Bytes()), 2, rvPack, out); err != nil {
		t.Fatal(err)
	}
	got := drain(t, out)
	// Pack-offset order: h2(50) then h1(200); index positions: h2→1, h1→0.
	if len(got) != 2 || got[0] != 1 || got[1] != 0 {
		t.Fatalf("rev entries = %v, want [1 0]", got)
	}
}

// TestDetail12 (shape — Inferable: no): a short read of the pack checksum is
// malformed even when the bytes match the prefix.
func TestDetail12(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("RIDX")
	binary.Write(&buf, binary.BigEndian, uint32(1))
	binary.Write(&buf, binary.BigEndian, uint32(1))
	binary.Write(&buf, binary.BigEndian, uint32(0))
	buf.Write(rvPack.Bytes()[:10]) // half the checksum
	sum := sha1.Sum(buf.Bytes())
	buf.Write(sum[:])
	out := make(chan uint32, 4)
	if err := Decode(bytes.NewReader(buf.Bytes()), 1, rvPack, out); err == nil {
		t.Fatal("short pack checksum accepted")
	}
	drain(t, out)
}
