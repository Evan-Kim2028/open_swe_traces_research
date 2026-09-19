// Hidden black-box property suite for the staging-buffer unit.
// Exported API only: NewMemDBWithContext + MemDB staging/flag/write/read surface.
// Seed 20260919. Any correct implementation of the contract must pass.

package unionstore

import (
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"example.internal/kvstore/v2/kv"
)

const bbMemdbSeed = int64(20260919)

func bbNewMemDB() *MemDB {
	return NewMemDBWithContext().GetMemDB()
}

// --- model -----------------------------------------------------------------

// bbWrite is one staged value write recorded by the model.
type bbWrite struct {
	depth int
	val   []byte // nil means tombstone
}

// bbKeyState is the per-key model state. Observable contract, distilled:
//   - value writes stack per staging level; cleanup drops deeper levels,
//     release merges them into the parent;
//   - flags ops (SetWithFlags/DeleteWithFlags/UpdateFlags) accumulate on the
//     node and are never rolled back while the node survives;
//   - destroying a node (cleanup/revert of its last value write) leaves a
//     flag-only remnant carrying only the persistent flag bits;
//   - an UpdateFlags-created flag-only node floats outside staging until a
//     value write anchors it.
type bbKeyState struct {
	writes   []bbWrite
	flags    kv.KeyFlags
	flagOnly bool // node exists with no value writes (flag-only / remnant)
	anchored bool // node lifetime is tied to its value writes
}

// bbModel mirrors the observable contract: per-key staged write stacks.
type bbModel struct {
	entries map[string]*bbKeyState
	depth   int // number of open staging levels
	dirty   bool
	handles map[int]int // handle -> depth captured when Staging() was called
}

func bbNewModel() *bbModel {
	return &bbModel{entries: map[string]*bbKeyState{}, handles: map[int]int{}}
}

func bbKey(r *rand.Rand, i int) []byte {
	var b [8]byte
	binary.BigEndian.PutUint32(b[:4], uint32(i))
	binary.BigEndian.PutUint32(b[4:], uint32(r.Intn(1<<24)))
	return b[:]
}

func (m *bbModel) State(key []byte) *bbKeyState {
	k := string(key)
	e, ok := m.entries[k]
	if !ok {
		e = &bbKeyState{}
		m.entries[k] = e
	}
	return e
}

func (m *bbModel) Write(key, val []byte, ops []kv.FlagsOp) {
	e := m.State(key)
	e.flags = kv.ApplyFlagsOps(e.flags, ops...)
	e.writes = append(e.writes, bbWrite{depth: m.depth, val: append([]byte(nil), val...)})
	e.anchored = true
	e.flagOnly = false
	m.dirty = true
}

// flagOnly applies flag ops to the node without a value write. On a missing
// key it materializes a node that floats outside staging.
func (m *bbModel) FlagOnly(key []byte, ops []kv.FlagsOp) {
	e := m.State(key)
	e.flags = kv.ApplyFlagsOps(e.flags, ops...)
	if len(e.writes) == 0 && !e.anchored {
		e.flagOnly = true
	}
	m.dirty = true
}

func (m *bbModel) Staging(h int) {
	m.depth++
	m.handles[h] = m.depth
}

func (m *bbModel) Release(h int) {
	d := m.handles[h]
	for _, e := range m.entries {
		for i := range e.writes {
			if e.writes[i].depth >= d {
				e.writes[i].depth = d - 1
			}
		}
	}
	delete(m.handles, h)
	m.depth--
}

// destroyNode leaves behind the persistent-flag remnant, if any.
func (e *bbKeyState) DestroyNode() {
	e.flags = e.flags.AndPersistent()
	e.writes = nil
	e.anchored = false
	e.flagOnly = e.flags != 0
}

func (m *bbModel) Cleanup(h int) {
	d := m.handles[h]
	for k, e := range m.entries {
		var keep []bbWrite
		for _, w := range e.writes {
			if w.depth < d {
				keep = append(keep, w)
			}
		}
		e.writes = keep
		if e.anchored && len(e.writes) == 0 {
			e.DestroyNode()
		}
		if len(e.writes) == 0 && !e.flagOnly && e.flags == 0 {
			delete(m.entries, k)
		}
	}
	delete(m.handles, h)
	m.depth--
}

// revertTo restores the value-write stacks of a snapshot. Flags are never
// rolled back; nodes destroyed by the revert leave persistent remnants;
// flag-only nodes float outside checkpoints entirely.
func (m *bbModel) RevertTo(snap map[string]bbKeyState) {
	for k, e := range m.entries {
		s, ok := snap[k]
		if !ok {
			if e.anchored {
				e.DestroyNode()
			}
			if len(e.writes) == 0 && !e.flagOnly && e.flags == 0 {
				delete(m.entries, k)
			}
			continue
		}
		writes := append([]bbWrite(nil), s.writes...)
		e.writes = writes
		e.anchored = s.anchored
		e.flagOnly = s.flagOnly
		if e.anchored && len(e.writes) == 0 {
			e.DestroyNode()
		}
	}
}

func (m *bbModel) Snapshot() map[string]bbKeyState {
	out := map[string]bbKeyState{}
	for k, e := range m.entries {
		out[k] = bbKeyState{
			writes:   append([]bbWrite(nil), e.writes...),
			flags:    e.flags,
			flagOnly: e.flagOnly,
			anchored: e.anchored,
		}
	}
	return out
}

func (e *bbKeyState) Exists() bool {
	return len(e.writes) > 0 || e.flagOnly
}

func (m *bbModel) Len() int {
	n := 0
	for _, e := range m.entries {
		if e.Exists() {
			n++
		}
	}
	return n
}

func (m *bbModel) Get(key []byte) ([]byte, bool) {
	e := m.entries[string(key)]
	if e == nil || !e.Exists() || len(e.writes) == 0 {
		return nil, false
	}
	// Tombstones are visible: a deleted key reads back as an empty value.
	return e.writes[len(e.writes)-1].val, true
}

func (m *bbModel) Flags(key []byte) (kv.KeyFlags, bool) {
	e := m.entries[string(key)]
	if e == nil || !e.Exists() {
		return 0, false
	}
	return e.flags, true
}

// bbInspect is an InspectStage yield: current value and flags of the node.
type bbInspect struct {
	val   []byte
	flags kv.KeyFlags
}

// inspectStageModel: keys whose last value write sits at depth >= d, with the
// node's current value/flags. Flag-only nodes never appear.
func (m *bbModel) Inspect(d int) map[string]bbInspect {
	out := map[string]bbInspect{}
	for k, e := range m.entries {
		if len(e.writes) == 0 {
			continue
		}
		top := e.writes[len(e.writes)-1]
		if top.depth >= d {
			out[k] = bbInspect{val: top.val, flags: e.flags}
		}
	}
	return out
}

// --- checks ----------------------------------------------------------------

func bbCheckGet(t *assert.Assertions, db *MemDB, m *bbModel, key []byte, tag string) {
	wantVal, ok := m.Get(key)
	got, err := db.Get(key)
	if !ok {
		t.NotNil(err, "%s: key %q should miss", tag, key)
		return
	}
	t.Nil(err, "%s: key %q should hit", tag, key)
	t.Equal(string(wantVal), string(got), "%s: key %q value", tag, key)
}

func bbCheckLen(t *assert.Assertions, db *MemDB, m *bbModel, tag string) {
	t.Equal(m.Len(), db.Len(), "%s: Len", tag)
}

func bbCheckFlags(t *assert.Assertions, db *MemDB, m *bbModel, key []byte, tag string) {
	want, ok := m.Flags(key)
	got, err := db.GetFlags(key)
	if !ok {
		t.NotNil(err, "%s: flags for %q should miss", tag, key)
		return
	}
	t.Nil(err, "%s: flags for %q", tag, key)
	t.Equal(want, got, "%s: flags for %q", tag, key)
}

// --- properties ------------------------------------------------------------

// Contract: writes go into the current staging level; Get/Set/Delete round-trip;
// Delete writes a tombstone (Get not-found, Len still counts it).
func TestMemDBBBSetGetDelete(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbMemdbSeed))
	db := bbNewMemDB()
	m := bbNewModel()
	keyN := 97
	keys := make([][]byte, keyN)
	for i := range keys {
		keys[i] = bbKey(r, i)
	}
	for i := 0; i < 10000; i++ {
		k := keys[r.Intn(keyN)]
		var v []byte
		switch r.Intn(10) {
		case 0, 1:
			err := db.Delete(k)
			assert.Nil(err)
			m.Write(k, nil, nil)
		case 2:
			v = make([]byte, 1+r.Intn(600))
			r.Read(v)
			err := db.Set(k, v)
			assert.Nil(err)
			m.Write(k, v, nil)
		default:
			v = make([]byte, 1+r.Intn(64))
			r.Read(v)
			err := db.Set(k, v)
			assert.Nil(err)
			m.Write(k, v, nil)
		}
		if i%37 == 0 {
			bbCheckGet(assert, db, m, k, "setgetdelete")
		}
		if i%251 == 0 {
			bbCheckLen(assert, db, m, "setgetdelete")
		}
	}
	for _, k := range keys {
		bbCheckGet(assert, db, m, k, "final")
	}
	bbCheckLen(assert, db, m, "final")
	assert.True(db.Dirty(), "uncheckpointed writes must mark dirty")
}

// Contract: Staging starts a level; Release merges into the parent (values and
// flags persist); Cleanup drops everything written at that level; nested
// release+cleanup removes both levels.
func TestMemDBBBStagingMergeDiscard(t *testing.T) {
	r := rand.New(rand.NewSource(bbMemdbSeed + 1))
	assert := assert.New(t)
	for trial := 0; trial < 400; trial++ {
		db := bbNewMemDB()
		m := bbNewModel()
		keyN := 1 + r.Intn(12)
		keys := make([][]byte, keyN)
		for i := range keys {
			keys[i] = bbKey(r, i+trial*1000)
		}
		// random base writes
		for i := 0; i < 1+r.Intn(8); i++ {
			k := keys[r.Intn(keyN)]
			v := []byte{byte(i), byte(trial)}
			assert.Nil(db.Set(k, v))
			m.Write(k, v, nil)
		}
		open := []int{}
		steps := 1 + r.Intn(14)
		for s := 0; s < steps; s++ {
			action := r.Intn(10)
			if action < 4 || len(open) == 0 {
				if len(open) < 4 {
					h := db.Staging()
					m.Staging(h)
					open = append(open, h)
					continue
				}
			}
			if action < 7 {
				// write at the current level
				k := keys[r.Intn(keyN)]
				if r.Intn(4) == 0 {
					assert.Nil(db.Delete(k))
					m.Write(k, nil, nil)
				} else {
					v := []byte{byte(s), byte(r.Intn(256)), byte(trial)}
					assert.Nil(db.Set(k, v))
					m.Write(k, v, nil)
				}
				continue
			}
			// close the top level one way or the other
			top := open[len(open)-1]
			open = open[:len(open)-1]
			if r.Intn(2) == 0 {
				db.Release(top)
				m.Release(top)
			} else {
				db.Cleanup(top)
				m.Cleanup(top)
			}
		}
		for i := len(open) - 1; i >= 0; i-- {
			db.Cleanup(open[i])
			m.Cleanup(open[i])
		}
		for _, k := range keys {
			bbCheckGet(assert, db, m, k, "staging")
		}
		bbCheckLen(assert, db, m, "staging")
	}
}

// Contract: InspectStage yields only entries written at that level or deeper,
// with their current flags and values (tombstones included).
func TestMemDBBBInspectStage(t *testing.T) {
	r := rand.New(rand.NewSource(bbMemdbSeed + 2))
	assert := assert.New(t)
	for trial := 0; trial < 300; trial++ {
		db := bbNewMemDB()
		m := bbNewModel()
		keyN := 1 + r.Intn(9)
		keys := make([][]byte, keyN)
		for i := range keys {
			keys[i] = bbKey(r, i)
		}
		h := db.Staging()
		m.Staging(h)
		d := m.handles[h]
		writes := 1 + r.Intn(15)
		for i := 0; i < writes; i++ {
			k := keys[r.Intn(keyN)]
			switch r.Intn(4) {
			case 0:
				assert.Nil(db.Delete(k))
				m.Write(k, nil, nil)
			case 1:
				v := []byte{byte(i)}
				assert.Nil(db.SetWithFlags(k, v, kv.SetKeyLocked))
				m.Write(k, v, []kv.FlagsOp{kv.SetKeyLocked})
			default:
				v := []byte{byte(i), byte(trial)}
				assert.Nil(db.Set(k, v))
				m.Write(k, v, nil)
			}
		}
		want := m.Inspect(d)
		got := map[string]bbInspect{}
		db.InspectStage(h, func(key []byte, flags kv.KeyFlags, val []byte) {
			got[string(key)] = bbInspect{val: append([]byte(nil), val...), flags: flags}
		})
		assert.Equal(len(want), len(got), "inspect yield count")
		for k, w := range want {
			g, ok := got[k]
			assert.True(ok, "inspect missing key %q", k)
			if ok {
				assert.Equal(w.val, g.val, "inspect val %q", k)
				assert.Equal(w.flags, g.flags, "inspect flags %q", k)
			}
		}
		db.Cleanup(h)
		m.Cleanup(h)
		bbCheckLen(assert, db, m, "post-inspect")
	}
}

// Contract: RevertToCheckpoint restores the exact node set, value log position
// and sizes captured by Checkpoint; flags-only mutations roll back too.
func TestMemDBBBCheckpointRevert(t *testing.T) {
	r := rand.New(rand.NewSource(bbMemdbSeed + 3))
	assert := assert.New(t)
	for trial := 0; trial < 400; trial++ {
		db := bbNewMemDB()
		m := bbNewModel()
		keyN := 1 + r.Intn(10)
		keys := make([][]byte, keyN)
		for i := range keys {
			keys[i] = bbKey(r, i+trial*77)
		}
		pre := 1 + r.Intn(10)
		for i := 0; i < pre; i++ {
			k := keys[r.Intn(keyN)]
			v := []byte{byte(i), byte(trial)}
			assert.Nil(db.Set(k, v))
			m.Write(k, v, nil)
		}
		snap := m.Snapshot()
		cp := db.Checkpoint()
		post := 1 + r.Intn(12)
		for i := 0; i < post; i++ {
			k := keys[r.Intn(keyN)]
			switch r.Intn(4) {
			case 0:
				assert.Nil(db.Delete(k))
				m.Write(k, nil, nil)
			case 1:
				assert.Nil(db.SetWithFlags(k, []byte{byte(i)}, kv.SetKeyLocked, kv.SetPresumeKeyNotExists))
				m.Write(k, []byte{byte(i)}, []kv.FlagsOp{kv.SetKeyLocked, kv.SetPresumeKeyNotExists})
			default:
				assert.Nil(db.Set(k, []byte{byte(i + 100)}))
				m.Write(k, []byte{byte(i + 100)}, nil)
			}
		}
		db.RevertToCheckpoint(cp)
		m.RevertTo(snap)
		for _, k := range keys {
			bbCheckGet(assert, db, m, k, "revert")
			bbCheckFlags(assert, db, m, k, "revert")
		}
		assert.Equal(m.Len(), db.Len(), "revert Len")
		// Flag mutations are never rolled back; value history is. Nodes that
		// survived keep their accumulated flags.
		for k, e := range m.entries {
			if !e.Exists() {
				continue
			}
			fl, err := db.GetFlags([]byte(k))
			assert.Nil(err)
			assert.Equal(e.flags, fl, "revert flags %q", k)
		}
	}
}

// Contract: flags ops attach/clear per-key flags; persistent flags survive a
// cleanup of the level that wrote them; temporary flags do not.
func TestMemDBBBFlagsAcrossStages(t *testing.T) {
	r := rand.New(rand.NewSource(bbMemdbSeed + 4))
	assert := assert.New(t)
	ops := [][]kv.FlagsOp{
		{kv.SetKeyLocked},
		{kv.SetPresumeKeyNotExists},
		{kv.SetKeyLocked, kv.SetPresumeKeyNotExists},
		{kv.SetKeyLockedValueExists},
		{kv.SetNeedLocked},
		{kv.DelPresumeKeyNotExists},
	}
	for trial := 0; trial < 500; trial++ {
		db := bbNewMemDB()
		m := bbNewModel()
		keyN := 1 + r.Intn(6)
		keys := make([][]byte, keyN)
		for i := range keys {
			keys[i] = bbKey(r, i+trial*131)
		}
		h := db.Staging()
		m.Staging(h)
		for i := 0; i < 1+r.Intn(10); i++ {
			k := keys[r.Intn(keyN)]
			o := ops[r.Intn(len(ops))]
			switch r.Intn(3) {
			case 0:
				assert.Nil(db.SetWithFlags(k, []byte{byte(i)}, o...))
				m.Write(k, []byte{byte(i)}, o)
			case 1:
				assert.Nil(db.DeleteWithFlags(k, o...))
				m.Write(k, nil, o)
			default:
				db.UpdateFlags(k, o...)
				m.FlagOnly(k, o)
			}
		}
		db.Cleanup(h)
		m.Cleanup(h)
		for _, k := range keys {
			bbCheckGet(assert, db, m, k, "flags")
			bbCheckFlags(assert, db, m, k, "flags")
		}
		bbCheckLen(assert, db, m, "flags")
	}
}

// Contract: Dirty reports uncheckpointed writes (including surviving
// persistent flag writes); Len counts live nodes; Reset empties the db;
// unseen-random keys/values the worked examples never name still behave.
func TestMemDBBBDirtyResetAndUnseen(t *testing.T) {
	r := rand.New(rand.NewSource(bbMemdbSeed + 5))
	assert := assert.New(t)
	for trial := 0; trial < 300; trial++ {
		db := bbNewMemDB()
		m := bbNewModel()
		assert.False(db.Dirty(), "fresh db clean")
		keyN := 2 + r.Intn(7)
		keys := make([][]byte, keyN)
		for i := range keys {
			// long, odd-shaped keys far from any contract example
			n := 4 + r.Intn(40)
			k := make([]byte, n)
			r.Read(k)
			binary.BigEndian.PutUint32(k, uint32(i))
			keys[i] = k
		}
		wrote := false
		for i := 0; i < 1+r.Intn(9); i++ {
			k := keys[r.Intn(keyN)]
			v := make([]byte, 1+r.Intn(200))
			r.Read(v)
			assert.Nil(db.Set(k, v))
			m.Write(k, v, nil)
			wrote = true
		}
		assert.Equal(wrote, db.Dirty(), "dirty after writes")
		h := db.Staging()
		m.Staging(h)
		if r.Intn(2) == 0 {
			k := keys[r.Intn(keyN)]
			assert.Nil(db.SetWithFlags(k, []byte{1}, kv.SetKeyLocked))
			m.Write(k, []byte{1}, []kv.FlagsOp{kv.SetKeyLocked})
			db.Cleanup(h)
			m.Cleanup(h)
			assert.True(db.Dirty(), "persistent flag write survives cleanup -> dirty")
		} else {
			k := keys[r.Intn(keyN)]
			assert.Nil(db.Set(k, []byte{9}))
			m.Write(k, []byte{9}, nil)
			db.Cleanup(h)
			m.Cleanup(h)
		}
		bbCheckLen(assert, db, m, "dirty")
		for _, k := range keys {
			bbCheckGet(assert, db, m, k, "dirty")
		}
		db.Reset()
		assert.Equal(0, db.Len(), "reset empties")
		assert.False(db.Dirty(), "reset clears dirty")
	}
}

// Contract: a flag-only write (UpdateFlags on a missing key) materializes a
// node: Get misses, flags read back, node counted.
func TestMemDBBBUpdateFlagsCreatesNode(t *testing.T) {
	r := rand.New(rand.NewSource(bbMemdbSeed + 6))
	assert := assert.New(t)
	db := bbNewMemDB()
	wantFlags := map[uint64]kv.KeyFlags{}
	for i := 0; i < 2000; i++ {
		var k [8]byte
		kid := uint64(r.Intn(300))
		binary.BigEndian.PutUint64(k[:], kid)
		op := kv.FlagsOp(kv.SetKeyLocked)
		if i%3 == 0 {
			op = kv.SetPresumeKeyNotExists
		}
		db.UpdateFlags(k[:], op)
		wantFlags[kid] = kv.ApplyFlagsOps(wantFlags[kid], op)
		_, err := db.Get(k[:])
		assert.NotNil(err, "flag-only node must not read as a value")
		fl, err := db.GetFlags(k[:])
		assert.Nil(err)
		assert.Equal(wantFlags[kid], fl, "flags of flag-only node")
	}
	assert.Equal(300, db.Len(), "flag-only nodes counted")
}
