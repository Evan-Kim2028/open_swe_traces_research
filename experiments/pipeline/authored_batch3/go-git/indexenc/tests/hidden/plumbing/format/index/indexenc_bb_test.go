package index

import (
	"bytes"
	"crypto"
	"crypto/sha1"
	"encoding/binary"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	phash "example.internal/gitkit/v6/plumbing/hash"
)

func enc(idx *Index, opts ...Option) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	err := NewEncoder(&buf, phash.New(crypto.SHA1), opts...).Encode(idx)
	return &buf, err
}

func mkEntry(name string, stage Stage) *Entry {
	h, _ := plumbing.FromHex("1234567890123456789012345678901234567890")
	return &Entry{
		Name:       name,
		Hash:       h,
		CreatedAt:  time.Unix(1700000000, 123),
		ModifiedAt: time.Unix(1700000001, 456),
		Mode:       filemode.Regular,
		Size:       42,
		Stage:      stage,
	}
}

func roundTrip(t *testing.T, idx *Index, opts ...Option) *Index {
	t.Helper()
	buf, err := enc(idx, opts...)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := &Index{}
	dopts := opts // skip-hash needs the same flag on decode
	if err := NewDecoder(buf, phash.New(crypto.SHA1), dopts...).Decode(out); err != nil {
		t.Fatalf("Decode of encoded index: %v", err)
	}
	return out
}

// TestDetail01: DIRC signature, version, entry count as big-endian u32s;
// version above the supported bound fails before anything is written.
func TestDetail01(t *testing.T) {
	buf, err := enc(&Index{Version: 2, Entries: []*Entry{mkEntry("a", Merged)}})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	raw := buf.Bytes()
	if !bytes.Equal(raw[:4], []byte("DIRC")) {
		t.Fatalf("signature = %q", raw[:4])
	}
	if binary.BigEndian.Uint32(raw[4:8]) != 2 {
		t.Fatalf("version = %d", binary.BigEndian.Uint32(raw[4:8]))
	}
	if binary.BigEndian.Uint32(raw[8:12]) != 1 {
		t.Fatalf("entry count = %d", binary.BigEndian.Uint32(raw[8:12]))
	}

	buf, err = enc(&Index{Version: EncodeVersionSupported + 1})
	if err == nil {
		t.Fatal("unsupported version encoded without error")
	}
	if buf.Len() != 0 {
		t.Fatalf("unsupported version wrote %d bytes before failing", buf.Len())
	}
}

// TestDetail02: entries are written sorted by name then stage.
func TestDetail02(t *testing.T) {
	idx := &Index{Version: 2, Entries: []*Entry{
		mkEntry("b", TheirMode),
		mkEntry("a", Merged),
		mkEntry("b", AncestorMode),
	}}
	out := roundTrip(t, idx)
	want := []struct {
		n string
		s Stage
	}{{"a", Merged}, {"b", AncestorMode}, {"b", TheirMode}}
	if len(out.Entries) != 3 {
		t.Fatalf("got %d entries", len(out.Entries))
	}
	for i, e := range out.Entries {
		if e.Name != want[i].n || e.Stage != want[i].s {
			t.Fatalf("entry %d = %s stage %d, want %s stage %d", i, e.Name, e.Stage, want[i].n, want[i].s)
		}
	}
}

// TestDetail03: fixed part is ten u32s (ctime, mtime, dev, ino, mode, uid,
// gid, size) then hash then flags — verified by field round-trip.
func TestDetail03(t *testing.T) {
	e := mkEntry("path/to/f", TheirMode)
	e.Dev, e.Inode, e.UID, e.GID = 11, 22, 33, 44
	out := roundTrip(t, &Index{Version: 3, Entries: []*Entry{e}})
	got := out.Entries[0]
	if got.CreatedAt.Unix() != 1700000000 || got.CreatedAt.Nanosecond() != 123 ||
		got.ModifiedAt.Unix() != 1700000001 || got.ModifiedAt.Nanosecond() != 456 {
		t.Fatalf("timestamps not preserved: %v %v", got.CreatedAt, got.ModifiedAt)
	}
	if got.Dev != 11 || got.Inode != 22 || got.UID != 33 || got.GID != 44 ||
		got.Mode != filemode.Regular || got.Size != 42 || got.Stage != TheirMode ||
		got.Hash != e.Hash {
		t.Fatal("fixed-part fields did not round-trip")
	}
}

// TestDetail04: flags carry stage in bits 12-13 and name length in the low 12
// bits, saturating at 0xFFF rather than failing.
func TestDetail04(t *testing.T) {
	long := bytes.Repeat([]byte("n"), 5000)
	e := mkEntry(string(long), TheirMode)
	buf, err := enc(&Index{Version: 2, Entries: []*Entry{e}})
	if err != nil {
		t.Fatalf("Encode long name: %v", err)
	}
	raw := buf.Bytes()
	flags := binary.BigEndian.Uint16(raw[12+62-2 : 12+62])
	if flags&0xFFF != 0xFFF {
		t.Fatalf("long-name flags low bits = %#x, want saturated 0xFFF", flags&0xFFF)
	}
	if flags>>12&0x3 != uint16(TheirMode) {
		t.Fatalf("stage bits = %#x, want %d", flags>>12&0x3, TheirMode)
	}
	out := roundTrip(t, &Index{Version: 2, Entries: []*Entry{e}})
	if out.Entries[0].Name != e.Name {
		t.Fatalf("long name did not round-trip: %d bytes", len(out.Entries[0].Name))
	}
}

// TestDetail05: intent-to-add / skip-worktree set an extended-flags bit and
// append a second u16 present only when one of them is set.
func TestDetail05(t *testing.T) {
	plain := mkEntry("p", Merged)
	itAdd := mkEntry("i", Merged)
	itAdd.IntentToAdd = true
	swt := mkEntry("s", Merged)
	swt.SkipWorktree = true

	buf, err := enc(&Index{Version: 3, Entries: []*Entry{plain, itAdd, swt}})
	if err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()[12:]
	// Entries sorted: i, p, s. Entry "i" starts at offset 0; its 16-bit flags
	// sit at [60:62] of the fixed part (40 stat bytes + 20 hash + 2 flags).
	flags := binary.BigEndian.Uint16(raw[60:62])
	if flags&0x4000 == 0 {
		t.Fatalf("intent-to-add entry missing extended flag: %#x", flags)
	}
	ext := binary.BigEndian.Uint16(raw[62:64])
	if ext&intentToAddMask == 0 {
		t.Fatalf("extended flags = %#x, want intent-to-add bit", ext)
	}

	// "i" entry occupies 62 fixed + 2 ext + 1 name = 65 bytes, padded to 72;
	// the plain "p" entry's flags must NOT carry the extended bit.
	flagsP := binary.BigEndian.Uint16(raw[72+60 : 72+62])
	if flagsP&0x4000 != 0 {
		t.Fatalf("plain entry carried extended flag: %#x", flagsP)
	}

	out := roundTrip(t, &Index{Version: 3, Entries: []*Entry{itAdd, swt}})
	if !out.Entries[0].IntentToAdd || !out.Entries[1].SkipWorktree {
		t.Fatal("extended flags did not round-trip")
	}
}

// TestDetail06: zero timestamps encode as 0/0; negative values are an
// invalid-timestamp failure.
func TestDetail06(t *testing.T) {
	e := mkEntry("z", Merged)
	e.CreatedAt = time.Time{}
	e.ModifiedAt = time.Time{}
	if _, err := enc(&Index{Version: 2, Entries: []*Entry{e}}); err != nil {
		t.Fatalf("zero timestamp: %v", err)
	}
	e.CreatedAt = time.Unix(-5, 0)
	if _, err := enc(&Index{Version: 2, Entries: []*Entry{e}}); err == nil {
		t.Fatal("negative timestamp encoded without error")
	}
}

// TestDetail07: v2/v3 pad each entry to an 8-byte boundary with NULs, always
// at least one.
func TestDetail07(t *testing.T) {
	// fixed=62; name "ab" -> 64 -> pads 8 NULs; name "abcdef" -> 68 -> pads 4.
	for _, tc := range []struct {
		name string
		pad  int
	}{{"ab", 8}, {"abcdef", 4}, {"abcdefg", 3}} {
		buf, err := enc(&Index{Version: 2, Entries: []*Entry{mkEntry(tc.name, Merged)}})
		if err != nil {
			t.Fatal(err)
		}
		raw := buf.Bytes()[12:]
		entryLen := 62 + len(tc.name) + tc.pad
		if entryLen%8 != 0 {
			t.Fatalf("test construction: entry %d not 8-aligned", entryLen)
		}
		padding := raw[62+len(tc.name) : entryLen]
		for i, b := range padding {
			if b != 0 {
				t.Fatalf("name %q: padding byte %d = %#x, want NUL", tc.name, i, b)
			}
		}
		if len(padding) != tc.pad {
			t.Fatalf("name %q: pad %d, want %d", tc.name, len(padding), tc.pad)
		}
	}
}

// TestDetail08: v4 writes no padding — each name is a varint strip count off
// the END of the previous name, then the remaining suffix and a NUL.
func TestDetail08(t *testing.T) {
	idx := &Index{Version: 4, Entries: []*Entry{
		mkEntry("abcde", Merged), mkEntry("abcef", Merged), mkEntry("abxyz", Merged),
	}}
	buf, err := enc(idx)
	if err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	// Layout (relative to raw): header 12; e1 fixed 62 + vlq(0) 1 + "abcde\0" 6
	// = 69 -> e2 at 12+69=81; e2 fixed [81,143), vlq@143, "ef\0" [144,147);
	// e3 at 147: fixed [147,209), vlq@209, "xyz\0" [210,214); trailer 20.
	if len(raw) != 234 {
		t.Fatalf("v4 index size = %d, want 234 — padding must not appear", len(raw))
	}
	if raw[143] != 0x02 {
		t.Fatalf("e2 strip count = %#x, want 0x02 (strip \"de\")", raw[143])
	}
	if string(raw[144:147]) != "ef\x00" {
		t.Fatalf("e2 suffix = %q, want %q", raw[144:147], "ef\x00")
	}
	if raw[209] != 0x03 {
		t.Fatalf("e3 strip count = %#x, want 0x03 (strip \"cef\")", raw[209])
	}
	if string(raw[210:214]) != "xyz\x00" {
		t.Fatalf("e3 suffix = %q, want %q", raw[210:214], "xyz\x00")
	}
	out := roundTrip(t, idx)
	for i, e := range out.Entries {
		if e.Name != idx.Entries[i].Name {
			t.Fatalf("v4 entry %d name = %q, want %q", i, e.Name, idx.Entries[i].Name)
		}
	}
}

// TestDetail09: the first v4 entry strips nothing.
func TestDetail09(t *testing.T) {
	idx := &Index{Version: 4, Entries: []*Entry{mkEntry("first", Merged)}}
	buf, err := enc(idx)
	if err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()[12:]
	if raw[62] != 0x00 {
		t.Fatalf("first v4 entry strip count = %#x, want 0", raw[62])
	}
	if string(raw[63:68]) != "first" || raw[68] != 0 {
		t.Fatalf("first v4 name section = %q", raw[63:69])
	}
}

// TestDetail10: the padding length counts the 2-byte extended-flags word.
func TestDetail10(t *testing.T) {
	e := mkEntry("abcde", Merged) // 62 + 2(ext) + 5 = 69 -> pad 3
	e.IntentToAdd = true
	buf, err := enc(&Index{Version: 3, Entries: []*Entry{e}})
	if err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()[12:]
	total := 62 + 2 + 5
	pad := 8 - total%8
	for i := 0; i < pad; i++ {
		if raw[total+i] != 0 {
			t.Fatalf("padding byte %d = %#x", i, raw[total+i])
		}
	}
	// the trailer starts right after the padding (raw excludes the header)
	if got := len(raw) - 20; got != total+pad {
		t.Fatalf("entry+pad = %d bytes, want %d", got, total+pad)
	}
}

// TestDetail11: trailer is the hash of every byte written; under skip-hash a
// hash-length run of zeros is appended instead.
func TestDetail11(t *testing.T) {
	idx := &Index{Version: 2, Entries: []*Entry{mkEntry("a", Merged)}}
	buf, err := enc(idx)
	if err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	sum := sha1.Sum(raw[:len(raw)-20])
	if !bytes.Equal(raw[len(raw)-20:], sum[:]) {
		t.Fatal("trailer is not sha1 of the written stream")
	}

	buf, err = enc(idx, WithSkipHash())
	if err != nil {
		t.Fatal(err)
	}
	raw = buf.Bytes()
	trailer := raw[len(raw)-20:]
	if !bytes.Equal(trailer, make([]byte, 20)) {
		t.Fatalf("skip-hash trailer = %x, want zeros", trailer)
	}
	// the zero trailer must still decode when the reader also skips the hash
	roundTrip(t, idx, WithSkipHash())
}

// TestDetail12: a raw extension is sig+u32+payload; a signature of any other
// length is rejected.
func TestDetail12(t *testing.T) {
	var buf bytes.Buffer
	e := NewEncoder(&buf, phash.New(crypto.SHA1))
	if err := e.encodeRawExtension("TOOOLONG", []byte("x")); err == nil {
		t.Fatal("7-char extension signature accepted")
	}
	if err := e.encodeRawExtension("XY", []byte("x")); err == nil {
		t.Fatal("2-char extension signature accepted")
	}
	buf.Reset()
	if err := e.encodeRawExtension("ABCD", []byte("pay")); err != nil {
		t.Fatalf("4-char signature rejected: %v", err)
	}
	raw := buf.Bytes()
	if string(raw[:4]) != "ABCD" || binary.BigEndian.Uint32(raw[4:8]) != 3 || string(raw[8:11]) != "pay" {
		t.Fatalf("extension encoding = %x", raw)
	}
}
