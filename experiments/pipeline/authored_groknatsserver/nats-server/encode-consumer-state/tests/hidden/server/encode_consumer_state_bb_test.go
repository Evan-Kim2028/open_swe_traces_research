package server

import (
	"encoding/binary"
	"reflect"
	"testing"
	"time"
)

// ecsUvarint reads one uvarint at *off and advances it.
func ecsUvarint(t *testing.T, buf []byte, off *int) uint64 {
	t.Helper()
	v, n := binary.Uvarint(buf[*off:])
	if n <= 0 {
		t.Fatalf("uvarint decode failed at offset %d (n=%d)", *off, n)
	}
	*off += n
	return v
}

// ecsVarint reads one signed varint at *off and advances it.
func ecsVarint(t *testing.T, buf []byte, off *int) int64 {
	t.Helper()
	v, n := binary.Varint(buf[*off:])
	if n <= 0 {
		t.Fatalf("varint decode failed at offset %d (n=%d)", *off, n)
	}
	*off += n
	return v
}

func ecsBase() *ConsumerState {
	return &ConsumerState{
		AckFloor:  SequencePair{Consumer: 5, Stream: 10},
		Delivered: SequencePair{Consumer: 30, Stream: 40},
	}
}

// TestDetail01 (shape — Inferable: no): byte 0 of the encoding is the
// file-store magic byte.
func TestDetail01(t *testing.T) {
	out := encodeConsumerState(ecsBase())
	if len(out) == 0 || out[0] != magic {
		t.Fatalf("byte 0 = %d, want the file-store magic byte", out[0])
	}
}

// TestDetail02 (shape — Inferable: no): byte 1 is the version-2 marker.
func TestDetail02(t *testing.T) {
	out := encodeConsumerState(ecsBase())
	if len(out) < 2 || out[1] != newVersion {
		t.Fatalf("byte 1 = %d, want the v2 marker", out[1])
	}
}

// TestDetail03 (partially): after the 2-byte header the encoder writes, in
// order, uvarints for ack-floor consumer seq, ack-floor stream seq,
// delivered consumer seq, delivered stream seq, then the pending-map length.
// The derivable part — the five-field uvarint run — is asserted in order.
func TestDetail03(t *testing.T) {
	st := ecsBase()
	st.Pending = map[uint64]*Pending{15: {Sequence: 8, Timestamp: time.Now().UnixNano()}}
	out := encodeConsumerState(st)

	off := hdrLen
	if got := ecsUvarint(t, out, &off); got != 5 {
		t.Fatalf("field 1 (ack-floor consumer) = %d, want 5", got)
	}
	if got := ecsUvarint(t, out, &off); got != 10 {
		t.Fatalf("field 2 (ack-floor stream) = %d, want 10", got)
	}
	if got := ecsUvarint(t, out, &off); got != 30 {
		t.Fatalf("field 3 (delivered consumer) = %d, want 30", got)
	}
	if got := ecsUvarint(t, out, &off); got != 40 {
		t.Fatalf("field 4 (delivered stream) = %d, want 40", got)
	}
	if got := ecsUvarint(t, out, &off); got != 1 {
		t.Fatalf("field 5 (pending length) = %d, want 1", got)
	}
}

// TestDetail04 (yes): when the pending map is empty the encoder does not
// write a base timestamp — the field after the zero pending length is the
// redelivered length, and the buffer ends there.
func TestDetail04(t *testing.T) {
	out := encodeConsumerState(ecsBase()) // both maps nil

	off := hdrLen
	for i := 0; i < 4; i++ {
		ecsUvarint(t, out, &off)
	}
	if got := ecsUvarint(t, out, &off); got != 0 {
		t.Fatalf("pending length = %d, want 0", got)
	}
	if got := ecsUvarint(t, out, &off); got != 0 {
		t.Fatalf("field after empty pending = %d, want 0 (redelivered length, not a timestamp)", got)
	}
	if off != len(out) {
		t.Fatalf("%d trailing bytes after the two empty-map lengths", len(out)-off)
	}
}

// TestDetail05 (doc): a non-empty pending map writes a signed-varint base
// timestamp equal to now rounded to the second, then the records.
func TestDetail05(t *testing.T) {
	st := ecsBase()
	st.Pending = map[uint64]*Pending{15: {Sequence: 8, Timestamp: time.Now().UnixNano()}}

	before := time.Now().Round(time.Second).Unix()
	out := encodeConsumerState(st)
	after := time.Now().Round(time.Second).Unix()

	off := hdrLen
	for i := 0; i < 4; i++ {
		ecsUvarint(t, out, &off)
	}
	if got := ecsUvarint(t, out, &off); got != 1 {
		t.Fatalf("pending length = %d, want 1", got)
	}
	base := ecsVarint(t, out, &off)
	if base < before-1 || base > after+1 {
		t.Fatalf("base timestamp = %d, want ~%d (now, second resolution)", base, before)
	}
}

// TestDetail06 (shape — Inferable: no): each pending record is three varints
// — stream-seq delta, consumer-seq delta, inverted-seconds timestamp — which
// the remaining decoder must reconstruct exactly.
func TestDetail06(t *testing.T) {
	// Fixed second-aligned timestamp: the inverted-seconds field round-trips
	// it exactly regardless of what the encoder picks as its base.
	ts := time.Unix(1700000000, 0).UnixNano()
	st := ecsBase()
	st.Pending = map[uint64]*Pending{
		25: {Sequence: 15, Timestamp: ts},
		30: {Sequence: 17, Timestamp: ts + int64(time.Second)},
	}
	out := encodeConsumerState(st)

	dec, err := decodeConsumerState(out)
	if err != nil {
		t.Fatalf("decodeConsumerState: %v", err)
	}
	if !reflect.DeepEqual(dec.Pending, st.Pending) {
		t.Fatalf("pending round-trip = %+v, want %+v", dec.Pending, st.Pending)
	}
	// Shape: three varints per record between the base timestamp and the
	// redelivered length.
	off := hdrLen
	for i := 0; i < 4; i++ {
		ecsUvarint(t, out, &off)
	}
	ecsUvarint(t, out, &off) // pending len
	ecsVarint(t, out, &off)  // base ts
	for i := 0; i < len(st.Pending); i++ {
		ecsUvarint(t, out, &off)
		ecsUvarint(t, out, &off)
		ecsVarint(t, out, &off)
	}
	if got := ecsUvarint(t, out, &off); got != 0 {
		t.Fatalf("redelivered length = %d, want 0", got)
	}
	if off != len(out) {
		t.Fatalf("%d trailing bytes", len(out)-off)
	}
}

// TestDetail07 (doc): the redelivered-map length uvarint is always written,
// even when zero.
func TestDetail07(t *testing.T) {
	out := encodeConsumerState(ecsBase())
	if len(out) == 0 {
		t.Fatal("empty encoding")
	}
	// Empty state: 2-byte header, four floor uvarints, pending len,
	// redelivered len — the last byte is the redelivered count.
	off := hdrLen
	for i := 0; i < 4; i++ {
		ecsUvarint(t, out, &off)
	}
	ecsUvarint(t, out, &off) // pending len
	ecsUvarint(t, out, &off) // redelivered len — must exist
	if off != len(out) {
		t.Fatalf("encoding does not end after the redelivered length: %d extra bytes", len(out)-off)
	}
}

// TestDetail08 (partially): each redelivered record is two uvarints —
// stream-seq delta then the raw redelivery count — asserted via the decoder
// round-trip and a structural parse.
func TestDetail08(t *testing.T) {
	st := ecsBase()
	st.Redelivered = map[uint64]uint64{15: 3, 18: 9}
	out := encodeConsumerState(st)

	dec, err := decodeConsumerState(out)
	if err != nil {
		t.Fatalf("decodeConsumerState: %v", err)
	}
	if !reflect.DeepEqual(dec.Redelivered, st.Redelivered) {
		t.Fatalf("redelivered round-trip = %+v, want %+v", dec.Redelivered, st.Redelivered)
	}
	off := hdrLen
	for i := 0; i < 4; i++ {
		ecsUvarint(t, out, &off)
	}
	ecsUvarint(t, out, &off) // pending len (0)
	if got := ecsUvarint(t, out, &off); got != 2 {
		t.Fatalf("redelivered length = %d, want 2", got)
	}
	for i := 0; i < 2; i++ {
		ecsUvarint(t, out, &off)
		ecsUvarint(t, out, &off)
	}
	if off != len(out) {
		t.Fatalf("%d trailing bytes", len(out)-off)
	}
}

// TestDetail09 (yes): map iteration order is Go map order — decoders must not
// assume a sort. Asserted by round-tripping a multi-entry map: every entry
// survives regardless of emission order.
func TestDetail09(t *testing.T) {
	ts := time.Unix(1700000000, 0).UnixNano()
	st := ecsBase()
	st.Pending = map[uint64]*Pending{
		11: {Sequence: 6, Timestamp: ts},
		12: {Sequence: 7, Timestamp: ts},
		40: {Sequence: 9, Timestamp: ts},
		35: {Sequence: 8, Timestamp: ts},
	}
	st.Redelivered = map[uint64]uint64{11: 1, 40: 2, 19: 5, 30: 4}
	out := encodeConsumerState(st)

	dec, err := decodeConsumerState(out)
	if err != nil {
		t.Fatalf("decodeConsumerState: %v", err)
	}
	if !reflect.DeepEqual(dec.Pending, st.Pending) || !reflect.DeepEqual(dec.Redelivered, st.Redelivered) {
		t.Fatalf("round-trip lost entries: pending=%+v redelivered=%+v", dec.Pending, dec.Redelivered)
	}
}

// TestDetail10 (yes): the returned slice is truncated to the bytes actually
// written — a full structural parse must consume it exactly.
func TestDetail10(t *testing.T) {
	ts := time.Unix(1700000000, 0).UnixNano()
	st := ecsBase()
	st.Pending = map[uint64]*Pending{25: {Sequence: 15, Timestamp: ts}}
	st.Redelivered = map[uint64]uint64{20: 4}
	out := encodeConsumerState(st)

	off := hdrLen
	for i := 0; i < 4; i++ {
		ecsUvarint(t, out, &off)
	}
	np := int(ecsUvarint(t, out, &off))
	if np > 0 {
		ecsVarint(t, out, &off)
	}
	for i := 0; i < np; i++ {
		ecsUvarint(t, out, &off)
		ecsUvarint(t, out, &off)
		ecsVarint(t, out, &off)
	}
	nr := int(ecsUvarint(t, out, &off))
	for i := 0; i < nr; i++ {
		ecsUvarint(t, out, &off)
		ecsUvarint(t, out, &off)
	}
	if off != len(out) {
		t.Fatalf("parse consumed %d of %d bytes — buffer not truncated to written", off, len(out))
	}
}

// TestDetail11 (partially): with both maps empty the encoding fits inside the
// seqsHdrSize stack buffer — the derivable consequence of the committed
// small-buffer path.
func TestDetail11(t *testing.T) {
	st := &ConsumerState{
		AckFloor:  SequencePair{Consumer: 1 << 60, Stream: 1 << 60},
		Delivered: SequencePair{Consumer: 1<<60 + 7, Stream: 1<<60 + 9},
	}
	out := encodeConsumerState(st)
	if len(out) > seqsHdrSize {
		t.Fatalf("empty-map encoding = %d bytes, exceeds seqsHdrSize=%d", len(out), seqsHdrSize)
	}
	if len(out) == 0 {
		t.Fatal("empty encoding")
	}
}
