package server

import (
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

// TestDetail01: the consumer-state record is byte0 magic, byte1 version,
// then uvarint AckFloor.Consumer, AckFloor.Stream, Delivered.Consumer,
// Delivered.Stream, uvarint len(Pending).
func TestDetail01(t *testing.T) {
	st := &ConsumerState{
		Delivered: SequencePair{Consumer: 300, Stream: 400},
		AckFloor:  SequencePair{Consumer: 10, Stream: 20},
	}
	buf := encodeConsumerState(st)
	if len(buf) < 7 || buf[0] != 22 || buf[1] != 2 {
		t.Fatalf("header bytes %v", buf[:4])
	}
	bi := 2
	read := func() uint64 {
		v, n := binary.Uvarint(buf[bi:])
		if n <= 0 {
			t.Fatalf("bad uvarint at %d", bi)
		}
		bi += n
		return v
	}
	if got := read(); got != 10 {
		t.Fatalf("AckFloor.Consumer = %d", got)
	}
	if got := read(); got != 20 {
		t.Fatalf("AckFloor.Stream = %d", got)
	}
	if got := read(); got != 300 {
		t.Fatalf("Delivered.Consumer = %d", got)
	}
	if got := read(); got != 400 {
		t.Fatalf("Delivered.Stream = %d", got)
	}
	if got := read(); got != 0 {
		t.Fatalf("len(Pending) = %d", got)
	}
}

// TestDetail02: with pending entries the encoder writes a varint mints
// anchor then per-entry stream-seq delta, consumer seq, and a signed
// second-resolution timestamp offset; a pending timestamp up to ~1s in
// the future still decodes.
func TestDetail02(t *testing.T) {
	future := time.Now().Add(500 * time.Millisecond).UnixNano()
	st := &ConsumerState{
		Delivered: SequencePair{Consumer: 100, Stream: 200},
		AckFloor:  SequencePair{Consumer: 50, Stream: 60},
		Pending:   map[uint64]*Pending{70: {Sequence: 55, Timestamp: future}},
	}
	buf := encodeConsumerState(st)
	ds, err := decodeConsumerState(buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	p, ok := ds.Pending[70]
	if !ok {
		t.Fatalf("pending entry lost: %+v", ds.Pending)
	}
	if p.Sequence != 55 {
		t.Fatalf("pending consumer seq = %d, want 55", p.Sequence)
	}
	// Second-resolution: decoded timestamp within a second of the input.
	if d := p.Timestamp - future; d < -int64(time.Second) || d > int64(time.Second) {
		t.Fatalf("timestamp drifted %dns", d)
	}
}

// TestDetail03: the redelivered count is always written, and entries are
// (stream-seq delta, count) uvarint pairs.
func TestDetail03(t *testing.T) {
	st := &ConsumerState{
		Delivered:   SequencePair{Consumer: 100, Stream: 200},
		AckFloor:    SequencePair{Consumer: 50, Stream: 60},
		Redelivered: map[uint64]uint64{80: 3},
	}
	buf := encodeConsumerState(st)
	ds, err := decodeConsumerState(buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ds.Redelivered[80] != 3 {
		t.Fatalf("redelivered: %+v", ds.Redelivered)
	}
	// Empty redelivered still encodes the count field.
	st2 := &ConsumerState{}
	buf2 := encodeConsumerState(st2)
	if len(buf2) < 7 {
		t.Fatalf("empty state too short: %d", len(buf2))
	}
	if ds2, err := decodeConsumerState(buf2); err != nil || ds2 == nil {
		t.Fatalf("empty decode: %v", err)
	}
}

// TestDetail04: the encoder returns a buffer sliced to the exact encoded
// size — no trailing slack.
func TestDetail04(t *testing.T) {
	st := &ConsumerState{
		Delivered:   SequencePair{Consumer: 1, Stream: 2},
		AckFloor:    SequencePair{Consumer: 3, Stream: 4},
		Redelivered: map[uint64]uint64{9: 1},
	}
	buf := encodeConsumerState(st)
	// Expected size from the committed layout: hdrLen + four uvarints +
	// pending count + redelivered count + two redelivered uvarints.
	want := 2 +
		uvarintLen(3) + uvarintLen(4) + uvarintLen(1) + uvarintLen(2) +
		uvarintLen(0) +
		uvarintLen(1) + uvarintLen(9-4) + uvarintLen(1)
	if len(buf) != want {
		t.Fatalf("len=%d, want exact %d (no slack)", len(buf), want)
	}
}

// TestDetail05: checkConsumerHeader gates on magic and version 1 or 2.
func TestDetail05(t *testing.T) {
	if _, err := checkConsumerHeader([]byte{22}); err == nil {
		t.Fatal("short header did not error")
	}
	if _, err := checkConsumerHeader([]byte{99, 2}); !errors.Is(err, errCorruptState) {
		t.Fatalf("bad magic: %v, want errCorruptState", err)
	}
	for _, v := range []uint8{1, 2} {
		if got, err := checkConsumerHeader([]byte{22, v}); err != nil || got != v {
			t.Fatalf("version %d: got %d err %v", v, got, err)
		}
	}
	if _, err := checkConsumerHeader([]byte{22, 9}); err == nil {
		t.Fatal("version 9 did not error")
	}
}

// TestDetail06: a failed varint mid-record poisons the cursor and yields
// errCorruptState.
func TestDetail06(t *testing.T) {
	st := &ConsumerState{
		Delivered:   SequencePair{Consumer: 1 << 40, Stream: 1 << 45},
		AckFloor:    SequencePair{Consumer: 1 << 33, Stream: 1 << 36},
		Redelivered: map[uint64]uint64{1 << 30: 7},
	}
	buf := encodeConsumerState(st)
	fails := 0
	for k := 2; k < len(buf); k++ {
		_, err := decodeConsumerState(buf[:k])
		if err == nil {
			continue // clean cut at a record boundary is allowed
		}
		if !errors.Is(err, errCorruptState) {
			t.Fatalf("truncation at %d: %v, want errCorruptState", k, err)
		}
		fails++
	}
	if fails == 0 {
		t.Fatal("no truncation errored — decoder too lenient")
	}
}

// TestDetail07 (shape): a version-1 record still decodes — Delivered is
// adjusted upward relative to the encoded values when the floor is > 1.
func TestDetail07(t *testing.T) {
	st := &ConsumerState{
		Delivered:   SequencePair{Consumer: 100, Stream: 200},
		AckFloor:    SequencePair{Consumer: 50, Stream: 60},
		Redelivered: map[uint64]uint64{80: 3},
	}
	buf := encodeConsumerState(st)
	v1 := append([]byte{}, buf...)
	v1[1] = 1
	ds, err := decodeConsumerState(v1)
	if err != nil {
		t.Fatalf("v1 decode: %v", err)
	}
	if ds.Delivered.Consumer < 100 || ds.Delivered.Stream < 200 {
		t.Fatalf("v1 delivered went backwards: %+v", ds.Delivered)
	}
}

// TestDetail08: corruption guards — top-bit stream sequences and a pending
// entry resolving to stream seq 0 error; zero-seq or zero-count
// redelivered entries are skipped.
func TestDetail08(t *testing.T) {
	// AckFloor.Stream with the top bit set.
	b := []byte{22, 2}
	b = binary.AppendUvarint(b, 1)
	b = binary.AppendUvarint(b, 1<<63|5)
	b = binary.AppendUvarint(b, 1)
	b = binary.AppendUvarint(b, 1)
	b = binary.AppendUvarint(b, 0)
	b = binary.AppendUvarint(b, 0)
	if _, err := decodeConsumerState(b); !errors.Is(err, errCorruptState) {
		t.Fatalf("top-bit AckFloor.Stream: %v", err)
	}
	// Pending stream-seq delta of 0 resolves to seq 0.
	b = []byte{22, 2}
	b = binary.AppendUvarint(b, 0)
	b = binary.AppendUvarint(b, 0)
	b = binary.AppendUvarint(b, 1)
	b = binary.AppendUvarint(b, 1)
	b = binary.AppendUvarint(b, 1)
	b = binary.AppendVarint(b, 1700000000)
	b = binary.AppendUvarint(b, 0)
	b = binary.AppendUvarint(b, 1)
	b = binary.AppendUvarint(b, 0)
	if _, err := decodeConsumerState(b); !errors.Is(err, errCorruptState) {
		t.Fatalf("pending seq 0: %v", err)
	}
	// Redelivered entries with seq 0 or count 0 are skipped.
	b = []byte{22, 2}
	b = binary.AppendUvarint(b, 0)
	b = binary.AppendUvarint(b, 0)
	b = binary.AppendUvarint(b, 1)
	b = binary.AppendUvarint(b, 1)
	b = binary.AppendUvarint(b, 0)
	b = binary.AppendUvarint(b, 2)
	b = binary.AppendUvarint(b, 0)
	b = binary.AppendUvarint(b, 0)
	b = binary.AppendUvarint(b, 5)
	b = binary.AppendUvarint(b, 2)
	ds, err := decodeConsumerState(b)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(ds.Redelivered) != 1 || ds.Redelivered[5] != 2 {
		t.Fatalf("zero redelivered entries not skipped: %+v", ds.Redelivered)
	}
}

// TestDetail09: the stream snapshot frame is magic + version + uvarints
// for Msgs, Bytes, FirstSeq, LastSeq, Failed; IsEncodedStreamState gates
// on magic, version, and minimum length only.
func TestDetail09(t *testing.T) {
	var f []byte
	f = append(f, 42, 1)
	for _, v := range []uint64{11, 22, 1, 9, 0} {
		f = binary.AppendUvarint(f, v)
	}
	f = binary.AppendUvarint(f, 0)
	if !IsEncodedStreamState(f) {
		t.Fatal("valid frame not recognised")
	}
	ss, err := DecodeStreamState(f)
	if err != nil {
		t.Fatalf("DecodeStreamState: %v", err)
	}
	if ss.Msgs != 11 || ss.Bytes != 22 || ss.FirstSeq != 1 || ss.LastSeq != 9 || ss.Failed != 0 {
		t.Fatalf("fields: %+v", ss)
	}
	if IsEncodedStreamState(nil) || IsEncodedStreamState([]byte{42, 1}) {
		t.Fatal("short/empty buffers recognised")
	}
	if IsEncodedStreamState([]byte{99, 1, 0, 0, 0, 0, 0}) {
		t.Fatal("bad magic recognised")
	}
}

// TestDetail10: the with-sources version carries, after Failed, a uvarint
// source count then per-source name/seq/ident, before any deleted blocks.
func TestDetail10(t *testing.T) {
	var f []byte
	f = append(f, 42, 2)
	for _, v := range []uint64{11, 22, 1, 9, 0} {
		f = binary.AppendUvarint(f, v)
	}
	f = binary.AppendUvarint(f, 1) // one source
	f = binary.AppendUvarint(f, 3)
	f = append(f, "src"...)
	f = binary.AppendUvarint(f, 77)
	f = binary.AppendUvarint(f, 2)
	f = append(f, "id"...)
	f = binary.AppendUvarint(f, 0) // no deleted blocks
	ss, err := DecodeStreamState(f)
	if err != nil {
		t.Fatalf("DecodeStreamState: %v", err)
	}
	src, ok := ss.Sources["src"]
	if !ok || src.Seq != 77 || src.Ident != "id" {
		t.Fatalf("sources: %+v", ss.Sources)
	}
}

// TestDetail11: deleted blocks start with a count then per-block magic —
// runLengthMagic decodes a (first, num) uvarint pair; unknown magic
// errors.
func TestDetail11(t *testing.T) {
	var f []byte
	f = append(f, 42, 1)
	for _, v := range []uint64{0, 0, 0, 9, 0} {
		f = binary.AppendUvarint(f, v)
	}
	f = binary.AppendUvarint(f, 1)
	f = append(f, runLengthMagic)
	f = binary.AppendUvarint(f, 5)
	f = binary.AppendUvarint(f, 3)
	ss, err := DecodeStreamState(f)
	if err != nil {
		t.Fatalf("run-length block: %v", err)
	}
	if len(ss.Deleted) != 1 {
		t.Fatalf("deleted blocks: %+v", ss.Deleted)
	}
	first, last, num := ss.Deleted[0].State()
	if first != 5 || last != 7 || num != 3 {
		t.Fatalf("block state %d %d %d, want 5 7 3", first, last, num)
	}
	// Unknown block magic errors.
	bad := append([]byte{}, f[:len(f)-3]...)
	bad = append(bad, 0xEE)
	bad = binary.AppendUvarint(bad, 5)
	bad = binary.AppendUvarint(bad, 3)
	if _, err := DecodeStreamState(bad); err == nil {
		t.Fatal("unknown block magic did not error")
	}
}

// TestDetail12: DeleteRange and DeleteSlice State/Range semantics, and
// DeleteBlocks.NumDeleted summation.
func TestDetail12(t *testing.T) {
	dr := &DeleteRange{First: 5, Num: 3}
	if f, l, n := dr.State(); f != 5 || l != 7 || n != 3 {
		t.Fatalf("DeleteRange.State = %d %d %d", f, l, n)
	}
	var seen []uint64
	dr.Range(func(s uint64) bool { seen = append(seen, s); return true })
	if len(seen) != 3 || seen[0] != 5 || seen[2] != 7 {
		t.Fatalf("DeleteRange.Range: %v", seen)
	}
	ds := DeleteSlice{9, 2, 7}
	if f, l, n := ds.State(); f != 9 || l != 7 || n != 3 {
		t.Fatalf("DeleteSlice.State = %d %d %d", f, l, n)
	}
	var empty DeleteSlice
	if f, l, n := empty.State(); f != 0 || l != 0 || n != 0 {
		t.Fatalf("empty DeleteSlice.State = %d %d %d", f, l, n)
	}
	seen = seen[:0]
	ds.Range(func(s uint64) bool { seen = append(seen, s); return true })
	if len(seen) != 3 || seen[0] != 9 || seen[1] != 2 || seen[2] != 7 {
		t.Fatalf("DeleteSlice.Range order: %v", seen)
	}
	dbs := DeleteBlocks{dr, ds}
	if dbs.NumDeleted() != 6 {
		t.Fatalf("NumDeleted = %d, want 6", dbs.NumDeleted())
	}
}

// TestDetail13: uvarintLen matches binary.PutUvarint's length;
// appendRunLength emits the runLengthMagic then the two uvarints.
func TestDetail13(t *testing.T) {
	var tmp [binary.MaxVarintLen64]byte
	for _, v := range []uint64{0, 1, 127, 128, 300, 1 << 35, 1 << 63} {
		if uvarintLen(v) != binary.PutUvarint(tmp[:], v) {
			t.Fatalf("uvarintLen(%d) = %d", v, uvarintLen(v))
		}
	}
	if runLengthEncodeLen(5, 3) != 1+uvarintLen(5)+uvarintLen(3) {
		t.Fatal("runLengthEncodeLen mismatch")
	}
	b := appendRunLength(nil, 5, 3)
	if len(b) != runLengthEncodeLen(5, 3) || b[0] != runLengthMagic {
		t.Fatalf("appendRunLength: %v", b)
	}
	first, n1 := binary.Uvarint(b[1:])
	num, n2 := binary.Uvarint(b[1+n1:])
	if first != 5 || num != 3 || 1+n1+n2 != len(b) {
		t.Fatalf("appendRunLength fields: %d %d", first, num)
	}
}
