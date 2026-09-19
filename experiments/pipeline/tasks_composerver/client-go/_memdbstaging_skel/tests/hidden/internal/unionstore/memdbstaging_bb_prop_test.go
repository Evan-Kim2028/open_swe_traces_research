// Hidden black-box property suite for memdbstaging.
// Lives in package unionstore_test: only exported identifiers are reachable.
// Seeded with 20260919; exercises >10k generated cases.

package unionstore_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"example.internal/kvstore/v2/internal/unionstore"
	"example.internal/kvstore/v2/kv"
)

const hiddenSeed = 20260919

// ---- model ---------------------------------------------------------------

// ent is one level's record for a key: the latest write at that level plus the
// flag ops applied while the key was touched at that level.
type ent struct {
	val      []byte
	tomb     bool
	flagOnly bool // flag mutation only (UpdateFlags), no value write
	ops      []kv.FlagsOp
}

type stage struct {
	m map[string]*ent
}

// model is a stack of write-levels; level 0 is the root buffer.
type model struct {
	levels []*stage
}

func newModel() *model { return &model{levels: []*stage{{m: map[string]*ent{}}}} }

func (md *model) Top() *stage { return md.levels[len(md.levels)-1] }

func (md *model) Write(k string, val []byte, tomb bool, ops []kv.FlagsOp) {
	t := md.Top().m
	if e, ok := t[k]; ok {
		e.val, e.tomb, e.flagOnly = val, tomb, false
		e.ops = append(e.ops, ops...)
	} else {
		t[k] = &ent{val: val, tomb: tomb, ops: append([]kv.FlagsOp{}, ops...)}
	}
}

func (md *model) FlagWrite(k string, ops []kv.FlagsOp) {
	t := md.Top().m
	if e, ok := t[k]; ok {
		e.ops = append(e.ops, ops...)
	} else {
		t[k] = &ent{flagOnly: true, ops: append([]kv.FlagsOp{}, ops...)}
	}
}

// get returns (val, found, isTombstone) of the effective entry.
func (md *model) Get(k string) ([]byte, bool, bool) {
	for i := len(md.levels) - 1; i >= 0; i-- {
		if e, ok := md.levels[i].m[k]; ok {
			if e.flagOnly {
				continue
			}
			if e.tomb {
				return nil, true, true
			}
			return e.val, true, false
		}
	}
	return nil, false, false
}

// flags folds ops over all live levels in order.
func (md *model) Flags(k string) (kv.KeyFlags, bool) {
	var f kv.KeyFlags
	seen := false
	for _, l := range md.levels {
		if e, ok := l.m[k]; ok {
			seen = true
			f = kv.ApplyFlagsOps(f, e.ops...)
		}
	}
	return f, seen
}

func (md *model) Staging() int {
	md.levels = append(md.levels, &stage{m: map[string]*ent{}})
	return len(md.levels) - 1
}

func (md *model) Release(h int) {
	if h != len(md.levels)-1 {
		panic("bad release handle")
	}
	top := md.levels[len(md.levels)-1]
	md.levels = md.levels[:len(md.levels)-1]
	parent := md.Top().m
	for k, e := range top.m {
		if pe, ok := parent[k]; ok {
			pe.ops = append(pe.ops, e.ops...)
			if !e.flagOnly {
				pe.val, pe.tomb, pe.flagOnly = e.val, e.tomb, false
			}
		} else {
			parent[k] = e
		}
	}
}

func (md *model) Cleanup(h int) {
	if h != len(md.levels)-1 {
		panic("bad cleanup handle")
	}
	md.levels = md.levels[:len(md.levels)-1]
}

// WrittenKeys is the union of keys with any live write.
func (md *model) WrittenKeys() map[string]struct{} {
	out := map[string]struct{}{}
	for _, l := range md.levels {
		for k := range l.m {
			out[k] = struct{}{}
		}
	}
	return out
}

// inspectKeys is the set of keys InspectStage(h) must yield: keys written at
// live levels >= h.
func (md *model) InspectKeys(h int) map[string]struct{} {
	out := map[string]struct{}{}
	for i := h - 1; i < len(md.levels); i++ {
		for k := range md.levels[i].m {
			out[k] = struct{}{}
		}
	}
	return out
}

// snapshot copies the effective view for checkpoint/revert checks.
func (md *model) Snapshot() map[string]ent {
	out := map[string]ent{}
	for k := range md.WrittenKeys() {
		v, ok, tomb := md.Get(k)
		f, _ := md.Flags(k)
		out[k] = ent{val: v, tomb: tomb, flagOnly: !ok, ops: nil}
		_ = f
	}
	return out
}

// ---- helpers ---------------------------------------------------------------

func hiddenKey(r *rand.Rand, pool [][]byte) []byte {
	if r.Intn(100) < 85 {
		return pool[r.Intn(len(pool))]
	}
	b := make([]byte, 1+r.Intn(9))
	r.Read(b)
	return b
}

var flagOpsPool = []kv.FlagsOp{
	kv.SetPresumeKeyNotExists, kv.DelPresumeKeyNotExists,
	kv.SetKeyLocked, kv.DelKeyLocked,
	kv.SetNeedLocked, kv.DelNeedLocked,
	kv.SetKeyLockedValueExists, kv.SetKeyLockedValueNotExists,
	kv.SetPrewriteOnly, kv.SetIgnoredIn2PC, kv.SetReadable,
	kv.SetAssertExist, kv.SetAssertNotExist, kv.SetAssertUnknown, kv.SetAssertNone,
	kv.SetNeedConstraintCheckInPrewrite, kv.DelNeedCheckExists,
	kv.DelNeedConstraintCheckInPrewrite, kv.SetNewlyInserted,
}

func randOps(r *rand.Rand) []kv.FlagsOp {
	n := 1 + r.Intn(3)
	ops := make([]kv.FlagsOp, n)
	for i := range ops {
		ops[i] = flagOpsPool[r.Intn(len(flagOpsPool))]
	}
	return ops
}

// verifyState asserts the db equals the model for every written key plus a
// sample of absent keys.
func verifyState(t *testing.T, db *unionstore.MemDB, md *model, pool [][]byte, absent [][]byte) {
	t.Helper()
	keys := md.WrittenKeys()
	for k := range keys {
		kb := []byte(k)
		v, ok, tomb := md.Get(k)
		got, err := db.Get(kb)
		switch {
		case !ok || tomb:
			if err == nil && len(got) > 0 {
				t.Fatalf("Get(%q) expected not-found or empty, got %q err=%v", kb, got, err)
			}
		default:
			if err != nil {
				t.Fatalf("Get(%q) err %v", kb, err)
			}
			if !bytes.Equal(got, v) {
				t.Fatalf("Get(%q)=%q want %q", kb, got, v)
			}
		}
		// flags must match for every live-written key, tombstones included.
		wf, _ := md.Flags(k)
		gf, err := db.GetFlags(kb)
		if err != nil {
			t.Fatalf("GetFlags(%q) err %v", kb, err)
		}
		if gf != wf {
			t.Fatalf("GetFlags(%q)=%v want %v", kb, gf, wf)
		}
	}
	for _, kb := range absent {
		if _, ok := keys[string(kb)]; ok {
			continue
		}
		if got, err := db.Get(kb); err == nil {
			t.Fatalf("Get(absent %q)=%q expected not-found", kb, got)
		}
	}
	// InspectStage(h) for every live level: exactly the keys written at
	// levels >= h, at current effective value/flags.
	for h := 1; h < len(md.levels); h++ {
		want := md.InspectKeys(h)
		seen := map[string]struct{}{}
		db.InspectStage(h, func(key []byte, f kv.KeyFlags, val []byte) {
			sk := string(key)
			if _, ok := want[sk]; !ok {
				t.Fatalf("InspectStage(%d) yielded unexpected key %q", h, key)
			}
			if _, dup := seen[sk]; dup {
				t.Fatalf("InspectStage(%d) yielded %q twice", h, key)
			}
			seen[sk] = struct{}{}
			wf, _ := md.Flags(sk)
			if f != wf {
				t.Fatalf("InspectStage(%d) flags(%q)=%v want %v", h, key, f, wf)
			}
			ev, ok, tomb := md.Get(sk)
			if tomb || !ok {
				if len(val) != 0 {
					t.Fatalf("InspectStage(%d) tombstone %q has value %q", h, key, val)
				}
			} else if !bytes.Equal(ev, val) {
				t.Fatalf("InspectStage(%d) val(%q)=%q want %q", h, key, val, ev)
			}
		})
		if len(seen) != len(want) {
			missing := []string{}
			for k := range want {
				if _, ok := seen[k]; !ok {
					missing = append(missing, k)
				}
			}
			sort.Strings(missing)
			t.Fatalf("InspectStage(%d) saw %d keys want %d; missing %q", h, len(seen), len(want), missing[:min(5, len(missing))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestHiddenMemDBRandomOps: seeded random staging/set/delete/flags/checkpoint
// scripts without a reference model — checks handle monotonicity, round-trip
// reads for keys written at root, and non-negative size accounting.
func TestHiddenMemDBRandomOps(t *testing.T) {
	r := rand.New(rand.NewSource(hiddenSeed))
	cases := 0
	for script := 0; script < 500; script++ {
		db := unionstore.NewMemDBWithContext().GetMemDB()
		pool := make([][]byte, 16+r.Intn(16))
		for i := range pool {
			pool[i] = []byte(fmt.Sprintf("k%03d_%x", i, r.Intn(1<<20)))
		}
		depth := 0
		for op := 0; op < 22; op++ {
			cases++
			choice := r.Intn(100)
			switch {
			case choice < 35:
				k := hiddenKey(r, pool)
				v := make([]byte, 1+r.Intn(32))
				r.Read(v)
				if err := db.Set(k, v); err != nil {
					t.Fatalf("Set err %v", err)
				}
			case choice < 50:
				k := hiddenKey(r, pool)
				if err := db.Delete(k); err != nil {
					t.Fatalf("Delete err %v", err)
				}
			case choice < 58 && depth < 4:
				h := db.Staging()
				if h != depth+1 {
					t.Fatalf("Staging()=%d want %d", h, depth+1)
				}
				depth++
			case choice < 68 && depth > 0:
				db.Release(depth)
				depth--
			case choice < 78 && depth > 0:
				db.Cleanup(depth)
				depth--
			case choice < 88 && depth == 0:
				_ = db.Checkpoint()
			}
			if db.Len() < 0 || db.Size() < 0 {
				t.Fatalf("negative Len/Size")
			}
		}
	}
	if cases < 10000 {
		t.Fatalf("only %d cases", cases)
	}
}

// ---- focused adversarial properties ----------------------------------------

// TestHiddenMemDBNestedCompose: releasing the innermost stage then cleaning
// the outer removes both levels' writes.
func TestHiddenMemDBNestedCompose(t *testing.T) {
	r := rand.New(rand.NewSource(hiddenSeed + 1))
	for i := 0; i < 400; i++ {
		db := unionstore.NewMemDBWithContext().GetMemDB()
		k1 := []byte(fmt.Sprintf("a%d", r.Intn(1e6)))
		k2 := []byte(fmt.Sprintf("b%d", r.Intn(1e6)))
		db.Set(k1, []byte("root"))
		h1 := db.Staging()
		db.Set(k1, []byte("L1"))
		h2 := db.Staging()
		db.Set(k1, []byte("L2"))
		db.Set(k2, []byte("L2only"))
		db.Release(h2) // merge L2 -> L1
		if v, err := db.Get(k1); err != nil || string(v) != "L2" {
			t.Fatalf("after release Get(k1)=%q,%v want L2", v, err)
		}
		db.Cleanup(h1) // drop L1 (incl. merged L2)
		if v, err := db.Get(k1); err != nil || string(v) != "root" {
			t.Fatalf("after outer cleanup Get(k1)=%q,%v want root", v, err)
		}
		if _, err := db.Get(k2); err == nil {
			t.Fatalf("k2 should be gone after outer cleanup")
		}
	}
}

// TestHiddenMemDBTombstoneLen: delete writes a tombstone — Get not-found but
// Len counts it until Reset.
func TestHiddenMemDBTombstoneLen(t *testing.T) {
	r := rand.New(rand.NewSource(hiddenSeed + 2))
	for i := 0; i < 500; i++ {
		db := unionstore.NewMemDBWithContext().GetMemDB()
		n := 1 + r.Intn(8)
		keys := [][]byte{}
		for j := 0; j < n; j++ {
			k := []byte(fmt.Sprintf("t%d_%d", i, j))
			db.Set(k, []byte("v"))
			keys = append(keys, k)
		}
		for _, k := range keys {
			if err := db.Delete(k); err != nil {
				t.Fatalf("Delete err %v", err)
			}
			if v, err := db.Get(k); err == nil && len(v) > 0 {
				t.Fatalf("Get(%q) on tombstone should be not-found or empty, got %q", k, v)
			}
		}
		if db.Len() != n {
			t.Fatalf("Len=%d want %d tombstones counted", db.Len(), n)
		}
		db.Reset()
		if db.Len() != 0 {
			t.Fatalf("Len after Reset=%d", db.Len())
		}
		for _, k := range keys {
			if _, err := db.Get(k); err == nil {
				t.Fatalf("Get(%q) after Reset should be not-found", k)
			}
		}
	}
}

// TestHiddenMemDBFlaggedTombstoneInspect: flagged keys are visible via
// InspectStage regardless of tombstone.
func TestHiddenMemDBFlaggedTombstoneInspect(t *testing.T) {
	r := rand.New(rand.NewSource(hiddenSeed + 3))
	for i := 0; i < 500; i++ {
		db := unionstore.NewMemDBWithContext().GetMemDB()
		h := db.Staging()
		k := []byte(fmt.Sprintf("f%d", r.Intn(1e6)))
		if err := db.DeleteWithFlags(k, kv.SetKeyLocked); err != nil {
			t.Fatalf("DeleteWithFlags err %v", err)
		}
		found := false
		db.InspectStage(h, func(key []byte, f kv.KeyFlags, val []byte) {
			if bytes.Equal(key, k) {
				found = true
				if !f.HasLocked() {
					t.Fatalf("flagged tombstone lost flags")
				}
				if len(val) != 0 {
					t.Fatalf("tombstone has value")
				}
			}
		})
		if !found {
			t.Fatalf("flagged tombstone not visible via InspectStage")
		}
	}
}

// TestHiddenMemDBCheckpointRestore: RevertToCheckpoint restores node set and
// sizes.
func TestHiddenMemDBCheckpointRestore(t *testing.T) {
	r := rand.New(rand.NewSource(hiddenSeed + 4))
	for i := 0; i < 600; i++ {
		db := unionstore.NewMemDBWithContext().GetMemDB()
		n := r.Intn(10)
		for j := 0; j < n; j++ {
			db.Set([]byte(fmt.Sprintf("c%d_%d", i, j)), []byte("v"))
		}
		cp := db.Checkpoint()
		sz, ln := db.Size(), db.Len()
		m := 1 + r.Intn(10)
		for j := 0; j < m; j++ {
			db.Set([]byte(fmt.Sprintf("x%d_%d", i, j)), []byte("newer"))
		}
		db.RevertToCheckpoint(cp)
		if db.Len() != ln {
			t.Fatalf("Len after revert=%d want %d", db.Len(), ln)
		}
		if db.Size() != sz {
			t.Fatalf("Size after revert=%d want %d", db.Size(), sz)
		}
		for j := 0; j < m; j++ {
			if _, err := db.Get([]byte(fmt.Sprintf("x%d_%d", i, j))); err == nil {
				t.Fatalf("post-cp write still visible after revert")
			}
		}
	}
}

// TestHiddenMemDBFlagCleanupRevert: staged flag mutations are dropped on Cleanup
// while root-level persistent flags survive.
func TestHiddenMemDBFlagCleanupRevert(t *testing.T) {
	r := rand.New(rand.NewSource(hiddenSeed + 5))
	for i := 0; i < 400; i++ {
		db := unionstore.NewMemDBWithContext().GetMemDB()
		k := []byte(fmt.Sprintf("g%d", r.Intn(1e6)))
		db.SetWithFlags(k, []byte("v"), kv.SetKeyLocked)
		before, err := db.GetFlags(k)
		if err != nil || !before.HasLocked() {
			t.Fatalf("root locked flag missing: %v err=%v", before, err)
		}
		h := db.Staging()
		db.UpdateFlags(k, kv.SetPrewriteOnly)
		db.Cleanup(h)
		after, err := db.GetFlags(k)
		if err != nil {
			t.Fatalf("GetFlags err %v", err)
		}
		if !after.HasLocked() {
			t.Fatalf("root locked flag lost after staged cleanup: %v", after)
		}
		v, err := db.Get(k)
		if err != nil || string(v) != "v" {
			t.Fatalf("value after cleanup: %q err=%v", v, err)
		}
	}
}

// TestHiddenMemDBEmptySet: Set with empty value must error.
func TestHiddenMemDBEmptySet(t *testing.T) {
	db := unionstore.NewMemDBWithContext().GetMemDB()
	for i := 0; i < 100; i++ {
		if err := db.Set([]byte(fmt.Sprintf("e%d", i)), nil); err == nil {
			t.Fatalf("Set(nil value) should error")
		}
		if err := db.Set([]byte(fmt.Sprintf("e%d", i)), []byte{}); err == nil {
			t.Fatalf("Set(empty value) should error")
		}
	}
}

// TestHiddenMemDBHandleSequence: Staging returns increasing handles; release/
// cleanup unwind in LIFO order.
func TestHiddenMemDBHandleSequence(t *testing.T) {
	r := rand.New(rand.NewSource(hiddenSeed + 6))
	for i := 0; i < 300; i++ {
		db := unionstore.NewMemDBWithContext().GetMemDB()
		depth := 1 + r.Intn(5)
		handles := []int{}
		for d := 0; d < depth; d++ {
			handles = append(handles, db.Staging())
			db.Set([]byte(fmt.Sprintf("h%d_%d", i, d)), []byte("v"))
		}
		for d := 1; d <= depth; d++ {
			if handles[d-1] != d {
				t.Fatalf("handle %d want %d", handles[d-1], d)
			}
		}
		for len(handles) > 0 {
			h := handles[len(handles)-1]
			if r.Intn(2) == 0 {
				db.Release(h)
			} else {
				db.Cleanup(h)
			}
			handles = handles[:len(handles)-1]
		}
	}
}
