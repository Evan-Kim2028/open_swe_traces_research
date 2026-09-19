package unionstore

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

const nextSeed = 20260918
const nextCases = 10000

func TestIteratorNextOrderProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(nextSeed))
	for i := 0; i < nextCases; i++ {
		db := newMemDB()
		n := rng.Intn(6) + 2
		want := make([][]byte, 0, n)
		seen := map[string]bool{}
		for len(want) < n {
			k := []byte{byte(rng.Intn(200) + 1)}
			if seen[string(k)] {
				continue
			}
			seen[string(k)] = true
			if err := db.Set(k, k); err != nil {
				t.Fatalf("case %d set: %v", i, err)
			}
			want = append(want, append([]byte(nil), k...))
		}
		for a := 0; a < len(want); a++ {
			for b := a + 1; b < len(want); b++ {
				if bytes.Compare(want[a], want[b]) > 0 {
					want[a], want[b] = want[b], want[a]
				}
			}
		}
		it, err := db.Iter(nil, nil)
		if err != nil {
			t.Fatalf("case %d iter: %v", i, err)
		}
		got := [][]byte{}
		for it.Valid() {
			got = append(got, append([]byte(nil), it.Key()...))
			if !bytes.Equal(it.Value(), it.Key()) {
				t.Fatalf("case %d value mismatch", i)
			}
			if err := it.Next(); err != nil {
				t.Fatalf("case %d next: %v", i, err)
			}
		}
		it.Close()
		if len(got) != len(want) {
			t.Fatalf("case %d len %d want %d", i, len(got), len(want))
		}
		for j := range want {
			if !bytes.Equal(got[j], want[j]) {
				t.Fatalf("case %d order got %q want %q at %d", i, got[j], want[j], j)
			}
		}
	}
}

func TestIteratorNextContractExamples(t *testing.T) {
	db := newMemDB()
	for _, k := range []string{"a", "c", "b"} {
		if err := db.Set([]byte(k), []byte(k)); err != nil {
			t.Fatal(err)
		}
	}
	it, err := db.Iter(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer it.Close()
	var got []string
	for it.Valid() {
		got = append(got, string(it.Key()))
		if err := it.Next(); err != nil {
			t.Fatal(err)
		}
	}
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("example order %v want a b c", got)
	}
}

func TestIteratorNextUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(nextSeed + 1))
	db := newMemDB()
	k := []byte(fmt.Sprintf("n-%d", rng.Intn(1<<20)))
	if err := db.Set(k, []byte("v")); err != nil {
		t.Fatal(err)
	}
	it, err := db.Iter(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer it.Close()
	if !it.Valid() || !bytes.Equal(it.Key(), k) {
		t.Fatalf("unmentioned key not at cursor")
	}
	if err := it.Next(); err != nil {
		t.Fatal(err)
	}
	if it.Valid() {
		t.Fatalf("single-key iterator should be exhausted after one advance")
	}
}
