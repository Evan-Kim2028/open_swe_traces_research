package locate

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/pingcap/kvproto/pkg/metapb"
)

const bucketSeed = 20260918
const bucketCases = 10000

func bbKey(rng *rand.Rand) []byte {
	n := rng.Intn(8)
	b := make([]byte, n)
	rng.Read(b)
	return b
}

func bbLoc(start, end []byte, splits [][]byte, ver uint64) *KeyLocation {
	keys := make([][]byte, 0, len(splits)+2)
	keys = append(keys, append([]byte(nil), start...))
	keys = append(keys, splits...)
	keys = append(keys, append([]byte(nil), end...))
	return &KeyLocation{
		StartKey: start,
		EndKey:   end,
		Buckets:  &metapb.Buckets{Version: ver, Keys: keys},
	}
}

func TestBucketContainsRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(bucketSeed))
	for i := 0; i < bucketCases; i++ {
		a := bbKey(rng)
		b := bbKey(rng)
		if bytes.Compare(a, b) > 0 {
			a, b = b, a
		}
		if bytes.Equal(a, b) {
			b = append(append([]byte(nil), a...), 0xff)
		}
		loc := &KeyLocation{StartKey: a, EndKey: b}
		mid := append([]byte(nil), a...)
		if loc.Contains(mid) != true {
			t.Fatalf("case %d start must be inside", i)
		}
		if len(b) > 0 && loc.Contains(b) {
			t.Fatalf("case %d exclusive end leaked", i)
		}
		before := []byte{0}
		if bytes.Compare(before, a) < 0 && loc.Contains(before) {
			t.Fatalf("case %d key before start", i)
		}
	}
}

func TestLocateBucketProperties(t *testing.T) {
	rng := rand.New(rand.NewSource(bucketSeed))
	for i := 0; i < bucketCases; i++ {
		start, end := []byte{byte(rng.Intn(40))}, []byte{byte(80 + rng.Intn(40))}
		if bytes.Compare(start, end) >= 0 {
			end = []byte{start[0] + 10}
		}
		var splits [][]byte
		cur := start[0] + 1
		for cur < end[0] && len(splits) < 4 {
			splits = append(splits, []byte{cur})
			cur += byte(1 + rng.Intn(8))
		}
		ver := uint64(rng.Uint32())
		loc := bbLoc(start, end, splits, ver)
		if loc.GetBucketVersion() != ver {
			t.Fatalf("case %d version %d want %d", i, loc.GetBucketVersion(), ver)
		}
		probe := []byte{byte(start[0] + (end[0]-start[0])/2)}
		if !loc.Contains(probe) {
			continue
		}
		bk := loc.LocateBucket(probe)
		if bk == nil {
			t.Fatalf("case %d nil bucket for in-range key %x loc=[%x,%x)", i, probe, start, end)
		}
		if !bk.Contains(probe) {
			t.Fatalf("case %d bucket [%x,%x) misses %x", i, bk.StartKey, bk.EndKey, probe)
		}
		outside := []byte{0}
		if bytes.Compare(outside, start) < 0 {
			if loc.LocateBucket(outside) != nil {
				t.Fatalf("case %d outside key got a bucket", i)
			}
		}
	}
}

func TestBucketContractExamples(t *testing.T) {
	loc := bbLoc([]byte("a"), []byte("z"), [][]byte{[]byte("m")}, 7)
	if loc.GetBucketVersion() != 7 {
		t.Fatalf("version %d", loc.GetBucketVersion())
	}
	if !loc.Contains([]byte("a")) || loc.Contains([]byte("z")) {
		t.Fatal("half-open region interval")
	}
	left := loc.LocateBucket([]byte("b"))
	if left == nil || !left.Contains([]byte("b")) || left.Contains([]byte("m")) {
		t.Fatalf("left bucket %+v", left)
	}
	right := loc.LocateBucket([]byte("m"))
	if right == nil || !right.Contains([]byte("m")) {
		t.Fatalf("right bucket %+v", right)
	}
	if loc.LocateBucket([]byte{0x00}) != nil {
		t.Fatal("key before start must be nil")
	}
	wholeLoc := bbLoc([]byte("a"), []byte("z"), nil, 1)
	whole := wholeLoc.LocateBucket([]byte("k"))
	if whole == nil || !bytes.Equal(whole.StartKey, []byte("a")) || !bytes.Equal(whole.EndKey, []byte("z")) {
		t.Fatalf("no interior splits should cover the whole interval, got %+v", whole)
	}
}

func TestBucketUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bucketSeed + 3))
	for i := 0; i < bucketCases; i++ {
		start := []byte{byte(rng.Intn(50) + 10)}
		end := []byte{byte(int(start[0]) + 5 + rng.Intn(20))}
		loc := bbLoc(start, end, nil, uint64(i+1))
		k := []byte{byte(start[0] + 1)}
		if bytes.Equal(k, []byte("a")) || bytes.Equal(k, []byte("m")) || bytes.Equal(k, []byte("z")) {
			k = []byte{byte(start[0] + 2)}
		}
		if !loc.Contains(k) {
			continue
		}
		bk := loc.LocateBucket(k)
		if bk == nil || !bk.Contains(k) {
			t.Fatalf("unseen %d key %x", i, k)
		}
	}
}
