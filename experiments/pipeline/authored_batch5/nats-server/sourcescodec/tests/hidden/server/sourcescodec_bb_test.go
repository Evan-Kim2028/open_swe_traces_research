package server

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestDetail01: sources.db layout — version byte 1, LE u64 count, LE u64
// highSeq stamp = LastSeq+1; per source a uvarint-len name, uvarint seq,
// uvarint-len ident.
func TestDetail01(t *testing.T) {
	dir := t.TempDir()
	fs := &fileStore{}
	fs.dios = defaultDiskIOSemaphore()
	fs.fcfg.StoreDir = dir
	fs.state.LastSeq = 41
	fs.sources = map[string]*StreamSourceState{
		"srcA": {Seq: 10, Ident: "idA"},
		"srcB": {Seq: 20},
	}
	if err := fs.writeSourcesState(); err != nil {
		t.Fatalf("writeSourcesState: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, sourcesStreamStateFile))
	if err != nil {
		t.Fatal(err)
	}
	le := binary.LittleEndian
	if len(b) < sourcesHeaderLen || b[0] != 1 {
		t.Fatalf("missing version: %x", b)
	}
	if le.Uint64(b[1:]) != 2 {
		t.Fatalf("count = %d", le.Uint64(b[1:]))
	}
	if le.Uint64(b[9:]) != 42 {
		t.Fatalf("stamp should be LastSeq+1 = 42, got %d", le.Uint64(b[9:]))
	}
	// Per-source entries: uvarint len + name, uvarint seq, uvarint len + ident.
	type ent struct {
		name  string
		seq   uint64
		ident string
	}
	var ents []ent
	i := sourcesHeaderLen
	for i < len(b) {
		n, w := binary.Uvarint(b[i:])
		if w <= 0 || i+w+int(n) > len(b) {
			t.Fatalf("name field at %d", i)
		}
		i += w
		name := string(b[i : i+int(n)])
		i += int(n)
		seq, w := binary.Uvarint(b[i:])
		if w <= 0 {
			t.Fatalf("seq field at %d", i)
		}
		i += w
		n, w = binary.Uvarint(b[i:])
		if w <= 0 || i+w+int(n) > len(b) {
			t.Fatalf("ident field at %d", i)
		}
		i += w
		ents = append(ents, ent{name, seq, string(b[i : i+int(n)])})
		i += int(n)
	}
	if len(ents) != 2 {
		t.Fatalf("decoded %d sources", len(ents))
	}
	found := map[string]ent{}
	for _, e := range ents {
		found[e.name] = e
	}
	if a := found["srcA"]; a.seq != 10 || a.ident != "idA" {
		t.Fatalf("srcA = %+v", a)
	}
	if b_ := found["srcB"]; b_.seq != 20 || b_.ident != "" {
		t.Fatalf("srcB = %+v", b_)
	}
}

// TestDetail02: decodeSourcesState errors — <17 short buffer, bad version,
// io.ErrUnexpectedEOF per truncated field; sources allocated only when
// count>0.
func TestDetail02(t *testing.T) {
	dir := t.TempDir()
	fs := &fileStore{}
	fs.dios = defaultDiskIOSemaphore()
	fs.fcfg.StoreDir = dir
	fs.state.LastSeq = 41
	fs.sources = map[string]*StreamSourceState{"srcA": {Seq: 10, Ident: "idA"}}
	if err := fs.writeSourcesState(); err != nil {
		t.Fatal(err)
	}
	good, _ := os.ReadFile(filepath.Join(dir, sourcesStreamStateFile))

	fs2 := &fileStore{}
	if _, err := fs2.decodeSourcesState(good[:16]); !errors.Is(err, io.ErrShortBuffer) {
		t.Fatalf("short: %v", err)
	}
	badv := append([]byte{}, good...)
	badv[0] = 9
	if _, err := fs2.decodeSourcesState(badv); !errors.Is(err, errSourcesInvalidVersion) {
		t.Fatalf("bad version: %v", err)
	}
	for _, cut := range []int{17, 18, len(good) - 1} {
		fs3 := &fileStore{}
		if _, err := fs3.decodeSourcesState(good[:cut]); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("cut %d: %v", cut, err)
		}
	}
	// A uvarint length overrunning the buffer is unexpected EOF too.
	ov := append([]byte{}, good[:sourcesHeaderLen]...)
	ov = append(ov, 50) // declares a 50-byte name that isn't there
	if _, err := (&fileStore{}).decodeSourcesState(ov); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("overrun name: %v", err)
	}
	// Header with count>0 allocates sources even though entries are missing.
	fs4 := &fileStore{}
	_, err := fs4.decodeSourcesState(good[:sourcesHeaderLen])
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("count>0 header-only: %v", err)
	}
	if fs4.sources == nil {
		t.Fatal("count>0 should allocate fs.sources")
	}
	// Round trip returns the stamp and populates sources.
	fs5 := &fileStore{}
	stamp, err := fs5.decodeSourcesState(good)
	if err != nil || stamp != 42 {
		t.Fatalf("round trip stamp = %d err=%v", stamp, err)
	}
	if fs5.sources["srcA"].Seq != 10 || fs5.sources["srcA"].Ident != "idA" {
		t.Fatalf("sources = %v", fs5.sources)
	}
}

// TestDetail03: streamAndSeq dispatch — $JS.ACK headers go to the legacy
// decoder; space-split headers need != 1, 3 fields (2 or >=4).
func TestDetail03(t *testing.T) {
	// Legacy branch is keyed on the $JS.ACK prefix, not field count.
	st, _, seq, _ := streamAndSeq("$JS.ACK.st1.cons1.3.55.9.111.7")
	if st != "st1" || seq != 55 {
		t.Fatalf("v1 ack: %q %d", st, seq)
	}
	// 1- and 3-field space headers are rejected; 2 fields is a legacy pair.
	if st, in, sq, id := streamAndSeq("one"); st+in != "" || sq != 0 || id != "" {
		t.Fatalf("1 field: %q %q %d %q", st, in, sq, id)
	}
	if st, in, sq, id := streamAndSeq("a b c"); st+in != "" || sq != 0 || id != "" {
		t.Fatalf("3 fields: %q %q %d %q", st, in, sq, id)
	}
	st, in, seq, id := streamAndSeq("a b")
	if st != "a" || in != "" || id != "" {
		t.Fatalf("2 fields: %q %q %q", st, in, id)
	}
	_ = seq // seq value of the 2-field form is not pinned here
	if st, _, _, _ := streamAndSeq("a b c d"); st != "a" {
		t.Fatalf("4+ fields parse as v2: %q", st)
	}
}

// TestDetail04: v2 reconstruction — iname = fields[0]+" "+fields[2]+" "+
// fields[3]; seq = parseAckReplyNum(fields[1]); ident = fields[5] when
// present; empties on mismatch.
func TestDetail04(t *testing.T) {
	st, in, seq, id := streamAndSeq("mystream 42 src dest orig")
	if st != "mystream" || in != "mystream src dest" || seq != 42 || id != "" {
		t.Fatalf("v2 5-field: %q %q %d %q", st, in, seq, id)
	}
	st, in, seq, id = streamAndSeq("mystream 42 src dest orig ident1")
	if st != "mystream" || in != "mystream src dest" || seq != 42 || id != "ident1" {
		t.Fatalf("v2 6-field: %q %q %d %q", st, in, seq, id)
	}
	// A non-numeric seq field flows through parseAckReplyNum's invalid marker.
	_, _, seq, _ = streamAndSeq("mystream zz src dest orig")
	if seq != math.MaxUint64 {
		t.Fatalf("bad seq = %d", seq)
	}
}

// TestDetail05: genSourceHeader writes "<iname-part0> <seq> <iname-part1>
// <iname-part2> <orig> [ident]" — seq lifted from token 5 (v1) or 7 (v2)
// of the ack reply, defaulting to "1"; ident only when non-empty.
func TestDetail05(t *testing.T) {
	si := &sourceInfo{name: "STR", iname: "STR filt trs"}
	if h := si.genSourceHeader("origsubj", "$JS.ACK.STR.cons.3.77.9.111.7", "ident1"); h != "STR 77 filt trs origsubj ident1" {
		t.Fatalf("v1 ack header = %q", h)
	}
	if h := si.genSourceHeader("origsubj", "$JS.ACK.dom.acc.STR.cons.3.77.9.111.7", "ident1"); h != "STR 77 filt trs origsubj ident1" {
		t.Fatalf("v2 ack header = %q", h)
	}
	if h := si.genSourceHeader("origsubj", "not.an.ack", "ident1"); h != "STR 1 filt trs origsubj ident1" {
		t.Fatalf("non-ack header = %q", h)
	}
	if h := si.genSourceHeader("origsubj", "$JS.ACK.STR.cons.3.77.9.111.7", ""); h != "STR 77 filt trs origsubj" {
		t.Fatalf("no-ident header = %q", h)
	}
}

// TestDetail06: streamAndSeqFromAckReply lifts (stream, iname, sseq) from
// both v1 and v2 reply layouts; malformed replies return empties.
func TestDetail06(t *testing.T) {
	st, in, seq := streamAndSeqFromAckReply("$JS.ACK.st1.cons1.3.55.9.111.7")
	if st != "st1" || seq != 55 {
		t.Fatalf("v1: %q %q %d", st, in, seq)
	}
	st, in, seq = streamAndSeqFromAckReply("$JS.ACK.dom.acc.st2.cons2.3.66.9.111.7")
	if st != "st2" || seq != 66 {
		t.Fatalf("v2: %q %q %d", st, in, seq)
	}
	for _, r := range []string{"garbage", "$JS.ACK.too.few", ""} {
		if st, in, seq := streamAndSeqFromAckReply(r); st != "" || in != "" || seq != 0 {
			t.Fatalf("malformed %q: %q %q %d", r, st, in, seq)
		}
	}
}
