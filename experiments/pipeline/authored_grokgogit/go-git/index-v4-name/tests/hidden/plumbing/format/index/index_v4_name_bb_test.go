package index

import (
	"bytes"
	"crypto"
	"errors"
	"testing"
	"time"

	phash "example.internal/gitkit/v6/plumbing/hash"
)

const (
	ivHeader = 12 // signature + version + entry count
	ivFixed  = 62 // entryHeaderLength(42) + sha1 size(20)
	ivFooter = 20 // sha1 trailer
)

func ivEncode(t *testing.T, idx *Index, opts ...Option) []byte {
	t.Helper()
	var buf bytes.Buffer
	e := NewEncoder(&buf, phash.New(crypto.SHA1), opts...)
	if err := e.Encode(idx); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.Bytes()
}

func ivEntry(name string) *Entry {
	return &Entry{Name: name, CreatedAt: time.Unix(1700000000, 0), ModifiedAt: time.Unix(1700000000, 0)}
}

// failPastWriter accepts up to limit bytes total, then errors.
type failPastWriter struct {
	limit int
	got   int
}

func (w *failPastWriter) Write(p []byte) (int, error) {
	if w.got+len(p) > w.limit {
		// Commit only what fits, then fail — a mid-entry error.
		n := w.limit - w.got
		w.got += n
		return n, errIvWrite
	}
	w.got += len(p)
	return len(p), nil
}

var errIvWrite = errors.New("write budget exhausted")

// TestDetail01 (partially): index version 4 writes no NUL padding after an
// entry — total length is header + fixed + varint + suffix + NUL + footer.
func TestDetail01(t *testing.T) {
	out := ivEncode(t, &Index{
		Version: 4,
		Entries: []*Entry{ivEntry("aaaa"), ivEntry("aaab")},
	}, WithSkipHash())

	// e1 "aaaa": varint(0) + "aaaa" + NUL = 6 name bytes -> 68.
	// e2 "aaab": strip 1 -> varint(1) + "b" + NUL = 3 name bytes -> 65.
	wantLen := ivHeader + (ivFixed + 6) + (ivFixed + 3) + ivFooter
	if len(out) != wantLen {
		t.Fatalf("v4 output len = %d, want %d (padding would make it bigger)", len(out), wantLen)
	}
}

// TestDetail02 (shape — Inferable: no): versions other than 4 pad so that
// wrote+padLen is a multiple of 8, with padLen = 8 - wrote%8 — an already
// aligned entry still writes 8 NULs.
func TestDetail02(t *testing.T) {
	// name "xy": wrote = 62+2 = 64, already a multiple of 8 -> padLen 8.
	out := ivEncode(t, &Index{Version: 2, Entries: []*Entry{ivEntry("xy")}}, WithSkipHash())
	if len(out) != ivHeader+ivFixed+2+8+ivFooter {
		t.Fatalf("v2 aligned entry len = %d, want %d", len(out), ivHeader+ivFixed+2+8+ivFooter)
	}
	pad := out[ivHeader+ivFixed+2 : ivHeader+ivFixed+2+8]
	if !bytes.Equal(pad, make([]byte, 8)) {
		t.Fatalf("aligned pad bytes = %v, want 8 NULs", pad)
	}

	// name "x": wrote = 63 -> padLen 1.
	out = ivEncode(t, &Index{Version: 2, Entries: []*Entry{ivEntry("x")}}, WithSkipHash())
	if len(out) != ivHeader+ivFixed+1+1+ivFooter {
		t.Fatalf("v2 len-1 name total = %d, want %d", len(out), ivHeader+ivFixed+1+1+ivFooter)
	}
	if out[ivHeader+ivFixed+1] != 0 {
		t.Fatal("missing pad NUL after short name")
	}
}

// TestDetail03 (doc): v4 names are prefix-compressed against the previous
// entry's name in sorted order — entries are sorted before compression.
func TestDetail03(t *testing.T) {
	// Entries deliberately out of order: compression must run against the
	// sorted predecessor "aaaa", not the input neighbour.
	out := ivEncode(t, &Index{
		Version: 4,
		Entries: []*Entry{ivEntry("aaab"), ivEntry("aaaa")},
	}, WithSkipHash())

	// After sorting, e1 is "aaaa": varint 0 + "aaaa" + NUL.
	off := ivHeader + ivFixed
	if !bytes.Equal(out[off:off+6], []byte{0, 'a', 'a', 'a', 'a', 0}) {
		t.Fatalf("first v4 name region = %v, want [0 aaaa 0]", out[off:off+6])
	}
	// e2 "aaab" vs prev "aaaa": strip 4-3=1, suffix "b".
	off += 6 + ivFixed
	if !bytes.Equal(out[off:off+3], []byte{1, 'b', 0}) {
		t.Fatalf("second v4 name region = %v, want [1 'b' 0]", out[off:off+3])
	}
}

// TestDetail04 (shape — Inferable: no): the encoder writes a
// variable-width integer equal to len(previousName) - commonPrefixLen, and
// 0 for the first entry.
func TestDetail04(t *testing.T) {
	out := ivEncode(t, &Index{
		Version: 4,
		Entries: []*Entry{ivEntry("prefix-one"), ivEntry("prefix-two")},
	}, WithSkipHash())

	// e1: varint 0, full name, NUL.
	off := ivHeader + ivFixed
	if out[off] != 0 {
		t.Fatalf("first entry strip byte = %d, want 0", out[off])
	}
	// e2 "prefix-two" vs "prefix-one": common prefix "prefix-" = 7,
	// strip = 10-7 = 3 -> varint 3, suffix "two".
	off += 1 + len("prefix-one") + 1 + ivFixed
	if !bytes.Equal(out[off:off+5], []byte{3, 't', 'w', 'o', 0}) {
		t.Fatalf("second entry name region = %v, want [3 'two' 0]", out[off:off+5])
	}
}

// TestDetail05 (shape — Inferable: no): then the suffix current[prefix:]
// followed by a single NUL.
func TestDetail05(t *testing.T) {
	out := ivEncode(t, &Index{
		Version: 4,
		Entries: []*Entry{ivEntry("aaaa"), ivEntry("aaab")},
	}, WithSkipHash())

	off := ivHeader + ivFixed + 6 + ivFixed
	// Exactly varint + one suffix byte + one NUL, then the footer.
	if !bytes.Equal(out[off:off+3], []byte{1, 'b', 0}) {
		t.Fatalf("name suffix region = %v", out[off:off+3])
	}
}

// TestDetail06 (partially): commonPrefixLen is byte-wise, not rune-wise.
func TestDetail06(t *testing.T) {
	// "éz" = c3 a9 7a, "ëz" = c3 ab 7a — byte prefix 1, rune prefix 0.
	out := ivEncode(t, &Index{
		Version: 4,
		Entries: []*Entry{ivEntry("éz"), ivEntry("ëz")},
	}, WithSkipHash())

	off := ivHeader + ivFixed
	// e1 "éz": varint 0 + 3 name bytes + NUL.
	if out[off] != 0 {
		t.Fatalf("first entry strip = %d, want 0", out[off])
	}
	off += 1 + 3 + 1 + ivFixed
	// e2: byte-wise strip = len("éz") - 1 = 2, suffix = [0xab, 'z'].
	// (Rune-wise would give prefix 0 -> strip 3 -> [3 0xc3 0xab 'z' 0].)
	want := []byte{2, 0xab, 'z', 0}
	if !bytes.Equal(out[off:off+4], want) {
		t.Fatalf("multibyte name region = %v, want %v (byte-wise prefix)", out[off:off+4], want)
	}
}

// TestDetail07 (shape — Inferable: no): lastEntry is updated to the
// current entry before returning, including on a later write error.
func TestDetail07(t *testing.T) {
	e1, e2 := ivEntry("aaaa"), ivEntry("aaab")
	w := &failPastWriter{limit: ivHeader + (ivFixed + 6) + ivFixed + 1}
	e := NewEncoder(w, phash.New(crypto.SHA1), WithSkipHash())
	err := e.Encode(&Index{Version: 4, Entries: []*Entry{e1, e2}})
	if err == nil {
		t.Fatal("expected write error")
	}
	if e.lastEntry == nil || e.lastEntry.Name != "aaab" {
		var got string
		if e.lastEntry != nil {
			got = e.lastEntry.Name
		}
		t.Fatalf("lastEntry = %q after mid-entry error, want %q", got, "aaab")
	}
}

// TestDetail08 (yes): v2/v3 still write the name as raw bytes with no NUL
// of their own — padding supplies the NULs.
func TestDetail08(t *testing.T) {
	for _, v := range []uint32{2, 3} {
		out := ivEncode(t, &Index{Version: v, Entries: []*Entry{ivEntry("xy")}}, WithSkipHash())
		name := out[ivHeader+ivFixed : ivHeader+ivFixed+2]
		if !bytes.Equal(name, []byte("xy")) {
			t.Fatalf("v%d name field = %q, want raw \"xy\"", v, name)
		}
		if bytes.IndexByte(name, 0) != -1 {
			t.Fatalf("v%d name field contains a NUL", v)
		}
	}
}
