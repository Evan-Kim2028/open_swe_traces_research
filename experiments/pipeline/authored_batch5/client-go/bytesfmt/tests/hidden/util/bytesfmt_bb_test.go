package util

import (
	"strings"
	"testing"
	"time"
)

// Hidden suite for unit bytesfmt. One TestDetailNN per DETAILS.md line.
// Exact unit spellings / decimal layouts are not derivable from the visible
// tree, so the asserted facts are relational: delegation, scaling, and the
// parse/fallback contract.

// TestDetail01: FormatBytes delegates to BytesToString at or below the Bytes
// threshold — FormatBytes(1024) shows the raw byte count, not a scaled unit.
func TestDetail01(t *testing.T) {
	if FormatBytes(1024) != BytesToString(1024) {
		t.Fatalf("FormatBytes(1024)=%q differs from BytesToString(1024)=%q", FormatBytes(1024), BytesToString(1024))
	}
	if FormatBytes(512) != BytesToString(512) {
		t.Fatalf("FormatBytes(512)=%q differs from BytesToString(512)=%q", FormatBytes(512), BytesToString(512))
	}
	if !strings.Contains(FormatBytes(1024), "1024") {
		t.Fatalf("FormatBytes(1024)=%q does not render the raw count", FormatBytes(1024))
	}
}

// TestDetail02: FormatBytes prunes precision — its output differs from the
// raw-float BytesToString rendering once a scaled unit applies, and the same
// input renders the same way twice.
func TestDetail02(t *testing.T) {
	if FormatBytes(1025) == BytesToString(1025) {
		t.Fatalf("FormatBytes(1025)=%q shows raw float; precision was not pruned", FormatBytes(1025))
	}
	// Scaled rendering: the KB-magnitude value must not print the raw count.
	if strings.Contains(FormatBytes(2048), "2048") {
		t.Fatalf("FormatBytes(2048)=%q still shows the raw count", FormatBytes(2048))
	}
	if !strings.Contains(FormatBytes(2048), "2") {
		t.Fatalf("FormatBytes(2048)=%q lost the scaled value", FormatBytes(2048))
	}
	// Larger inputs pick a larger unit: the renderings must differ.
	if FormatBytes(1<<20) == FormatBytes(1<<10) {
		t.Fatalf("FormatBytes does not scale units: %q == %q", FormatBytes(1<<20), FormatBytes(1<<10))
	}
}

// TestDetail03: BytesToString keeps small values in the Bytes unit (strict
// >1 comparison) and scales larger ones.
func TestDetail03(t *testing.T) {
	if !strings.Contains(BytesToString(1024), "1024") {
		t.Fatalf("BytesToString(1024)=%q does not stay in the Bytes unit", BytesToString(1024))
	}
	if strings.Contains(BytesToString(2048), "2048") {
		t.Fatalf("BytesToString(2048)=%q did not scale to a larger unit", BytesToString(2048))
	}
	if BytesToString(1024) == BytesToString(2048) {
		t.Fatalf("BytesToString ignores magnitude: %q", BytesToString(1024))
	}
}

// TestDetail04: CompatibleParseGCTime parses the old format directly, accepts
// the optional fractional field, drops a trailing zone-abbrev field on
// failure, and errors only after the fallback.
func TestDetail04(t *testing.T) {
	want := time.Date(2018, 12, 18, 11, 53, 37, 0, time.UTC)

	plain, err := CompatibleParseGCTime("20181218-11:53:37 +0000")
	if err != nil || !plain.Equal(want) {
		t.Fatalf("old-format parse: %v %v", plain, err)
	}
	frac, err := CompatibleParseGCTime("20181218-11:53:37.000 +0000")
	if err != nil || !frac.Equal(want) {
		t.Fatalf("new-format parse: %v %v", frac, err)
	}
	// Trailing zone abbreviation: fallback drops the last space-field.
	fallback, err := CompatibleParseGCTime("20181218-19:53:37 +0800 CST")
	if err != nil || !fallback.Equal(want) {
		t.Fatalf("zone-abbrev fallback: %v %v", fallback, err)
	}
	// Junk fails even after the fallback.
	for _, bad := range []string{"", " ", "foo", "20181218-19:53:37 +0800 FOO BAR"} {
		if _, err := CompatibleParseGCTime(bad); err == nil {
			t.Fatalf("CompatibleParseGCTime(%q) succeeded", bad)
		}
	}
}

// TestDetail05: EncodeToString hex-encodes into []byte; HexRegionKey is
// uppercase hex; HexRegionKeyStr is its string form.
func TestDetail05(t *testing.T) {
	src := []byte{0xde, 0xad, 0xbe, 0xef}
	if got := EncodeToString(src); string(got) != "deadbeef" {
		t.Fatalf("EncodeToString = %q, want %q", got, "deadbeef")
	}
	if got := HexRegionKey(src); string(got) != "DEADBEEF" {
		t.Fatalf("HexRegionKey = %q, want uppercase hex", got)
	}
	if got := HexRegionKeyStr(src); got != "DEADBEEF" {
		t.Fatalf("HexRegionKeyStr = %q, want %q", got, "DEADBEEF")
	}
}

// TestDetail06: ToUpperASCIIInplace mutates the slice in place and only
// touches a-z.
func TestDetail06(t *testing.T) {
	s := []byte("aBz-019x")
	out := ToUpperASCIIInplace(s)
	if string(out) != "ABZ-019X" {
		t.Fatalf("ToUpperASCIIInplace = %q", out)
	}
	if string(s) != "ABZ-019X" {
		t.Fatalf("input slice not mutated in place: %q", s)
	}
}

// TestDetail07: String is a zero-copy view — mutating the source bytes is
// visible through the returned string.
func TestDetail07(t *testing.T) {
	b := []byte("abc")
	s := String(b)
	if s != "abc" {
		t.Fatalf("String = %q", s)
	}
	b[0] = 'X'
	if s[0] != 'X' {
		t.Fatalf("String copied the bytes instead of viewing them")
	}
}
