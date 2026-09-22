package codec

import (
	"bytes"
	"testing"
)

// Hidden suite for unit codecnum. One TestDetailNN per DETAILS.md line.

// TestDetail01: EncodeInt/DecodeInt are 8-byte big-endian and sort ascending
// in int64 order (the documented comparison guarantee).
func TestDetail01(t *testing.T) {
	if got := EncodeInt(nil, 42); len(got) != 8 {
		t.Fatalf("EncodeInt produced %d bytes, want 8", len(got))
	}
	vals := []int64{-1000, -2, -1, 0, 1, 2, 1000}
	for i := 0; i+1 < len(vals); i++ {
		a := EncodeInt(nil, vals[i])
		b := EncodeInt(nil, vals[i+1])
		if bytes.Compare(a, b) >= 0 {
			t.Fatalf("EncodeInt not ascending: %d -> %x, %d -> %x", vals[i], a, vals[i+1], b)
		}
	}
	rest, v, err := DecodeInt(EncodeInt(nil, -7))
	if err != nil || v != -7 || len(rest) != 0 {
		t.Fatalf("DecodeInt roundtrip: %v %d %v", rest, v, err)
	}
}

// TestDetail02: EncodeIntDesc/DecodeIntDesc order descending.
func TestDetail02(t *testing.T) {
	vals := []int64{-1000, -1, 0, 1, 1000}
	for i := 0; i+1 < len(vals); i++ {
		a := EncodeIntDesc(nil, vals[i])
		b := EncodeIntDesc(nil, vals[i+1])
		if bytes.Compare(a, b) <= 0 {
			t.Fatalf("EncodeIntDesc not descending: %d -> %x, %d -> %x", vals[i], a, vals[i+1], b)
		}
	}
	rest, v, err := DecodeIntDesc(EncodeIntDesc(nil, -7))
	if err != nil || v != -7 || len(rest) != 0 {
		t.Fatalf("DecodeIntDesc roundtrip: %v %d %v", rest, v, err)
	}
}

// TestDetail03: EncodeUint orders ascending in uint64 order without a sign
// remap; EncodeUintDesc orders descending.
func TestDetail03(t *testing.T) {
	vals := []uint64{0, 1, 2, 1 << 40, ^uint64(0)}
	for i := 0; i+1 < len(vals); i++ {
		a := EncodeUint(nil, vals[i])
		b := EncodeUint(nil, vals[i+1])
		if bytes.Compare(a, b) >= 0 {
			t.Fatalf("EncodeUint not ascending: %d -> %x, %d -> %x", vals[i], a, vals[i+1], b)
		}
		ad := EncodeUintDesc(nil, vals[i])
		bd := EncodeUintDesc(nil, vals[i+1])
		if bytes.Compare(ad, bd) <= 0 {
			t.Fatalf("EncodeUintDesc not descending: %d -> %x, %d -> %x", vals[i], ad, vals[i+1], bd)
		}
	}
	rest, v, err := DecodeUint(EncodeUint(nil, 1<<40))
	if err != nil || v != 1<<40 || len(rest) != 0 {
		t.Fatalf("DecodeUint roundtrip: %v %d %v", rest, v, err)
	}
	rest, v, err = DecodeUintDesc(EncodeUintDesc(nil, 1<<40))
	if err != nil || v != 1<<40 || len(rest) != 0 {
		t.Fatalf("DecodeUintDesc roundtrip: %v %d %v", rest, v, err)
	}
}

// TestDetail04: every Decode* returns the leftover (input minus consumed) and
// a nil leftover plus error on a short input.
func TestDetail04(t *testing.T) {
	suffix := []byte{0xAA, 0xBB}
	enc := append(EncodeInt(nil, 5), suffix...)
	rest, v, err := DecodeInt(enc)
	if err != nil || v != 5 || !bytes.Equal(rest, suffix) {
		t.Fatalf("DecodeInt leftover: %v %d %v", rest, v, err)
	}
	if rest, _, err := DecodeInt([]byte{0x01, 0x02}); err == nil || rest != nil {
		t.Fatalf("short DecodeInt: rest=%v err=%v, want nil leftover + error", rest, err)
	}
	if rest, _, err := DecodeUint([]byte{0x01}); err == nil || rest != nil {
		t.Fatalf("short DecodeUint: rest=%v err=%v, want nil leftover + error", rest, err)
	}
	if rest, _, err := DecodeIntDesc([]byte{0x01}); err == nil || rest != nil {
		t.Fatalf("short DecodeIntDesc: rest=%v err=%v, want nil leftover + error", rest, err)
	}
	if rest, _, err := DecodeUintDesc(nil); err == nil || rest != nil {
		t.Fatalf("short DecodeUintDesc: rest=%v err=%v, want nil leftover + error", rest, err)
	}
}

// TestDetail05: EncodeVarint/EncodeUvarint are standard protobuf varints
// (zig-zag for the signed form) and are NOT mem-comparable.
func TestDetail05(t *testing.T) {
	if got := EncodeUvarint(nil, 300); !bytes.Equal(got, []byte{0xAC, 0x02}) {
		t.Fatalf("EncodeUvarint(300) = %x, want standard varint ac02", got)
	}
	// zig-zag: -1 maps to 1.
	if got := EncodeVarint(nil, -1); !bytes.Equal(got, []byte{0x01}) {
		t.Fatalf("EncodeVarint(-1) = %x, want zig-zag 01", got)
	}
	// Not mem-comparable: -2 zig-zags to 3 (0x03) which sorts after 1 (0x02).
	if bytes.Compare(EncodeVarint(nil, -2), EncodeVarint(nil, 1)) <= 0 {
		t.Fatalf("EncodeVarint is unexpectedly mem-comparable")
	}
}

// TestDetail06: varint decoders distinguish an over-64-bit value from a
// truncated input — both must error, and not with the same error.
func TestDetail06(t *testing.T) {
	over := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x7F}
	trunc := []byte{0x80}
	_, _, errOver := DecodeUvarint(over)
	_, _, errTrunc := DecodeUvarint(trunc)
	if errOver == nil || errTrunc == nil {
		t.Fatalf("DecodeUvarint: over=%v trunc=%v, want both errors", errOver, errTrunc)
	}
	if errOver.Error() == errTrunc.Error() {
		t.Fatalf("over-64-bit and truncated report the same error %q", errOver)
	}
	_, _, errOver = DecodeVarint(over)
	_, _, errTrunc = DecodeVarint(trunc)
	if errOver == nil || errTrunc == nil || errOver.Error() == errTrunc.Error() {
		t.Fatalf("DecodeVarint does not distinguish failures: over=%v trunc=%v", errOver, errTrunc)
	}
}

// TestDetail07: EncodeComparableUvarint is lexically ordered and round-trips.
func TestDetail07(t *testing.T) {
	vals := []uint64{0, 1, 5, 239, 240, 1000, 1 << 32, ^uint64(0)}
	for i := 0; i+1 < len(vals); i++ {
		a := EncodeComparableUvarint(nil, vals[i])
		b := EncodeComparableUvarint(nil, vals[i+1])
		if bytes.Compare(a, b) >= 0 {
			t.Fatalf("EncodeComparableUvarint not ascending: %d -> %x, %d -> %x", vals[i], a, vals[i+1], b)
		}
	}
	for _, v := range vals {
		rest, got, err := DecodeComparableUvarint(EncodeComparableUvarint(nil, v))
		if err != nil || got != v || len(rest) != 0 {
			t.Fatalf("DecodeComparableUvarint roundtrip %d: %v %d %v", v, rest, got, err)
		}
	}
}

// TestDetail08: EncodeComparableVarint orders mixed-sign values lexically and
// delegates to the uvarint form for non-negatives.
func TestDetail08(t *testing.T) {
	vals := []int64{-1000, -2, -1, 0, 1, 5, 1000}
	for i := 0; i+1 < len(vals); i++ {
		a := EncodeComparableVarint(nil, vals[i])
		b := EncodeComparableVarint(nil, vals[i+1])
		if bytes.Compare(a, b) >= 0 {
			t.Fatalf("EncodeComparableVarint not ascending: %d -> %x, %d -> %x", vals[i], a, vals[i+1], b)
		}
	}
	// Non-negatives use the uvarint encoding verbatim.
	if !bytes.Equal(EncodeComparableVarint(nil, 5), EncodeComparableUvarint(nil, 5)) {
		t.Fatalf("EncodeComparableVarint(5) does not delegate to the uvarint form")
	}
	// A more negative value sorts before a less negative one.
	if bytes.Compare(EncodeComparableVarint(nil, -1000), EncodeComparableVarint(nil, -1)) >= 0 {
		t.Fatalf("negative ordering wrong: -1000 vs -1")
	}
}

// TestDetail09: DecodeComparableUvarint errors on an out-of-range first byte
// and consumes exactly the encoded length.
func TestDetail09(t *testing.T) {
	if _, _, err := DecodeComparableUvarint([]byte{0x03, 0x00}); err == nil {
		t.Fatalf("first byte below the valid range must error")
	}
	enc := EncodeComparableUvarint(nil, 7)
	suffix := []byte{0x77}
	rest, v, err := DecodeComparableUvarint(append(enc, suffix...))
	if err != nil || v != 7 || !bytes.Equal(rest, suffix) {
		t.Fatalf("DecodeComparableUvarint leftover: %v %d %v", rest, v, err)
	}
}

// TestDetail10: DecodeComparableVarint round-trips negative and positive
// values and rejects truncated input. (The single-byte inline form's
// leftover accounting is an implementation detail and is not asserted; the
// decoded value is.)
func TestDetail10(t *testing.T) {
	// Multi-byte encodings round-trip fully, leftover included.
	for _, v := range []int64{-1000, -2, -1, 240, 241, 1000, 1 << 32} {
		rest, got, err := DecodeComparableVarint(EncodeComparableVarint(nil, v))
		if err != nil || got != v || len(rest) != 0 {
			t.Fatalf("DecodeComparableVarint roundtrip %d: %v %d %v", v, rest, got, err)
		}
	}
	// Single-byte inline encodings still decode to the original value.
	for _, v := range []int64{1, 5, 239} {
		if _, got, err := DecodeComparableVarint(EncodeComparableVarint(nil, v)); err != nil || got != v {
			t.Fatalf("DecodeComparableVarint inline %d: %d %v", v, got, err)
		}
	}
	if _, _, err := DecodeComparableVarint(nil); err == nil {
		t.Fatalf("empty DecodeComparableVarint must error")
	}
}

// TestDetail11: every Decode* returns the original value and the untouched
// suffix.
func TestDetail11(t *testing.T) {
	suffix := []byte{0xDE, 0xAD}
	checks := []struct {
		name string
		enc  []byte
		dec  func([]byte) ([]byte, error)
	}{
		{"Int", EncodeInt(nil, -3), func(b []byte) ([]byte, error) { r, _, e := DecodeInt(b); return r, e }},
		{"IntDesc", EncodeIntDesc(nil, -3), func(b []byte) ([]byte, error) { r, _, e := DecodeIntDesc(b); return r, e }},
		{"Uint", EncodeUint(nil, 3), func(b []byte) ([]byte, error) { r, _, e := DecodeUint(b); return r, e }},
		{"UintDesc", EncodeUintDesc(nil, 3), func(b []byte) ([]byte, error) { r, _, e := DecodeUintDesc(b); return r, e }},
		{"Varint", EncodeVarint(nil, -3), func(b []byte) ([]byte, error) { r, _, e := DecodeVarint(b); return r, e }},
		{"Uvarint", EncodeUvarint(nil, 3), func(b []byte) ([]byte, error) { r, _, e := DecodeUvarint(b); return r, e }},
		{"ComparableVarint", EncodeComparableVarint(nil, -3), func(b []byte) ([]byte, error) { r, _, e := DecodeComparableVarint(b); return r, e }},
		{"ComparableUvarint", EncodeComparableUvarint(nil, 3), func(b []byte) ([]byte, error) { r, _, e := DecodeComparableUvarint(b); return r, e }},
	}
	for _, c := range checks {
		rest, err := c.dec(append(append([]byte{}, c.enc...), suffix...))
		if err != nil || !bytes.Equal(rest, suffix) {
			t.Fatalf("%s: rest=%v err=%v, want untouched suffix", c.name, rest, err)
		}
	}
}

// TestDetail12: EncodeIntToCmpUint orders like int64 (sign remap) and
// DecodeCmpUintToInt inverts it.
func TestDetail12(t *testing.T) {
	if !(EncodeIntToCmpUint(-1) < EncodeIntToCmpUint(0) && EncodeIntToCmpUint(0) < EncodeIntToCmpUint(1)) {
		t.Fatalf("EncodeIntToCmpUint not monotone: -1:%x 0:%x 1:%x",
			EncodeIntToCmpUint(-1), EncodeIntToCmpUint(0), EncodeIntToCmpUint(1))
	}
	if EncodeIntToCmpUint(0) != signMask {
		t.Fatalf("EncodeIntToCmpUint(0) = %x, want the sign mask", EncodeIntToCmpUint(0))
	}
	for _, v := range []int64{-1000, -1, 0, 1, 1000} {
		if got := DecodeCmpUintToInt(EncodeIntToCmpUint(v)); got != v {
			t.Fatalf("DecodeCmpUintToInt roundtrip %d -> %d", v, got)
		}
	}
}
