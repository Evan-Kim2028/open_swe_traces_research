// Hidden black-box property suite for the seqset unit.
// Drives only the exported API in server/avl (api.md): SequenceSet methods,
// Union, EncodeLen, Encode, Decode, and the documented error values.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package avl_test

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"testing"

	avl "example.internal/msgkit/v2/server/avl"
)

const bbSsHiddenSeed = 20260919

func bbSsSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbSsHiddenSeed
}

func bbSsRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbSsSeed()))
}

func bbSsMembers(ss *avl.SequenceSet) []uint64 {
	var out []uint64
	ss.Range(func(v uint64) bool { out = append(out, v); return true })
	return out
}

// Detail 1: inserting a duplicate leaves the size unchanged.
func TestDetail01_DuplicateInsert(t *testing.T) {
	rng := bbSsRng(t)
	var ss avl.SequenceSet
	for i := 0; i < 60; i++ {
		v := uint64(rng.Intn(300))
		ss.Insert(v)
		before := ss.Size()
		ss.Insert(v)
		ss.Insert(v)
		if ss.Size() != before {
			t.Fatalf("iter %d: duplicate insert grew size %d->%d", i, before, ss.Size())
		}
	}
}

// Detail 2: fixed 2048-entry windows; node count = non-empty windows.
func TestDetail02_WindowNodeCounting(t *testing.T) {
	rng := bbSsRng(t)
	var ss avl.SequenceSet
	windows := map[uint64]int{}
	for i := 0; i < 120; i++ {
		v := uint64(rng.Intn(8192))
		ss.Insert(v)
		windows[v/2048]++
	}
	if ss.Nodes() != len(windows) {
		t.Fatalf("nodes=%d want %d distinct windows", ss.Nodes(), len(windows))
	}
	// Fill a window partially and confirm it is one node regardless of fill.
	var one avl.SequenceSet
	n := 1 + rng.Intn(2048)
	for i := 0; i < n; i++ {
		one.Insert(uint64(i))
	}
	if one.Nodes() != 1 {
		t.Fatalf("partial window: nodes=%d want 1", one.Nodes())
	}
	if one.Size() != n {
		t.Fatalf("partial window: size=%d want %d", one.Size(), n)
	}
	// Two far-apart seqs -> 2 nodes.
	var two avl.SequenceSet
	two.Insert(0)
	two.Insert(4096)
	if two.Nodes() != 2 {
		t.Fatalf("two windows: nodes=%d want 2", two.Nodes())
	}
	// Boundary: 2047 and 2048 are different windows.
	var b avl.SequenceSet
	b.Insert(2047)
	b.Insert(2048)
	if b.Nodes() != 2 {
		t.Fatalf("boundary: nodes=%d want 2", b.Nodes())
	}
}

// Detail 3: membership is exact across sparse windows.
func TestDetail03_SparseMembership(t *testing.T) {
	rng := bbSsRng(t)
	var ss avl.SequenceSet
	present := map[uint64]bool{}
	for i := 0; i < 80; i++ {
		v := uint64(rng.Intn(1 << 16))
		ss.Insert(v)
		present[v] = true
	}
	// Sample a wide range including gaps.
	for i := 0; i < 400; i++ {
		v := uint64(rng.Intn(1 << 16))
		if ss.Exists(v) != present[v] {
			t.Fatalf("Exists(%d)=%v want %v", v, ss.Exists(v), present[v])
		}
	}
	// Window edges.
	for _, v := range []uint64{0, 1, 2047, 2048, 2049, 4095, 4096} {
		if ss.Exists(v) != present[v] {
			t.Fatalf("edge Exists(%d)=%v want %v", v, ss.Exists(v), present[v])
		}
	}
}

// Detail 4: Heights() returns the root's left and right children heights
// (0 for missing), not the root's own height.
func TestDetail04_RootChildHeights(t *testing.T) {
	var ss avl.SequenceSet
	// Single element -> single root node with no children -> (0,0).
	ss.Insert(100)
	l, r := ss.Heights()
	if l != 0 || r != 0 {
		t.Fatalf("single-node set heights=(%d,%d) want (0,0)", l, r)
	}
	// Two windows -> root + one child; one side is 1, other 0.
	var two avl.SequenceSet
	two.Insert(0)
	two.Insert(3000)
	l, r = two.Heights()
	if !((l == 1 && r == 0) || (l == 0 && r == 1)) {
		t.Fatalf("two-node set heights=(%d,%d) want one 1 one 0", l, r)
	}
	// Empty set -> (0,0).
	var e avl.SequenceSet
	if l, r := e.Heights(); l != 0 || r != 0 {
		t.Fatalf("empty heights=(%d,%d) want (0,0)", l, r)
	}
}

// Detail 5: Delete reports prior presence; deleting absent seqs — inside or
// outside existing windows — changes nothing and reports false.
func TestDetail05_DeleteReturnValue(t *testing.T) {
	rng := bbSsRng(t)
	var ss avl.SequenceSet
	present := map[uint64]bool{}
	for i := 0; i < 60; i++ {
		v := uint64(rng.Intn(5000))
		ss.Insert(v)
		present[v] = true
	}
	sizeBefore := ss.Size()
	nodesBefore := ss.Nodes()
	// Delete absent values.
	for i := 0; i < 100; i++ {
		v := uint64(rng.Intn(6000))
		if present[v] {
			continue
		}
		if ss.Delete(v) {
			t.Fatalf("Delete(%d) reported true for absent value", v)
		}
	}
	if ss.Size() != sizeBefore || ss.Nodes() != nodesBefore {
		t.Fatalf("absent deletes changed set: size %d->%d nodes %d->%d",
			sizeBefore, ss.Size(), nodesBefore, ss.Nodes())
	}
	// Delete present values reports true exactly once.
	for v := range present {
		if !ss.Delete(v) {
			t.Fatalf("Delete(%d) reported false for present value", v)
		}
		if ss.Delete(v) {
			t.Fatalf("second Delete(%d) reported true", v)
		}
	}
	if ss.Size() != 0 {
		t.Fatalf("size=%d after deleting all", ss.Size())
	}
}

// Detail 6: emptying a window releases its node; size 0 -> fully reset
// (no root, zero nodes).
func TestDetail06_NodeRelease(t *testing.T) {
	var ss avl.SequenceSet
	// Two windows.
	for i := 0; i < 10; i++ {
		ss.Insert(uint64(i))      // window 0
		ss.Insert(uint64(3000+i)) // window 1
	}
	if ss.Nodes() != 2 {
		t.Fatalf("nodes=%d want 2", ss.Nodes())
	}
	for i := 0; i < 10; i++ {
		ss.Delete(uint64(i))
	}
	if ss.Nodes() != 1 {
		t.Fatalf("nodes=%d after emptying window 0, want 1", ss.Nodes())
	}
	for i := 0; i < 10; i++ {
		ss.Delete(uint64(3000 + i))
	}
	if ss.Nodes() != 0 || ss.Size() != 0 {
		t.Fatalf("after full delete: nodes=%d size=%d want 0/0", ss.Nodes(), ss.Size())
	}
	// Fully reset: state queries yield empties.
	if mn, mx, n := ss.State(); mn != 0 || mx != 0 || n != 0 {
		t.Fatalf("State after reset=(%d,%d,%d) want (0,0,0)", mn, mx, n)
	}
	if l, r := ss.Heights(); l != 0 || r != 0 {
		t.Fatalf("Heights after reset=(%d,%d)", l, r)
	}
}

// Detail 7: Range visits every element in strictly ascending order and
// stops the moment the callback returns false, after delivering that element.
func TestDetail07_AscendingEarlyStop(t *testing.T) {
	rng := bbSsRng(t)
	var ss avl.SequenceSet
	var want []uint64
	seen := map[uint64]bool{}
	for len(want) < 80 {
		v := uint64(rng.Intn(1 << 14))
		if !seen[v] {
			seen[v] = true
			want = append(want, v)
			ss.Insert(v)
		}
	}
	sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })
	var got []uint64
	ss.Range(func(v uint64) bool { got = append(got, v); return true })
	if len(got) != len(want) {
		t.Fatalf("Range delivered %d want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("Range[%d]=%d want %d (ascending)", i, got[i], want[i])
		}
	}
	// Early stop: the stopping element IS delivered.
	stop := 1 + rng.Intn(len(want))
	got = got[:0]
	ss.Range(func(v uint64) bool {
		got = append(got, v)
		return len(got) < stop
	})
	if len(got) != stop {
		t.Fatalf("early stop delivered %d want %d", len(got), stop)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("early Range[%d]=%d want %d", i, got[i], want[i])
		}
	}
	// Callback returning false on first element delivers exactly one.
	got = got[:0]
	ss.Range(func(v uint64) bool { got = append(got, v); return false })
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("stop-first delivered %v want [%d]", got, want[0])
	}
}

// Detail 8: MinMax on empty is (0,0); State yields (0,0,0).
func TestDetail08_EmptyMinMaxState(t *testing.T) {
	var ss avl.SequenceSet
	if mn, mx := ss.MinMax(); mn != 0 || mx != 0 {
		t.Fatalf("empty MinMax=(%d,%d) want (0,0)", mn, mx)
	}
	if mn, mx, n := ss.State(); mn != 0 || mx != 0 || n != 0 {
		t.Fatalf("empty State=(%d,%d,%d) want (0,0,0)", mn, mx, n)
	}
	// Non-empty sanity.
	ss.Insert(50)
	ss.Insert(10)
	if mn, mx := ss.MinMax(); mn != 10 || mx != 50 {
		t.Fatalf("MinMax=(%d,%d) want (10,50)", mn, mx)
	}
	if _, _, n := ss.State(); n != 2 {
		t.Fatalf("State count=%d want 2", n)
	}
}

// Detail 9: Clone produces a deep copy equal in size and node count;
// cloning a nil pointer yields nil.
func TestDetail09_DeepClone(t *testing.T) {
	rng := bbSsRng(t)
	var ss avl.SequenceSet
	for i := 0; i < 100; i++ {
		ss.Insert(uint64(rng.Intn(10000)))
	}
	cl := ss.Clone()
	if cl == nil {
		t.Fatal("Clone returned nil")
	}
	if cl.Size() != ss.Size() || cl.Nodes() != ss.Nodes() {
		t.Fatalf("clone size=%d nodes=%d want %d/%d", cl.Size(), cl.Nodes(), ss.Size(), ss.Nodes())
	}
	// Deep: mutations to clone don't affect original.
	for _, v := range bbSsMembers(cl)[:10] {
		cl.Delete(v)
	}
	if cl.Size() == ss.Size() {
		t.Fatal("clone delete affected original size")
	}
	// Nil clone -> nil.
	var nilSS *avl.SequenceSet
	if got := nilSS.Clone(); got != nil {
		t.Fatalf("nil Clone=%p want nil", got)
	}
}

// Detail 10: two empty sets are equal (including nil receiver vs empty);
// equality is membership only; differing sizes never equal.
func TestDetail10_EqualitySemantics(t *testing.T) {
	rng := bbSsRng(t)
	var a, b avl.SequenceSet
	if !a.Equal(&b) || !b.Equal(&a) {
		t.Fatal("two empty sets must be equal")
	}
	var nilSS *avl.SequenceSet
	if !nilSS.Equal(&b) {
		t.Fatal("nil receiver vs empty set must be equal")
	}
	// Same members, different insertion order.
	vals := []uint64{}
	for len(vals) < 50 {
		v := uint64(rng.Intn(5000))
		vals = append(vals, v)
	}
	for _, v := range vals {
		a.Insert(v)
	}
	for i := len(vals) - 1; i >= 0; i-- {
		b.Insert(vals[i])
	}
	if !a.Equal(&b) || !b.Equal(&a) {
		t.Fatal("insertion order must not matter")
	}
	// Differing size never equal.
	b.Insert(999999)
	if a.Equal(&b) || b.Equal(&a) {
		t.Fatal("differing sizes must not be equal")
	}
}

// Detail 11: free Union() with no args -> nil; otherwise holds every member
// of inputs; method form merges argument's members into receiver.
func TestDetail11_UnionForms(t *testing.T) {
	rng := bbSsRng(t)
	if got := avl.Union(); got != nil {
		t.Fatalf("Union() = %p want nil", got)
	}
	var a, b, c avl.SequenceSet
	members := func(ss *avl.SequenceSet) map[uint64]bool {
		m := map[uint64]bool{}
		for _, v := range bbSsMembers(ss) {
			m[v] = true
		}
		return m
	}
	for i := 0; i < 40; i++ {
		a.Insert(uint64(rng.Intn(3000)))
		b.Insert(uint64(rng.Intn(3000)))
		c.Insert(uint64(rng.Intn(3000)))
	}
	u := avl.Union(&a, &b, &c)
	if u == nil {
		t.Fatal("Union(a,b,c) returned nil")
	}
	um := members(u)
	for _, v := range bbSsMembers(&a) {
		if !um[v] {
			t.Fatalf("union missing a-member %d", v)
		}
	}
	for _, v := range bbSsMembers(&b) {
		if !um[v] {
			t.Fatalf("union missing b-member %d", v)
		}
	}
	for _, v := range bbSsMembers(&c) {
		if !um[v] {
			t.Fatalf("union missing c-member %d", v)
		}
	}
	// Method form merges into receiver.
	var d avl.SequenceSet
	d.Insert(123456)
	d.Union(&a, &b)
	dm := members(&d)
	if !dm[123456] {
		t.Fatal("method union lost receiver member")
	}
	for _, v := range bbSsMembers(&a) {
		if !dm[v] {
			t.Fatalf("method union missing a-member %d", v)
		}
	}
	for _, v := range bbSsMembers(&b) {
		if !dm[v] {
			t.Fatalf("method union missing b-member %d", v)
		}
	}
}

// Detail 12: SetInitialMin on empty creates root window with base exactly
// that minimum even when unaligned; non-empty -> not-empty error; later
// inserts window relative to the unaligned base.
func TestDetail12_InitialMin(t *testing.T) {
	rng := bbSsRng(t)
	var ss avl.SequenceSet
	m := 1 + uint64(rng.Intn(2047)) // unaligned base in (0,2048).
	if err := ss.SetInitialMin(m); err != nil {
		t.Fatalf("SetInitialMin(%d): %v", m, err)
	}
	// Root window base observable through Encode: first node base == m.
	enc := ss.Encode(nil)
	if len(enc) < 10+266 {
		t.Fatalf("encoded len=%d too short for one node", len(enc))
	}
	base := binary.LittleEndian.Uint64(enc[10:18])
	if base != m {
		t.Fatalf("first node base=%d want %d", base, m)
	}
	// Values in [m, m+2048) stay in the same window.
	ss.Insert(m)
	ss.Insert(m + 2047)
	if ss.Nodes() != 1 {
		t.Fatalf("nodes=%d after inserts in window, want 1", ss.Nodes())
	}
	// m-1 lands in a different window.
	ss.Insert(m - 1)
	if ss.Nodes() != 2 {
		t.Fatalf("nodes=%d after m-1 insert, want 2", ss.Nodes())
	}
	// Non-empty -> ErrSetNotEmpty.
	var nonEmpty avl.SequenceSet
	nonEmpty.Insert(5)
	if err := nonEmpty.SetInitialMin(100); !errors.Is(err, avl.ErrSetNotEmpty) {
		t.Fatalf("SetInitialMin non-empty: %v want ErrSetNotEmpty", err)
	}
}

// Detail 13: snapshot layout — byte 22, byte 2, LE u32 node count, LE u32
// size, then per node pre-order a u64 base, 32 u64 buckets, u16 height;
// encoded length is 10 + 266*nodes; caller buffer reused when capacity
// suffices.
func TestDetail13_EncodeLayout(t *testing.T) {
	rng := bbSsRng(t)
	var ss avl.SequenceSet
	for i := 0; i < 100; i++ {
		ss.Insert(uint64(rng.Intn(10000)))
	}
	enc := ss.Encode(nil)
	if enc[0] != 22 {
		t.Fatalf("magic byte=%d want 22", enc[0])
	}
	if enc[1] != 2 {
		t.Fatalf("version byte=%d want 2", enc[1])
	}
	nodes := binary.LittleEndian.Uint32(enc[2:6])
	size := binary.LittleEndian.Uint32(enc[6:10])
	if int(nodes) != ss.Nodes() {
		t.Fatalf("header nodes=%d want %d", nodes, ss.Nodes())
	}
	if int(size) != ss.Size() {
		t.Fatalf("header size=%d want %d", size, ss.Size())
	}
	if len(enc) != 10+266*ss.Nodes() {
		t.Fatalf("enc len=%d want %d", len(enc), 10+266*ss.Nodes())
	}
	if ss.EncodeLen() != len(enc) {
		t.Fatalf("EncodeLen=%d want %d", ss.EncodeLen(), len(enc))
	}
	// Node records: u64 base then 32 u64 buckets then u16 height.
	off := 10
	for i := 0; i < ss.Nodes(); i++ {
		base := binary.LittleEndian.Uint64(enc[off : off+8])
		if base%2048 != 0 {
			t.Fatalf("node %d base=%d not window-aligned", i, base)
		}
		off += 8 + 32*8 + 2
	}
	// Caller buffer reuse.
	buf := make([]byte, 0, ss.EncodeLen()+64)
	out := ss.Encode(buf)
	if len(out) != ss.EncodeLen() {
		t.Fatalf("Encode into caller buf len=%d want %d", len(out), ss.EncodeLen())
	}
	if &out[0] != &buf[:1][0] {
		t.Fatal("Encode did not reuse caller buffer with sufficient capacity")
	}
	// Members recoverable by decoding.
	dec, n, err := avl.Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if n != len(enc) {
		t.Fatalf("bytes read=%d want %d", n, len(enc))
	}
	if !dec.Equal(&ss) {
		t.Fatal("decoded set not Equal to original")
	}
}

// Detail 14: Decode <10 bytes or first byte != 22 -> bad-encoding error and
// -1 bytes read; version 1 or 2 selects layout, else bad-version + -1.
func TestDetail14_DecodeRejects(t *testing.T) {
	for _, n := range []int{0, 1, 5, 9} {
		_, got, err := avl.Decode(make([]byte, n))
		if !errors.Is(err, avl.ErrBadEncoding) || got != -1 {
			t.Fatalf("decode %d bytes: read=%d err=%v want -1/ErrBadEncoding", n, got, err)
		}
	}
	// Bad magic.
	buf := make([]byte, 10)
	buf[0] = 21
	buf[1] = 2
	_, got, err := avl.Decode(buf)
	if !errors.Is(err, avl.ErrBadEncoding) || got != -1 {
		t.Fatalf("bad magic: read=%d err=%v want -1/ErrBadEncoding", got, err)
	}
	// Bad version.
	buf[0] = 22
	buf[1] = 9
	_, got, err = avl.Decode(buf)
	if !errors.Is(err, avl.ErrBadVersion) || got != -1 {
		t.Fatalf("bad version: read=%d err=%v want -1/ErrBadVersion", got, err)
	}
}

// Detail 15: v2 decoder bounds node count by remaining/266 — oversized
// counts fail ErrBadEncoding; nodes placed in tree order without
// rebalancing.
func TestDetail15_V2NodeCountBound(t *testing.T) {
	var ss avl.SequenceSet
	ss.Insert(1)
	enc := ss.Encode(nil)
	// Inflate the node-count field beyond what the buffer can hold.
	for _, fake := range []uint32{2, 10, 1000, 1 << 20} {
		bad := append([]byte(nil), enc...)
		binary.LittleEndian.PutUint32(bad[2:6], fake)
		if _, _, err := avl.Decode(bad); !errors.Is(err, avl.ErrBadEncoding) {
			t.Fatalf("node count %d: err=%v want ErrBadEncoding", fake, err)
		}
	}
}

// Detail 16: v1 decoder reads nodes of 64 buckets each, expands set bits
// through normal insertion, skips a trailing 2-byte height per node, and
// fails ErrBadEncoding when the decoded size disagrees with the header.
func TestDetail16_V1Decode(t *testing.T) {
	rng := bbSsRng(t)
	// Build a v1 encoding: byte22 byte1 u32 nodes u32 size, per node
	// u64 base + 64 u64 buckets + u16 height.
	build := func(members []uint64, declared uint32) []byte {
		// Group by window.
		windows := map[uint64][]uint64{}
		var order []uint64
		for _, v := range members {
			w := v / 2048
			if _, ok := windows[w]; !ok {
				order = append(order, w)
			}
			windows[w] = append(windows[w], v)
		}
		sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
		var buf []byte
		hdr := make([]byte, 10)
		hdr[0] = 22
		hdr[1] = 1
		binary.LittleEndian.PutUint32(hdr[2:6], uint32(len(order)))
		binary.LittleEndian.PutUint32(hdr[6:10], declared)
		buf = append(buf, hdr...)
		for _, w := range order {
			node := make([]byte, 8+64*8+2)
			binary.LittleEndian.PutUint64(node[0:8], w*2048)
			for _, v := range windows[w] {
				off := v - w*2048
				bucket := node[8+(off/64)*8:]
				binary.LittleEndian.PutUint64(bucket, binary.LittleEndian.Uint64(bucket)|(1<<(off%64)))
			}
			buf = append(buf, node...)
		}
		return buf
	}
	members := []uint64{}
	seen := map[uint64]bool{}
	for len(members) < 50 {
		v := uint64(rng.Intn(8000))
		if !seen[v] {
			seen[v] = true
			members = append(members, v)
		}
	}
	buf := build(members, uint32(len(members)))
	ss, _, err := avl.Decode(buf)
	if err != nil {
		t.Fatalf("v1 decode: %v", err)
	}
	for _, v := range members {
		if !ss.Exists(v) {
			t.Fatalf("v1-decoded set missing %d", v)
		}
	}
	if ss.Size() != len(members) {
		t.Fatalf("v1 size=%d want %d", ss.Size(), len(members))
	}
	// Declared size mismatch -> ErrBadEncoding.
	bad := build(members, uint32(len(members))+1)
	if _, _, err := avl.Decode(bad); !errors.Is(err, avl.ErrBadEncoding) {
		t.Fatalf("v1 size mismatch: err=%v want ErrBadEncoding", err)
	}
}

// Detail 17: insert/delete keep the tree height-balanced: root children's
// heights stay within |l-r| <= 1 after every operation.
func TestDetail17_BalanceInvariant(t *testing.T) {
	rng := bbSsRng(t)
	var ss avl.SequenceSet
	check := func(i int) {
		l, r := ss.Heights()
		if l-r > 1 || r-l > 1 {
			t.Fatalf("op %d: heights=(%d,%d) unbalanced", i, l, r)
		}
	}
	// Sequential inserts force worst-case growth patterns.
	for i := 0; i < 300; i++ {
		ss.Insert(uint64(i * 2048))
		check(i)
	}
	// Random inserts.
	for i := 0; i < 300; i++ {
		ss.Insert(uint64(rng.Intn(1<<18) * 2048))
		check(300 + i)
	}
	// Random deletes keep balance.
	for i := 0; i < 400; i++ {
		ss.Delete(uint64(rng.Intn(1<<18) * 2048))
		check(600 + i)
	}
}
