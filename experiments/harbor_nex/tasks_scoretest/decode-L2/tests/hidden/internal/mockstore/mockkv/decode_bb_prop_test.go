package mocktikv

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
)

const decodeSeed = 20260918
const decodeCases = 10000

func TestLockBinaryRoundTripProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(decodeSeed))
	ops := []kvrpcpb.Op{kvrpcpb.Op_Put, kvrpcpb.Op_Del, kvrpcpb.Op_Lock}
	for i := 0; i < decodeCases; i++ {
		src := mvccLock{
			startTS:     rng.Uint64(),
			primary:     []byte{byte(rng.Intn(255)), byte(rng.Intn(255)), byte(rng.Intn(255))},
			value:       []byte{byte(rng.Intn(255)), byte(rng.Intn(255))},
			op:          ops[rng.Intn(len(ops))],
			ttl:         rng.Uint64() % 10_000,
			forUpdateTS: rng.Uint64(),
			txnSize:     rng.Uint64() % 1000,
			minCommitTS: rng.Uint64(),
		}
		bin, err := src.MarshalBinary()
		if err != nil {
			t.Fatalf("case %d marshal: %v", i, err)
		}
		var got mvccLock
		if err := got.UnmarshalBinary(bin); err != nil {
			t.Fatalf("case %d unmarshal: %v", i, err)
		}
		if got.startTS != src.startTS || got.ttl != src.ttl || got.op != src.op ||
			got.forUpdateTS != src.forUpdateTS || got.txnSize != src.txnSize ||
			got.minCommitTS != src.minCommitTS ||
			!bytes.Equal(got.primary, src.primary) || !bytes.Equal(got.value, src.value) {
			t.Fatalf("case %d round-trip mismatch", i)
		}
	}
}

func TestValueBinaryRoundTripProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(decodeSeed + 1))
	for i := 0; i < decodeCases; i++ {
		src := mvccValue{
			valueType: mvccValueType(rng.Intn(4)),
			startTS:   rng.Uint64(),
			commitTS:  rng.Uint64(),
			value:     []byte{byte(rng.Intn(255)), byte(rng.Intn(255))},
		}
		bin, err := src.MarshalBinary()
		if err != nil {
			t.Fatalf("case %d marshal: %v", i, err)
		}
		var got mvccValue
		if err := got.UnmarshalBinary(bin); err != nil {
			t.Fatalf("case %d unmarshal: %v", i, err)
		}
		if got.valueType != src.valueType || got.startTS != src.startTS ||
			got.commitTS != src.commitTS || !bytes.Equal(got.value, src.value) {
			t.Fatalf("case %d round-trip mismatch", i)
		}
	}
}

func TestDecodeContractExamples(t *testing.T) {
	l := mvccLock{
		startTS: 47, primary: []byte{'a', 'b', 'c'}, value: []byte{'d', 'e'},
		op: kvrpcpb.Op_Put, ttl: 444, minCommitTS: 666,
	}
	bin, err := l.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var l1 mvccLock
	if err := l1.UnmarshalBinary(bin); err != nil {
		t.Fatal(err)
	}
	if l1.startTS != 47 || l1.ttl != 444 || l1.minCommitTS != 666 ||
		!bytes.Equal(l1.primary, []byte{'a', 'b', 'c'}) || !bytes.Equal(l1.value, []byte{'d', 'e'}) {
		t.Fatalf("lock example round-trip failed: %+v", l1)
	}
	v := mvccValue{valueType: typePut, startTS: 42, commitTS: 55, value: []byte{'d', 'e'}}
	bin, err = v.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var v1 mvccValue
	if err := v1.UnmarshalBinary(bin); err != nil {
		t.Fatal(err)
	}
	if v1.startTS != 42 || v1.commitTS != 55 || v1.valueType != typePut ||
		!bytes.Equal(v1.value, []byte{'d', 'e'}) {
		t.Fatalf("value example round-trip failed: %+v", v1)
	}
}

func TestDecodeUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(decodeSeed + 2))
	src := mvccLock{startTS: rng.Uint64() | 1, primary: []byte("zz"), value: []byte("yy"), ttl: 9}
	bin, err := src.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var got mvccLock
	if err := got.UnmarshalBinary(bin); err != nil {
		t.Fatal(err)
	}
	if got.startTS != src.startTS || !bytes.Equal(got.primary, src.primary) {
		t.Fatalf("unmentioned lock failed: %+v", got)
	}
}
