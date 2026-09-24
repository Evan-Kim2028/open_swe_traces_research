package server

import (
	"testing"
)

// TestDetail01 (yes): an int64 input is returned unchanged with a nil error.
func TestDetail01(t *testing.T) {
	for _, v := range []int64{0, 1, 12345, -5, 1 << 50} {
		got, err := getStorageSize(v)
		if err != nil || got != v {
			t.Fatalf("getStorageSize(%d) = (%d, %v)", v, got, err)
		}
	}
}

// TestDetail02 (yes): any other non-string type is an error.
func TestDetail02(t *testing.T) {
	for _, v := range []any{int(5), int32(5), uint64(5), float64(1.5), true, []byte("1K")} {
		if _, err := getStorageSize(v); err == nil {
			t.Fatalf("getStorageSize(%T %v) returned nil error", v, v)
		}
	}
}

// TestDetail03 (shape — Inferable: no): an empty string returns (0, nil).
func TestDetail03(t *testing.T) {
	got, err := getStorageSize("")
	if err != nil || got != 0 {
		t.Fatalf("getStorageSize(\"\") = (%d, %v), want (0, nil)", got, err)
	}
}

// TestDetail04 (yes): a non-empty string is prefix + lastChar, the prefix
// parsed as a base-10 int64.
func TestDetail04(t *testing.T) {
	got, err := getStorageSize("7K")
	if err != nil || got != 7<<10 {
		t.Fatalf("getStorageSize(7K) = (%d, %v)", got, err)
	}
	if _, err := getStorageSize("-2K"); err != nil {
		t.Fatalf("getStorageSize(-2K) errored: %v (negative prefix parses)", err)
	}
}

// TestDetail05 (yes): a non-int64 prefix returns the parse error.
func TestDetail05(t *testing.T) {
	for _, v := range []string{"abcK", "12.5K", "K", " 1K", "0x10K"} {
		if _, err := getStorageSize(v); err == nil {
			t.Fatalf("getStorageSize(%q) returned nil error", v)
		}
	}
}

// TestDetail06 (partially): K multiplies by 1<<10.
func TestDetail06(t *testing.T) {
	got, err := getStorageSize("2K")
	if err != nil || got != 2<<10 {
		t.Fatalf("getStorageSize(2K) = (%d, %v), want %d", got, err, 2<<10)
	}
}

// TestDetail07 (partially): M multiplies by 1<<20.
func TestDetail07(t *testing.T) {
	got, err := getStorageSize("3M")
	if err != nil || got != 3<<20 {
		t.Fatalf("getStorageSize(3M) = (%d, %v), want %d", got, err, 3<<20)
	}
}

// TestDetail08 (partially): G multiplies by 1<<30.
func TestDetail08(t *testing.T) {
	got, err := getStorageSize("4G")
	if err != nil || got != 4<<30 {
		t.Fatalf("getStorageSize(4G) = (%d, %v), want %d", got, err, 4<<30)
	}
}

// TestDetail09 (partially): T multiplies by 1<<40.
func TestDetail09(t *testing.T) {
	got, err := getStorageSize("5T")
	if err != nil || got != 5<<40 {
		t.Fatalf("getStorageSize(5T) = (%d, %v), want %d", got, err, 5<<40)
	}
}

// TestDetail10 (yes): any other last character is an error.
func TestDetail10(t *testing.T) {
	for _, v := range []string{"1P", "1B", "1x", "1024", "1K ", "1"} {
		if _, err := getStorageSize(v); err == nil {
			t.Fatalf("getStorageSize(%q) returned nil error", v)
		}
	}
}

// TestDetail11 (shape — Inferable: no): suffix letters are case-sensitive;
// lowercase is rejected with an error.
func TestDetail11(t *testing.T) {
	for _, v := range []string{"1k", "1m", "1g", "1t"} {
		if _, err := getStorageSize(v); err == nil {
			t.Fatalf("getStorageSize(%q) accepted a lowercase suffix", v)
		}
	}
}

// TestDetail12 (yes): there is no space between the number and the suffix.
func TestDetail12(t *testing.T) {
	for _, v := range []string{"1 K", "1  K", "1024 "} {
		if _, err := getStorageSize(v); err == nil {
			t.Fatalf("getStorageSize(%q) accepted a spaced form", v)
		}
	}
}
