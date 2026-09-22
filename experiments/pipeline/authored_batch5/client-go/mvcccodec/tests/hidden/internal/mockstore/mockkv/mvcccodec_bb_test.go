package mocktikv

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"example.internal/kvstore/v2/util/codec"
)

// Hidden suite for unit mvcccodec. One TestDetailNN per DETAILS.md line.

func bbLockFixture() *mvccLock {
	return &mvccLock{
		startTS:     11,
		primary:     []byte("prim"),
		value:       []byte("val"),
		op:          kvrpcpb.Op_Put,
		ttl:         33,
		forUpdateTS: 44,
		txnSize:     55,
		minCommitTS: 66,
	}
}

// TestDetail01: mvccLock marshals its eight fields and unmarshals them back
// — shape assertion: round-trip preserves all fields; encoding is
// deterministic; each field participates (changing one changes the bytes).
func TestDetail01(t *testing.T) {
	l := bbLockFixture()
	data, err := l.MarshalBinary()
	if err != nil || len(data) == 0 {
		t.Fatalf("marshal: %v %d", err, len(data))
	}
	again, _ := l.MarshalBinary()
	if !bytes.Equal(data, again) {
		t.Fatalf("marshal not deterministic")
	}
	var got mvccLock
	if err := got.UnmarshalBinary(data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if l.startTS != got.startTS || !bytes.Equal(l.primary, got.primary) ||
		!bytes.Equal(l.value, got.value) || l.op != got.op || l.ttl != got.ttl ||
		l.forUpdateTS != got.forUpdateTS || l.txnSize != got.txnSize ||
		l.minCommitTS != got.minCommitTS {
		t.Fatalf("round-trip mismatch: %+v vs %+v", l, &got)
	}
	// Each committed field participates in the encoding.
	for i, mutate := range []func(*mvccLock){
		func(x *mvccLock) { x.startTS++ },
		func(x *mvccLock) { x.primary = append(x.primary, 0) },
		func(x *mvccLock) { x.value = append(x.value, 0) },
		func(x *mvccLock) { x.op = kvrpcpb.Op_Del },
		func(x *mvccLock) { x.ttl++ },
		func(x *mvccLock) { x.forUpdateTS++ },
		func(x *mvccLock) { x.txnSize++ },
		func(x *mvccLock) { x.minCommitTS++ },
	} {
		m := bbLockFixture()
		mutate(m)
		md, _ := m.MarshalBinary()
		if bytes.Equal(data, md) {
			t.Fatalf("field %d not encoded", i)
		}
	}
}

// TestDetail02: mvccValue marshals its four fields and unmarshals them
// back — round-trip shape assertion.
func TestDetail02(t *testing.T) {
	v := mvccValue{valueType: typePut, startTS: 7, commitTS: 9, value: []byte("vv")}
	data, err := v.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got mvccValue
	if err := got.UnmarshalBinary(data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v.valueType != got.valueType || v.startTS != got.startTS ||
		v.commitTS != got.commitTS || !bytes.Equal(v.value, got.value) {
		t.Fatalf("round-trip mismatch: %+v vs %+v", v, got)
	}
	// A delete-type value with empty payload still round-trips.
	d := mvccValue{valueType: typeDelete, startTS: 7, commitTS: 9}
	dd, err := d.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal delete: %v", err)
	}
	var gotD mvccValue
	if err := gotD.UnmarshalBinary(dd); err != nil {
		t.Fatalf("unmarshal delete: %v", err)
	}
	if gotD.valueType != typeDelete || gotD.startTS != 7 || gotD.commitTS != 9 {
		t.Fatalf("delete round-trip: %+v", gotD)
	}
}

// bbFailWriter always errors.
type bbFailWriter struct{}

func (bbFailWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// TestDetail03: marshalHelper latches the FIRST error and every later
// write/read is a no-op.
func TestDetail03(t *testing.T) {
	mh := &marshalHelper{}
	mh.WriteNumber(bbFailWriter{}, uint64(1))
	if mh.err == nil {
		t.Fatalf("expected latched write error")
	}
	// After the latch, writes are no-ops even to a healthy writer.
	buf := &bytes.Buffer{}
	mh.WriteNumber(buf, uint64(2))
	mh.WriteSlice(buf, []byte("x"))
	if buf.Len() != 0 {
		t.Fatalf("write after error not a no-op: %d bytes", buf.Len())
	}
	// And reads are no-ops — input is not consumed.
	in := bytes.NewBuffer([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9})
	var n uint64
	var s []byte
	mh.ReadNumber(in, &n)
	mh.ReadSlice(in, &s)
	if in.Len() != 9 || n != 0 || s != nil {
		t.Fatalf("read after error not a no-op: %v %v %v", in.Len(), n, s)
	}
	// First error stays latched even after a second failure.
	mh2 := &marshalHelper{}
	mh2.ReadNumber(bytes.NewBuffer(nil), &n) // EOF on empty
	firstErr := mh2.err
	if firstErr == nil {
		t.Fatalf("expected latched read error")
	}
	mh2.WriteNumber(bbFailWriter{}, uint64(1))
	if mh2.err != firstErr {
		t.Fatalf("first error not latched")
	}
}

// TestDetail04: WriteSlice frames with uvarint and ReadSlice reads exactly
// sz bytes, rejecting slices declared larger than the cap.
func TestDetail04(t *testing.T) {
	buf := &bytes.Buffer{}
	mh := &marshalHelper{}
	mh.WriteSlice(buf, []byte("abc"))
	mh.WriteSlice(buf, []byte("defgh"))
	if mh.err != nil {
		t.Fatalf("WriteSlice: %v", mh.err)
	}
	var a, b []byte
	mh.ReadSlice(buf, &a)
	mh.ReadSlice(buf, &b)
	if mh.err != nil {
		t.Fatalf("ReadSlice: %v", mh.err)
	}
	if string(a) != "abc" || string(b) != "defgh" {
		t.Fatalf("slice round-trip: %q %q", a, b)
	}
	// uvarint framing is derivable: a frame declaring more than buffered
	// fails (exactly-sz reads via io.ReadFull).
	mh2 := &marshalHelper{}
	hdr := make([]byte, binary.MaxVarintLen64)
	hdr = hdr[:binary.PutUvarint(hdr, 100)]
	short := bytes.NewBuffer(append(hdr, 'x', 'y'))
	var s []byte
	mh2.ReadSlice(short, &s)
	if mh2.err == nil {
		t.Fatalf("short frame did not error")
	}
	// A declared length above the cap errors even with payload present.
	mh3 := &marshalHelper{}
	over := make([]byte, binary.MaxVarintLen64)
	over = over[:binary.PutUvarint(over, 11<<20)]
	over = append(over, make([]byte, 11<<20)...)
	mh3.ReadSlice(bytes.NewBuffer(over), &s)
	if mh3.err == nil {
		t.Fatalf("oversized slice did not error")
	}
}

// bbShortWriter writes at most n bytes per call.
type bbShortWriter struct {
	n   int
	got bytes.Buffer
}

func (w *bbShortWriter) Write(p []byte) (int, error) {
	if len(p) > w.n {
		p = p[:w.n]
	}
	return w.got.Write(p)
}

// TestDetail05: writeFull loops Write until all bytes are out.
func TestDetail05(t *testing.T) {
	w := &bbShortWriter{n: 3}
	in := []byte("0123456789")
	if err := writeFull(w, in); err != nil {
		t.Fatalf("writeFull: %v", err)
	}
	if !bytes.Equal(w.got.Bytes(), in) {
		t.Fatalf("writeFull delivered %q", w.got.Bytes())
	}
}

// TestDetail06: NewMvccKey encodes via codec.EncodeBytes and is empty for
// empty input; Raw decodes and panics on malformed data.
func TestDetail06(t *testing.T) {
	if len(NewMvccKey(nil)) != 0 {
		t.Fatalf("empty key did not encode empty")
	}
	k := []byte("hello")
	mk := NewMvccKey(k)
	if !bytes.Equal([]byte(mk), codec.EncodeBytes(nil, k)) {
		t.Fatalf("NewMvccKey != EncodeBytes(nil, k)")
	}
	if !bytes.Equal(mk.Raw(), k) {
		t.Fatalf("Raw != original: %q", mk.Raw())
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("Raw on malformed key did not panic")
			}
		}()
		MvccKey([]byte{0xFF, 0xFF, 0xFF}).Raw()
	}()
}
