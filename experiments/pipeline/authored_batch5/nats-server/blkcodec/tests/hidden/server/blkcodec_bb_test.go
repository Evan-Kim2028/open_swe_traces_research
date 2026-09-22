package server

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestDetail01: MarshalMetadata emits "cmp" + algorithm byte + uvarint
// OriginalSize; UnmarshalMetadata round-trips it.
func TestDetail01(t *testing.T) {
	for _, size := range []uint64{0, 1, 300, 1 << 40} {
		ci := &CompressionInfo{Algorithm: S2Compression, OriginalSize: size}
		b := ci.MarshalMetadata()
		if len(b) < 5 || string(b[:3]) != "cmp" {
			t.Fatalf("metadata %q missing cmp magic", b)
		}
		if b[3] != byte(S2Compression) {
			t.Fatalf("alg byte = %d, want %d", b[3], S2Compression)
		}
		v, n := binary.Uvarint(b[4:])
		if n <= 0 || v != size || 4+n != len(b) {
			t.Fatalf("uvarint tail = v%d n%d len%d, want %d", v, n, len(b), size)
		}
		var back CompressionInfo
		m, err := back.UnmarshalMetadata(b)
		if err != nil || m != len(b) {
			t.Fatalf("round-trip: n=%d err=%v", m, err)
		}
		if back.Algorithm != S2Compression || back.OriginalSize != size {
			t.Fatalf("round-trip fields: %+v", back)
		}
	}
}

// TestDetail02: UnmarshalMetadata is a sniffer — short input, wrong magic,
// and non-S2 algorithm bytes all return (0, nil), never an error. Only a
// bad uvarint after a valid header errors.
func TestDetail02(t *testing.T) {
	var c CompressionInfo
	nonCompressed := [][]byte{
		nil,
		{1, 2, 3, 4},
		[]byte("xyz123456789"),
		{'c', 'm', 'p'},                    // magic, but < 5 bytes
		{'c', 'm', 'p', 0, 5},              // NoCompression alg byte
		{'c', 'm', 'p', 7, 5},              // unknown alg byte
		{'c', 'm', 'p', byte(S2Compression)}, // S2 but no uvarint room
	}
	for _, b := range nonCompressed {
		c.Algorithm, c.OriginalSize = 9, 9
		if n, err := c.UnmarshalMetadata(b); n != 0 || err != nil {
			t.Fatalf("sniff %q: n=%d err=%v, want (0,nil)", b, n, err)
		}
	}
	// A bad uvarint after a valid "cmp"+S2 header is the only error.
	bad := append([]byte{'c', 'm', 'p', byte(S2Compression)},
		bytes.Repeat([]byte{0x80}, 12)...)
	if n, err := c.UnmarshalMetadata(bad); err == nil || n != 0 {
		t.Fatalf("bad uvarint: n=%d err=%v, want error", n, err)
	}
}

// TestDetail03: Compress/Decompress preserve the last checksumSize bytes as
// a plaintext trailer; len(buf) < checksumSize errors; NoCompression is
// identity.
func TestDetail03(t *testing.T) {
	payload := make([]byte, 4096)
	for i := range payload {
		payload[i] = byte(i * 31)
	}
	out, err := S2Compression.Compress(payload)
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	if !bytes.Equal(out[len(out)-checksumSize:], payload[len(payload)-checksumSize:]) {
		t.Fatal("checksum trailer was not preserved verbatim")
	}
	rt, err := S2Compression.Decompress(out)
	if err != nil || !bytes.Equal(rt, payload) {
		t.Fatalf("round-trip: err=%v eq=%v", err, bytes.Equal(rt, payload))
	}
	if _, err := S2Compression.Compress(payload[:checksumSize-1]); err == nil {
		t.Fatal("Compress of short buffer did not error")
	}
	if _, err := S2Compression.Decompress(payload[:checksumSize-1]); err == nil {
		t.Fatal("Decompress of short buffer did not error")
	}
	id, err := NoCompression.Compress(payload)
	if err != nil || !bytes.Equal(id, payload) {
		t.Fatal("NoCompression.Compress is not identity")
	}
	id, err = NoCompression.Decompress(payload)
	if err != nil || !bytes.Equal(id, payload) {
		t.Fatal("NoCompression.Decompress is not identity")
	}
}

// TestDetail04: msgBlock.decode sniffs metadata — n==0 returns the buffer
// as NoCompression; a compressed block decompresses; a corrupt compressed
// block returns an error; and an uncompressed record whose length collides
// with the "cmp"+S2 header is still read correctly.
func TestDetail04(t *testing.T) {
	mb := &msgBlock{}
	plain := []byte("an uncompressed block buffer")
	d, alg, err := mb.decode(plain)
	if err != nil || alg != NoCompression || !bytes.Equal(d, plain) {
		t.Fatalf("plain decode: alg=%v err=%v", alg, err)
	}
	payload := bytes.Repeat([]byte("x"), 4096)
	ci := &CompressionInfo{Algorithm: S2Compression, OriginalSize: uint64(len(payload))}
	comp, err := S2Compression.Compress(payload)
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	buf := append(ci.MarshalMetadata(), comp...)
	d, alg, err = mb.decode(buf)
	if err != nil || alg != S2Compression || !bytes.Equal(d, payload) {
		t.Fatalf("compressed decode: alg=%v err=%v eq=%v", alg, err, bytes.Equal(d, payload))
	}
	// Corrupt compressed payload, small enough to skip the length fallback.
	bad := append((&CompressionInfo{Algorithm: S2Compression, OriginalSize: 100}).MarshalMetadata(),
		bytes.Repeat([]byte{0xAB}, 256)...)
	if _, _, err := mb.decode(bad); err == nil {
		t.Fatal("corrupt compressed block did not error")
	}

	// The collision case: an uncompressed record of exactly 24145251 bytes
	// begins with bytes equal to a valid "cmp"+S2 header.
	dir := t.TempDir()
	fs, err := newFileStore(
		FileStoreConfig{StoreDir: dir, Compression: NoCompression},
		StreamConfig{Name: "TEST", Storage: FileStorage})
	if err != nil {
		t.Fatalf("newFileStore: %v", err)
	}
	defer fs.Stop()
	subj := "test"
	msg := bytes.Repeat([]byte("a"), 24145251-int(fileStoreMsgSize(subj, nil, nil)))
	if _, _, err := fs.StoreMsg(subj, nil, msg, 0); err != nil {
		t.Fatalf("StoreMsg: %v", err)
	}
	if err := fs.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	fs, err = newFileStore(
		FileStoreConfig{StoreDir: dir, Compression: NoCompression},
		StreamConfig{Name: "TEST", Storage: FileStorage})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer fs.Stop()
	sm, err := fs.LoadMsg(1, nil)
	if err != nil {
		t.Fatalf("LoadMsg on collision-length record: %v", err)
	}
	if !bytes.Equal(sm.msg, msg) {
		t.Fatal("collision-length record did not round-trip")
	}
}

// TestDetail05: StoreCompression String/JSON codec — "None"/"S2"/"Unknown
// StoreCompression"; JSON lowercase "none"/"s2"; anything else errors in
// both directions.
func TestDetail05(t *testing.T) {
	if NoCompression.String() != "None" || S2Compression.String() != "S2" {
		t.Fatalf("String: %q %q", NoCompression, S2Compression)
	}
	if StoreCompression(77).String() != "Unknown StoreCompression" {
		t.Fatalf("unknown String: %q", StoreCompression(77))
	}
	for alg, want := range map[StoreCompression]string{NoCompression: `"none"`, S2Compression: `"s2"`} {
		j, err := alg.MarshalJSON()
		if err != nil || string(j) != want {
			t.Fatalf("MarshalJSON(%d) = %s, %v; want %s", alg, j, err, want)
		}
		var back StoreCompression
		if err := back.UnmarshalJSON(j); err != nil || back != alg {
			t.Fatalf("UnmarshalJSON(%s) = %d, %v", j, back, err)
		}
	}
	if _, err := StoreCompression(77).MarshalJSON(); err == nil {
		t.Fatal("MarshalJSON of unknown algorithm did not error")
	}
	var x StoreCompression
	if err := x.UnmarshalJSON([]byte(`"gzip"`)); err == nil {
		t.Fatal("UnmarshalJSON of unknown string did not error")
	}
}

// TestDetail06: codec failures surface as non-nil errors (shape only — the
// wrap/detail text is internal).
func TestDetail06(t *testing.T) {
	// A corrupt s2 stream errors from Decompress.
	if _, err := S2Compression.Decompress(bytes.Repeat([]byte{0xFE}, 64)); err == nil {
		t.Fatal("corrupt stream did not error")
	}
	// And well-formed input never errors.
	payload := bytes.Repeat([]byte("z"), 1024)
	out, err := S2Compression.Compress(payload)
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	if _, err := S2Compression.Decompress(out); err != nil {
		t.Fatalf("Decompress well-formed: %v", err)
	}
}
