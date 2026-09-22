package commitgraph

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
)

func mkHash(first, fill byte) plumbing.Hash {
	var b [20]byte
	b[0] = first
	for i := 1; i < 20; i++ {
		b[i] = fill
	}
	h, ok := plumbing.FromBytes(b[:])
	if !ok {
		panic("bad test hash")
	}
	return h
}

type cgFile struct {
	raw       []byte
	nChunks   int
	hashVer   byte
	tocSig    []string
	tocOff    []uint64
	chunkData map[string][]byte
}

func parseCG(t *testing.T, raw []byte) *cgFile {
	t.Helper()
	f := &cgFile{raw: raw, chunkData: map[string][]byte{}}
	if len(raw) < 8 {
		t.Fatal("file shorter than header")
	}
	f.hashVer = raw[5]
	f.nChunks = int(raw[6])
	n := f.nChunks + 1 // terminator entry
	if len(raw) < 8+n*12+20 {
		t.Fatal("file too short for TOC + checksum")
	}
	for i := 0; i < n; i++ {
		e := 8 + i*12
		f.tocSig = append(f.tocSig, string(raw[e:e+4]))
		f.tocOff = append(f.tocOff, binary.BigEndian.Uint64(raw[e+4:e+12]))
	}
	// chunk data spans between consecutive TOC offsets; the terminator
	// entry's offset is the end of chunk data (start of checksum).
	for i := 0; i < n-1; i++ {
		f.chunkData[f.tocSig[i]] = raw[f.tocOff[i]:f.tocOff[i+1]]
	}
	return f
}

func mkIndex(t *testing.T, specs ...*CommitData) (*MemoryIndex, []plumbing.Hash) {
	t.Helper()
	idx := NewMemoryIndex()
	var hs []plumbing.Hash
	for i, cd := range specs {
		h := mkHash(byte(i*0x40+1), byte(i)+1)
		idx.Add(h, cd)
		hs = append(hs, h)
	}
	return idx, hs
}

func encode(t *testing.T, idx Index) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := NewEncoder(&buf).Encode(idx); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.Bytes()
}

// TestDetail01: file opens CGPH + {1, hashVersion, chunkCount, 0}.
func TestDetail01(t *testing.T) {
	idx, _ := mkIndex(t, &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
	f := parseCG(t, encode(t, idx))
	if string(f.raw[:4]) != "CGPH" {
		t.Fatalf("magic = %q, want CGPH", f.raw[:4])
	}
	if f.raw[4] != 1 {
		t.Fatalf("version byte = %d, want 1", f.raw[4])
	}
	if f.hashVer != 1 {
		t.Fatalf("hash version = %d, want 1 for a 20-byte hash", f.hashVer)
	}
	if f.raw[7] != 0 {
		t.Fatalf("reserved header byte = %d, want 0", f.raw[7])
	}
	if f.nChunks != len(f.tocSig)-1 {
		t.Fatalf("header chunk count %d != TOC entries %d", f.nChunks, len(f.tocSig)-1)
	}
}

// TestDetail02: TOC is (4-byte sig + u64 absolute offset) pairs, closed by a
// zero-signature entry carrying the end-of-data offset.
func TestDetail02(t *testing.T) {
	idx, _ := mkIndex(t, &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
	raw := encode(t, idx)
	f := parseCG(t, raw)
	first := f.tocOff[0]
	if first != uint64(8+(f.nChunks+1)*12) {
		t.Fatalf("first chunk offset %d, want header+TOC size %d", first, 8+(f.nChunks+1)*12)
	}
	if f.tocSig[len(f.tocSig)-1] != "\x00\x00\x00\x00" {
		t.Fatalf("terminator TOC sig = %x, want zero", f.tocSig[len(f.tocSig)-1])
	}
	term := f.tocOff[len(f.tocOff)-1]
	if term != uint64(len(raw)-20) {
		t.Fatalf("terminator offset %d != end-of-data %d", term, len(raw)-20)
	}
	for i := 1; i < len(f.tocOff); i++ {
		if f.tocOff[i] < f.tocOff[i-1] {
			t.Fatalf("TOC offsets not ascending at %d", i)
		}
	}
}

// TestDetail03: chunk order is fanout, oid lookup, commit data, then optional
// extra edges, generation data, generation overflow.
func TestDetail03(t *testing.T) {
	idx, _ := mkIndex(t, &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
	f := parseCG(t, encode(t, idx))
	want := []string{"OIDF", "OIDL", "CDAT", "\x00\x00\x00\x00"}
	if len(f.tocSig) != len(want) {
		t.Fatalf("TOC sigs = %q, want %q", f.tocSig, want)
	}
	for i := range want {
		if f.tocSig[i] != want[i] {
			t.Fatalf("TOC[%d] = %q, want %q", i, f.tocSig[i], want[i])
		}
	}
}

// TestDetail04: fanout is 256 cumulative counts over the bytewise-sorted hash
// list — fanout[i] counts hashes with first byte <= i.
func TestDetail04(t *testing.T) {
	idx := NewMemoryIndex()
	idx.Add(mkHash(0x00, 1), &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
	idx.Add(mkHash(0x10, 2), &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
	idx.Add(mkHash(0x10, 3), &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
	idx.Add(mkHash(0xFF, 4), &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
	f := parseCG(t, encode(t, idx))
	fan := f.chunkData["OIDF"]
	if len(fan) != 256*4 {
		t.Fatalf("OIDF size = %d, want 1024", len(fan))
	}
	at := func(i int) uint32 { return binary.BigEndian.Uint32(fan[i*4:]) }
	if at(0) != 1 || at(0x0F) != 1 || at(0x10) != 3 || at(0xFE) != 3 || at(0xFF) != 4 {
		t.Fatalf("fanout not cumulative: f[0]=%d f[0x0f]=%d f[0x10]=%d f[0xfe]=%d f[0xff]=%d",
			at(0), at(0x0F), at(0x10), at(0xFE), at(0xFF))
	}
	// OIDL holds the hashes sorted bytewise.
	oidl := f.chunkData["OIDL"]
	if len(oidl) != 4*20 {
		t.Fatalf("OIDL size = %d, want 80", len(oidl))
	}
	if bytes.Compare(oidl[0:20], mkHash(0x00, 1).Bytes()) != 0 ||
		bytes.Compare(oidl[20:40], mkHash(0x10, 2).Bytes()) != 0 ||
		bytes.Compare(oidl[40:60], mkHash(0x10, 3).Bytes()) != 0 {
		t.Fatal("OIDL not sorted bytewise")
	}
}

// TestDetail05: commit-data row = tree hash + two u32 parent slots + u64 of
// generation<<34 | unixTime.
func TestDetail05(t *testing.T) {
	tree := mkHash(0xAB, 0xCD)
	when := time.Unix(1700000000, 0)
	idx, hs := mkIndex(t,
		&CommitData{TreeHash: tree, Generation: 7, When: when},
	)
	f := parseCG(t, encode(t, idx))
	cdat := f.chunkData["CDAT"]
	if len(cdat) != 36 {
		t.Fatalf("CDAT size = %d, want 36", len(cdat))
	}
	if !bytes.Equal(cdat[:20], tree.Bytes()) {
		t.Fatal("CDAT row does not start with the tree hash")
	}
	packed := binary.BigEndian.Uint64(cdat[28:36])
	if packed>>34 != 7 {
		t.Fatalf("generation bits = %d, want 7 (top 30 bits)", packed>>34)
	}
	if packed&((1<<34)-1) != uint64(when.Unix()) {
		t.Fatalf("time bits = %d, want %d", packed&((1<<34)-1), when.Unix())
	}
	_ = hs
}

// TestDetail06: parent slots hold the parent's position; missing parent is the
// none marker; a parent not in the index is an error.
func TestDetail06(t *testing.T) {
	parent := mkHash(0x10, 0x01)
	child := mkHash(0x80, 0x02)
	idx := NewMemoryIndex()
	idx.Add(parent, &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
	idx.Add(child, &CommitData{TreeHash: mkHash(8, 8), Generation: 2, ParentHashes: []plumbing.Hash{parent}})
	f := parseCG(t, encode(t, idx))
	cdat := f.chunkData["CDAT"]
	// sorted order: 0x10.. parent at index 0, 0x80.. child at index 1.
	p1 := binary.BigEndian.Uint32(cdat[36+20:])
	p2 := binary.BigEndian.Uint32(cdat[36+24:])
	if p1 != 0 {
		t.Fatalf("child parent slot 1 = %d, want 0 (parent's sorted position)", p1)
	}
	if p2 != 0x70000000 {
		t.Fatalf("missing-parent slot = %#x, want the none marker", p2)
	}
	p1 = binary.BigEndian.Uint32(cdat[20:])
	p2 = binary.BigEndian.Uint32(cdat[24:])
	if p1 != 0x70000000 || p2 != 0x70000000 {
		t.Fatalf("root commit parent slots = %#x %#x, want none marker", p1, p2)
	}

	// Parent not in the index must error, never silently encode.
	bad := NewMemoryIndex()
	bad.Add(child, &CommitData{TreeHash: mkHash(8, 8), Generation: 2,
		ParentHashes: []plumbing.Hash{mkHash(0xEE, 0xEE)}})
	var buf bytes.Buffer
	if err := NewEncoder(&buf).Encode(bad); err == nil {
		t.Fatal("Encode with missing parent succeeded — silent zero parent")
	}
}

// TestDetail07: >2 parents — slot 2 becomes extraListIndex|MSB and the extra
// list carries the remaining parents, last OR'd with the last-marker bit.
func TestDetail07(t *testing.T) {
	ps := []plumbing.Hash{mkHash(0x10, 1), mkHash(0x20, 2), mkHash(0x30, 3), mkHash(0x40, 4)}
	idx := NewMemoryIndex()
	for i, p := range ps {
		idx.Add(p, &CommitData{TreeHash: mkHash(9, byte(i)), Generation: 1})
	}
	child := mkHash(0x90, 9)
	idx.Add(child, &CommitData{TreeHash: mkHash(8, 8), Generation: 2, ParentHashes: ps})
	f := parseCG(t, encode(t, idx))
	cdat := f.chunkData["CDAT"]
	// child hash 0x90 sorts last: index 4.
	row := cdat[4*36:]
	p2 := binary.BigEndian.Uint32(row[24:28])
	if p2&0x80000000 == 0 {
		t.Fatalf("slot2 for octopus parent = %#x, want MSB set (extra list)", p2)
	}
	extraIdx := p2 & 0x7fffffff
	edge, ok := f.chunkData["EDGE"]
	if !ok {
		t.Fatal("EDGE chunk missing for >2 parents")
	}
	if int(extraIdx)*4+12 > len(edge) {
		t.Fatalf("extra list index %d out of EDGE bounds (%d bytes)", extraIdx, len(edge))
	}
	got := []uint32{
		binary.BigEndian.Uint32(edge[extraIdx*4:]),
		binary.BigEndian.Uint32(edge[extraIdx*4+4:]),
		binary.BigEndian.Uint32(edge[extraIdx*4+8:]),
	}
	// parents 1..3 (sorted positions 1,2,3); the last carries the marker.
	if got[0] != 1 || got[1] != 2 || got[2] != 3|0x80000000 {
		t.Fatalf("extra edges = %#v, want [1 2 3|MSB]", got)
	}
}

// TestDetail08: generation-v2 values that cannot fit a slot write
// rowIndex|MSB into GDA2 and queue the full u64 into GDO2 in file order.
func TestDetail08(t *testing.T) {
	when := time.Unix(1700000000, 0)
	big := uint64(1<<31) + 42 // corrected date that overflows the slot
	idx := NewMemoryIndex()
	idx.Add(mkHash(0x10, 1), &CommitData{TreeHash: mkHash(9, 1), Generation: 1, When: when,
		GenerationV2: uint64(when.Unix()) + big})
	idx.Add(mkHash(0x20, 2), &CommitData{TreeHash: mkHash(9, 2), Generation: 1, When: when,
		GenerationV2: uint64(when.Unix()) + 5})
	f := parseCG(t, encode(t, idx))
	gda, ok := f.chunkData["GDA2"]
	if !ok {
		t.Fatal("GDA2 chunk missing for generation-v2 index")
	}
	// GDA2 slots are u32; the overflow flag is the u32 MSB.
	slot0 := binary.BigEndian.Uint32(gda[0:4])
	if slot0&0x80000000 == 0 {
		t.Fatalf("oversized generation-v2 slot = %#x, want MSB flag", slot0)
	}
	// Overflow u64s follow the last TOC offset, ahead of the checksum.
	gdo := f.raw[f.tocOff[f.nChunks] : len(f.raw)-20]
	pos := int64(slot0 & 0x7fffffff)
	if int64(len(gdo)) < (pos+1)*8 {
		t.Fatalf("overflow area %d bytes, can't hold pos %d", len(gdo), pos)
	}
	ov := binary.BigEndian.Uint64(gdo[pos*8:])
	if ov != big {
		t.Fatalf("overflow value = %d, want %d", ov, big)
	}
	if binary.BigEndian.Uint32(gda[4:8]) != 5 {
		t.Fatalf("fitting generation-v2 slot = %#x, want 5 unflagged", binary.BigEndian.Uint32(gda[4:8]))
	}
}

// TestDetail09: trailing checksum covers every byte from the magic on.
func TestDetail09(t *testing.T) {
	idx, _ := mkIndex(t, &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
	raw := encode(t, idx)
	sum := sha1.Sum(raw[:len(raw)-20])
	if !bytes.Equal(raw[len(raw)-20:], sum[:]) {
		t.Fatal("trailing checksum does not match sha1 of all preceding bytes")
	}
}

// TestDetail10: chunk presence is decided before writing — the same index
// encodes deterministically and optional chunks exist only when needed.
func TestDetail10(t *testing.T) {
	build := func() []byte {
		idx, _ := mkIndex(t, &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
		return encode(t, idx)
	}
	if !bytes.Equal(build(), build()) {
		t.Fatal("same index encoded to different bytes — chunk layout not precomputed")
	}
	idx, _ := mkIndex(t, &CommitData{TreeHash: mkHash(9, 9), Generation: 1})
	f := parseCG(t, encode(t, idx))
	for _, sig := range []string{"EDGE", "GDA2", "GDO2"} {
		if _, ok := f.chunkData[sig]; ok {
			t.Fatalf("unneeded optional chunk %s present", sig)
		}
	}
}

// TestDetail11: all oversized generation-v2 values land in GDO2 in file order.
// (The input-slice aliasing described in DETAILS is an internal mechanic with
// no observable signature; asserted shape is the overflow queue's content.)
func TestDetail11(t *testing.T) {
	when := time.Unix(1700000000, 0)
	big1 := uint64(1<<31) + 1
	big2 := uint64(1<<31) + 2
	idx := NewMemoryIndex()
	idx.Add(mkHash(0x10, 1), &CommitData{TreeHash: mkHash(9, 1), Generation: 1, When: when,
		GenerationV2: uint64(when.Unix()) + big1})
	idx.Add(mkHash(0x20, 2), &CommitData{TreeHash: mkHash(9, 2), Generation: 1, When: when,
		GenerationV2: uint64(when.Unix()) + big2})
	f := parseCG(t, encode(t, idx))
	// The overflow queue sits between the last TOC offset and the checksum.
	gdo := f.raw[f.tocOff[f.nChunks] : len(f.raw)-20]
	if len(gdo) != 16 {
		t.Fatalf("overflow area = %d bytes, want 16 (two u64)", len(gdo))
	}
	if binary.BigEndian.Uint64(gdo[:8]) != big1 || binary.BigEndian.Uint64(gdo[8:]) != big2 {
		t.Fatal("overflow values not queued in file order")
	}
}

// TestDetail12: an empty index still writes all three mandatory chunks with
// zero-length data sections — not an error.
func TestDetail12(t *testing.T) {
	f := parseCG(t, encode(t, NewMemoryIndex()))
	for _, sig := range []string{"OIDF", "OIDL", "CDAT"} {
		if _, ok := f.chunkData[sig]; !ok {
			t.Fatalf("mandatory chunk %s missing on empty index", sig)
		}
	}
	if len(f.chunkData["OIDF"]) != 1024 {
		t.Fatalf("empty OIDF size = %d, want 1024", len(f.chunkData["OIDF"]))
	}
	if len(f.chunkData["OIDL"]) != 0 || len(f.chunkData["CDAT"]) != 0 {
		t.Fatal("empty index wrote non-empty OIDL/CDAT")
	}
}
