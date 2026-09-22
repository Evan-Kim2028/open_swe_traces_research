package codec

import (
	"bytes"
	"testing"
)

// Hidden suite for unit codecbytes. One TestDetailNN per DETAILS.md line.
// The grouped-padding format literals below are the worked table in the
// solver-visible EncodeBytes doc comment.

// TestDetail01: [group][marker] pairs — 8-byte groups zero-padded on the
// right, marker = 0xFF - padCount.
func TestDetail01(t *testing.T) {
	got := EncodeBytes(nil, []byte{1, 2, 3})
	want := []byte{1, 2, 3, 0, 0, 0, 0, 0, 250}
	if !bytes.Equal(got, want) {
		t.Fatalf("EncodeBytes([1 2 3]) = %v, want %v", got, want)
	}
	got = EncodeBytes(nil, []byte{1, 2, 3, 0})
	want = []byte{1, 2, 3, 0, 0, 0, 0, 0, 251}
	if !bytes.Equal(got, want) {
		t.Fatalf("EncodeBytes([1 2 3 0]) = %v, want %v", got, want)
	}
}

// TestDetail02: empty input still emits one all-pad group.
func TestDetail02(t *testing.T) {
	got := EncodeBytes(nil, nil)
	want := []byte{0, 0, 0, 0, 0, 0, 0, 0, 247}
	if !bytes.Equal(got, want) {
		t.Fatalf("EncodeBytes(nil) = %v, want %v", got, want)
	}
}

// TestDetail03: an exact group multiple gets a full group (marker 0xFF) and a
// terminating all-pad group.
func TestDetail03(t *testing.T) {
	got := EncodeBytes(nil, []byte{1, 2, 3, 4, 5, 6, 7, 8})
	want := []byte{1, 2, 3, 4, 5, 6, 7, 8, 255, 0, 0, 0, 0, 0, 0, 0, 0, 247}
	if !bytes.Equal(got, want) {
		t.Fatalf("EncodeBytes(8 bytes) = %v, want %v", got, want)
	}
}

// TestDetail04: DecodeBytes stops at the first marker implying padding and
// returns the leftover input as its first result.
func TestDetail04(t *testing.T) {
	suffix := []byte{0xAA, 0xBB, 0xCC}
	enc := append(EncodeBytes(nil, []byte{1, 2, 3}), suffix...)
	rest, val, err := DecodeBytes(enc, nil)
	if err != nil {
		t.Fatalf("DecodeBytes: %v", err)
	}
	if !bytes.Equal(rest, suffix) {
		t.Fatalf("leftover = %v, want %v", rest, suffix)
	}
	if !bytes.Equal(val, []byte{1, 2, 3}) {
		t.Fatalf("value = %v, want [1 2 3]", val)
	}
}

// TestDetail05: decode errors on an impossible marker, a nonzero padding
// byte, and a truncated tail. (Shapes only — wording is unpinned.)
func TestDetail05(t *testing.T) {
	// Marker 0xF0 implies pad count 15 > 8: invalid marker.
	bad := []byte{1, 2, 3, 4, 5, 6, 7, 8, 0xF0}
	if _, _, err := DecodeBytes(bad, nil); err == nil {
		t.Fatalf("marker implying >8 pad bytes accepted")
	}
	// Padding byte not 0x00.
	bad = []byte{1, 2, 3, 9, 9, 9, 9, 9, 250}
	if _, _, err := DecodeBytes(bad, nil); err == nil {
		t.Fatalf("nonzero padding accepted")
	}
	// Truncated: a partial group with no marker.
	bad = []byte{1, 2, 3}
	if _, _, err := DecodeBytes(bad, nil); err == nil {
		t.Fatalf("truncated input accepted")
	}
	bad = []byte{1, 2, 3, 4, 5, 6, 7, 8}
	if _, _, err := DecodeBytes(bad, nil); err == nil {
		t.Fatalf("group without marker accepted")
	}
}

// TestDetail06: a non-nil buf is reused for the decoded output.
func TestDetail06(t *testing.T) {
	enc := EncodeBytes(nil, []byte{9, 8, 7})
	buf := make([]byte, 0, 128)
	_, dec, err := DecodeBytes(enc, buf)
	if err != nil || len(dec) == 0 {
		t.Fatalf("DecodeBytes with buf: %v %v", dec, err)
	}
	if &dec[0] != &buf[:1][0] {
		t.Fatalf("DecodeBytes did not reuse the caller buf")
	}
}

// TestDetail07: round-trip returns data verbatim and an empty leftover for
// lengths 0, 7, 8, 9, 16.
func TestDetail07(t *testing.T) {
	for _, n := range []int{0, 7, 8, 9, 16} {
		data := make([]byte, n)
		for i := range data {
			data[i] = byte(i + 1)
		}
		rest, val, err := DecodeBytes(EncodeBytes(nil, data), nil)
		if err != nil {
			t.Fatalf("len %d: %v", n, err)
		}
		if !bytes.Equal(val, data) || len(rest) != 0 {
			t.Fatalf("len %d roundtrip: val=%v rest=%v", n, val, rest)
		}
	}
}

// TestDetail08: reallocBytes preserves length and contents and makes room for
// n more bytes, reallocating only when the capacity is too small.
func TestDetail08(t *testing.T) {
	b := []byte{1, 2, 3, 4}
	r := reallocBytes(b, 4)
	if len(r) != 4 || !bytes.Equal(r, b) || cap(r) < 8 {
		t.Fatalf("reallocBytes len=%d cap=%d contents=%v", len(r), cap(r), r)
	}
	big := make([]byte, 4, 64)
	for i := range big {
		big[i] = byte(i)
	}
	r2 := reallocBytes(big, 4)
	if len(r2) != 4 || !bytes.Equal(r2, big) {
		t.Fatalf("reallocBytes on sufficient cap changed contents: %v", r2)
	}
	if &r2[0] != &big[0] {
		t.Fatalf("reallocBytes reallocated despite sufficient capacity")
	}
}
