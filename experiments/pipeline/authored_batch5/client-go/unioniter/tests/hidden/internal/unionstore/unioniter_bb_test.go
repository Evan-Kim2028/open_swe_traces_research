package unionstore

import (
	"testing"
)

// Hidden suite for unit unioniter. One TestDetailNN per DETAILS.md line.

func bbMemWith(t *testing.T, pairs ...[]string) *MemDB {
	t.Helper()
	db := newMemDB()
	for _, kv := range pairs {
		if len(kv) == 1 {
			if err := db.Delete([]byte(kv[0])); err != nil {
				t.Fatalf("delete %q: %v", kv[0], err)
			}
		} else {
			if err := db.Set([]byte(kv[0]), []byte(kv[1])); err != nil {
				t.Fatalf("set %q: %v", kv[0], err)
			}
		}
	}
	return db
}

func bbCollect(t *testing.T, it Iterator) ([]string, []string) {
	t.Helper()
	var keys, vals []string
	for ; it.Valid(); it.Next() {
		keys = append(keys, string(it.Key()))
		vals = append(vals, string(it.Value()))
	}
	return keys, vals
}

func bbForward(t *testing.T, dirty, snap *MemDB) *UnionIter {
	t.Helper()
	di, err := dirty.Iter(nil, nil)
	if err != nil {
		t.Fatalf("dirty iter: %v", err)
	}
	si, err := snap.Iter(nil, nil)
	if err != nil {
		t.Fatalf("snapshot iter: %v", err)
	}
	u, err := NewUnionIter(di, si, false)
	if err != nil {
		t.Fatalf("NewUnionIter: %v", err)
	}
	return u
}

func bbEq(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// TestDetail01: the union iterator merges dirty and snapshot children in
// key order.
func TestDetail01(t *testing.T) {
	snap := bbMemWith(t, []string{"a", "1"}, []string{"c", "3"}, []string{"e", "5"})
	dirty := bbMemWith(t, []string{"b", "2"}, []string{"d", "4"})
	u := bbForward(t, dirty, snap)
	defer u.Close()
	keys, _ := bbCollect(t, u)
	bbEq(t, keys, []string{"a", "b", "c", "d", "e"})
}

// TestDetail02: empty-value dirty entries are tombstones — an equal-key
// tombstone masks the snapshot record; a tombstone ahead of the snapshot
// key leaves the snapshot record alone.
func TestDetail02(t *testing.T) {
	snap := bbMemWith(t, []string{"a", "1"}, []string{"b", "2"}, []string{"c", "3"})
	dirty := bbMemWith(t, []string{"b"})
	u := bbForward(t, dirty, snap)
	defer u.Close()
	keys, _ := bbCollect(t, u)
	bbEq(t, keys, []string{"a", "c"})

	// Tombstone ordered before the snapshot keys consumes only itself.
	dirty2 := bbMemWith(t, []string{"0"}, []string{"z"})
	u = bbForward(t, dirty2, snap)
	defer u.Close()
	keys, _ = bbCollect(t, u)
	bbEq(t, keys, []string{"a", "b", "c"})
}

// TestDetail03: on equal keys the dirty record wins and the duplicate is
// not emitted twice.
func TestDetail03(t *testing.T) {
	snap := bbMemWith(t, []string{"a", "1"}, []string{"b", "2"})
	dirty := bbMemWith(t, []string{"a", "9"})
	u := bbForward(t, dirty, snap)
	defer u.Close()
	keys, vals := bbCollect(t, u)
	bbEq(t, keys, []string{"a", "b"})
	bbEq(t, vals, []string{"9", "2"})
}

// TestDetail04: reverse iteration negates the comparison — descending merge,
// tombstones still mask.
func TestDetail04(t *testing.T) {
	snap := bbMemWith(t, []string{"a", "1"}, []string{"c", "3"})
	dirty := bbMemWith(t, []string{"b", "2"})
	di, err := dirty.IterReverse(nil, nil)
	if err != nil {
		t.Fatalf("dirty iter: %v", err)
	}
	si, err := snap.IterReverse(nil, nil)
	if err != nil {
		t.Fatalf("snapshot iter: %v", err)
	}
	u, err := NewUnionIter(di, si, true)
	if err != nil {
		t.Fatalf("NewUnionIter: %v", err)
	}
	defer u.Close()
	keys, _ := bbCollect(t, u)
	bbEq(t, keys, []string{"c", "b", "a"})

	// Reverse tombstone masking (also exercised by the in-tree test).
	snap2 := bbMemWith(t, []string{"1", "1"}, []string{"2", "2"}, []string{"3", "3"})
	dirty2 := bbMemWith(t, []string{"2"})
	di, _ = dirty2.IterReverse(nil, nil)
	si, _ = snap2.IterReverse(nil, nil)
	u, err = NewUnionIter(di, si, true)
	if err != nil {
		t.Fatalf("NewUnionIter: %v", err)
	}
	defer u.Close()
	keys, _ = bbCollect(t, u)
	bbEq(t, keys, []string{"3", "1"})
}

// TestDetail05: tombstone runs are skipped and validity ends only when both
// children are exhausted.
func TestDetail05(t *testing.T) {
	// Dirty tombstone on a shared key plus a live dirty record.
	snap := bbMemWith(t, []string{"a", "1"}, []string{"c", "3"})
	dirty := bbMemWith(t, []string{"a"}, []string{"b", "9"})
	u := bbForward(t, dirty, snap)
	defer u.Close()
	keys, vals := bbCollect(t, u)
	bbEq(t, keys, []string{"b", "c"})
	bbEq(t, vals, []string{"9", "3"})

	// One side empty: still valid, yields the other side.
	u = bbForward(t, bbMemWith(t), snap)
	defer u.Close()
	if !u.Valid() {
		t.Fatalf("empty dirty made iter invalid")
	}
	keys, _ = bbCollect(t, u)
	bbEq(t, keys, []string{"a", "c"})

	// Dirty with only tombstones and an empty snapshot: nothing to yield.
	u = bbForward(t, bbMemWith(t, []string{"a"}), bbMemWith(t))
	defer u.Close()
	if u.Valid() {
		t.Fatalf("tombstone-only dirty over empty snapshot is valid")
	}
}

// TestDetail06: Next advances the producer and re-resolves; Key/Value
// delegate to the current child; Close is clean.
func TestDetail06(t *testing.T) {
	snap := bbMemWith(t, []string{"a", "1"})
	dirty := bbMemWith(t, []string{"b", "2"})
	u := bbForward(t, dirty, snap)
	if !u.Valid() || string(u.Key()) != "a" || string(u.Value()) != "1" {
		t.Fatalf("initial position: key=%q val=%q valid=%v", u.Key(), u.Value(), u.Valid())
	}
	if err := u.Next(); err != nil {
		t.Fatalf("Next: %v", err)
	}
	if string(u.Key()) != "b" || string(u.Value()) != "2" {
		t.Fatalf("after Next: key=%q val=%q", u.Key(), u.Value())
	}
	if err := u.Next(); err != nil {
		t.Fatalf("Next: %v", err)
	}
	if u.Valid() {
		t.Fatalf("still valid after exhaustion")
	}
	u.Close()
}
