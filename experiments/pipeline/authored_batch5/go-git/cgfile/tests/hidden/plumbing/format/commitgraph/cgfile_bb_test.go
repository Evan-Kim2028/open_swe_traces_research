package commitgraph

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/memfs"

	"example.internal/gitkit/v6/plumbing"
)

// --- helpers ---------------------------------------------------------------

// rac is a sizeable ReaderAtCloser (bytes.Reader exposes Size()).
type rac struct{ *bytes.Reader }

func (rac) Close() error { return nil }

// racNoSize hides Size/Seek so readerSize cannot report a length.
type racNoSize struct {
	r  io.ReaderAt
	ec error
}

func (r racNoSize) ReadAt(b []byte, off int64) (int, error) { return r.r.ReadAt(b, off) }
func (r racNoSize) Close() error                            { return r.ec }

var errReaderClose = errors.New("reader close failed")

func cgHash(first byte) plumbing.Hash {
	h, _ := plumbing.FromHex(fmt.Sprintf("%02x", first) + strings.Repeat("ab", 19))
	return h
}

func treeHash() plumbing.Hash {
	h, _ := plumbing.FromHex("cd" + strings.Repeat("01", 19))
	return h
}

func mkCommit(h plumbing.Hash, parents []plumbing.Hash, gen, genV2 uint64) *CommitData {
	return &CommitData{
		TreeHash:     treeHash(),
		ParentHashes: parents,
		Generation:   gen,
		GenerationV2: genV2,
		When:         time.Unix(1700000000, 0),
	}
}

func encodeCG(t *testing.T, idx Index) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := NewEncoder(&buf).Encode(idx); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.Bytes()
}

// threeCommitFile returns a valid encoded commit-graph with three commits
// (sorted hashes, linear parent chain) plus the sorted hash list.
func threeCommitFile(t *testing.T) ([]byte, []plumbing.Hash, *MemoryIndex) {
	t.Helper()
	hs := []plumbing.Hash{cgHash(0x10), cgHash(0x40), cgHash(0xa0)}
	mi := NewMemoryIndex()
	mi.Add(hs[0], mkCommit(hs[0], nil, 1, 0))
	mi.Add(hs[1], mkCommit(hs[1], []plumbing.Hash{hs[0]}, 2, 0))
	mi.Add(hs[2], mkCommit(hs[2], []plumbing.Hash{hs[1]}, 3, 0))
	return encodeCG(t, mi), hs, mi
}

// tocOffset returns the file offset stored in the i-th table-of-contents
// entry (entries start at byte 8 and are 12 bytes wide: 4 sig + 8 offset).
func tocOffset(b []byte, i int) int64 {
	return int64(binary.BigEndian.Uint64(b[8+i*12+4:]))
}

func setTocOffset(b []byte, i int, off int64) {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], uint64(off))
	copy(b[8+i*12+4:], tmp[:])
}

func openCG(t *testing.T, b []byte) (Index, error) {
	t.Helper()
	return OpenFileIndex(rac{bytes.NewReader(b)})
}

// --- tests -----------------------------------------------------------------

// TestDetail01: header check — signature, version==1, hash byte — each maps
// to its own error var.
func TestDetail01(t *testing.T) {
	good, _, _ := threeCommitFile(t)

	bad := bytes.Clone(good)
	bad[0] = 'X'
	if _, err := openCG(t, bad); err == nil {
		t.Fatal("bad signature accepted")
	}

	bad = bytes.Clone(good)
	bad[4] = 2
	if _, err := openCG(t, bad); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("version=2: %v, want ErrUnsupportedVersion", err)
	}

	bad = bytes.Clone(good)
	bad[5] = 9
	if _, err := openCG(t, bad); !errors.Is(err, ErrUnsupportedHash) {
		t.Fatalf("hash=9: %v, want ErrUnsupportedHash", err)
	}
}

// TestDetail02: whole-file size precheck — truncated file fails on open when
// the reader reports a size; a reader that cannot report a size skips the
// precheck and still opens a well-formed file.
func TestDetail02(t *testing.T) {
	good, _, _ := threeCommitFile(t)

	// drop the trailer and part of the last chunk so the file is smaller
	// than header + full TOC + fanout + trailer.
	trunc := good[:len(good)-200]
	if _, err := openCG(t, trunc); err == nil {
		t.Fatal("truncated file opened")
	}

	idx, err := OpenFileIndex(racNoSize{r: bytes.NewReader(good)})
	if err != nil {
		t.Fatalf("non-sizer reader on valid file: %v", err)
	}
	defer idx.Close()
	if idx.MaximumNumberOfHashes() != 3 {
		t.Fatalf("hashes=%d, want 3", idx.MaximumNumberOfHashes())
	}
}

// TestDetail03: TOC walk bounded by the declared chunk count; offsets must be
// monotonically non-decreasing and below fileSize minus trailer.
func TestDetail03(t *testing.T) {
	good, _, _ := threeCommitFile(t)

	// non-monotonic: entry 2 offset earlier than entry 1
	bad := bytes.Clone(good)
	setTocOffset(bad, 2, tocOffset(good, 1)-1)
	if _, err := openCG(t, bad); err == nil {
		t.Fatal("non-monotonic TOC accepted")
	}

	// beyond fileSize minus trailer
	bad = bytes.Clone(good)
	setTocOffset(bad, 2, int64(len(good))-8)
	if _, err := openCG(t, bad); err == nil {
		t.Fatal("TOC offset into trailer accepted")
	}
}

// TestDetail04: a duplicate chunk id is malformed, and so is a zero id before
// the declared count ends.
func TestDetail04(t *testing.T) {
	good, _, _ := threeCommitFile(t)

	bad := bytes.Clone(good)
	copy(bad[8+1*12:], OIDFanoutChunk.Signature()) // duplicate OIDF
	if _, err := openCG(t, bad); err == nil {
		t.Fatal("duplicate chunk id accepted")
	}

	bad = bytes.Clone(good)
	for i := 0; i < 4; i++ {
		bad[8+0*12+i] = 0 // zero id in first entry
	}
	if _, err := openCG(t, bad); err == nil {
		t.Fatal("early zero chunk id accepted")
	}
}

// TestDetail05: a single zero-id terminator must follow the table.
func TestDetail05(t *testing.T) {
	good, _, _ := threeCommitFile(t)
	n := int(good[6]) // numChunks

	bad := bytes.Clone(good)
	copy(bad[8+n*12:], ExtraEdgeListChunk.Signature()) // non-zero terminator
	if _, err := openCG(t, bad); err == nil {
		t.Fatal("non-zero terminator accepted")
	}
}

// TestDetail06: chunk cardinalities are checked against the fanout commit
// count at open time — a shrunken lookup chunk fails on open.
func TestDetail06(t *testing.T) {
	good, _, _ := threeCommitFile(t)

	bad := bytes.Clone(good)
	// shrink OIDL region to a single hash by moving the next chunk's offset
	setTocOffset(bad, 2, tocOffset(good, 1)+20)
	if _, err := openCG(t, bad); err == nil {
		t.Fatal("cardinality-truncated file opened")
	}
}

// TestDetail07: fanout bucketing by first hash byte, then binary search; a
// miss falls through to the parent index before reporting not-found.
func TestDetail07(t *testing.T) {
	child, chs, _ := threeCommitFile(t)

	parent := NewMemoryIndex()
	ph1, ph2 := cgHash(0x05), cgHash(0x90)
	parent.Add(ph1, mkCommit(ph1, nil, 1, 0))
	parent.Add(ph2, mkCommit(ph2, []plumbing.Hash{ph1}, 2, 0))

	idx, err := OpenFileIndexWithParent(rac{bytes.NewReader(child)}, parent)
	if err != nil {
		t.Fatalf("open with parent: %v", err)
	}
	defer idx.Close()

	for i, h := range chs {
		got, err := idx.GetIndexByHash(h)
		if err != nil || int(got) != 2+i {
			t.Fatalf("GetIndexByHash(child %d)=%d,%v want %d", i, got, err, 2+i)
		}
	}
	for i, h := range []plumbing.Hash{ph1, ph2} {
		got, err := idx.GetIndexByHash(h)
		if err != nil || int(got) != i {
			t.Fatalf("GetIndexByHash(parent %d)=%d,%v want %d", i, got, err, i)
		}
	}
	if _, err := idx.GetIndexByHash(cgHash(0x77)); err == nil {
		t.Fatal("unknown hash found")
	}
}

// TestDetail08: indexes below the parent's count delegate to the parent; the
// local fanout total is not consulted for them.
func TestDetail08(t *testing.T) {
	child, chs, _ := threeCommitFile(t)

	parent := NewMemoryIndex()
	ph := cgHash(0x05)
	pcd := mkCommit(ph, nil, 1, 0)
	pcd.TreeHash, _ = plumbing.FromHex("ee" + strings.Repeat("02", 19))
	parent.Add(ph, pcd)

	idx, err := OpenFileIndexWithParent(rac{bytes.NewReader(child)}, parent)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer idx.Close()

	h, err := idx.GetHashByIndex(0)
	if err != nil || h != ph {
		t.Fatalf("GetHashByIndex(0)=%v,%v want parent %v", h, err, ph)
	}
	h, err = idx.GetHashByIndex(1)
	if err != nil || h != chs[0] {
		t.Fatalf("GetHashByIndex(1)=%v,%v want child %v", h, err, chs[0])
	}
	cd, err := idx.GetCommitDataByIndex(0)
	if err != nil {
		t.Fatalf("GetCommitDataByIndex(0): %v", err)
	}
	if cd.TreeHash != pcd.TreeHash {
		t.Fatal("index 0 did not delegate to parent")
	}
}

// TestDetail09: an octopus merge walks the edge-list chunk until a terminator
// bit; an unterminated walk is malformed.
func TestDetail09(t *testing.T) {
	h0, h1, h2, h3 := cgHash(0x10), cgHash(0x20), cgHash(0x30), cgHash(0x40)
	mi := NewMemoryIndex()
	mi.Add(h0, mkCommit(h0, nil, 1, 0))
	mi.Add(h1, mkCommit(h1, []plumbing.Hash{h0}, 2, 0))
	mi.Add(h2, mkCommit(h2, []plumbing.Hash{h0, h1}, 3, 0))
	mi.Add(h3, mkCommit(h3, []plumbing.Hash{h0, h1, h2}, 4, 0))
	good := encodeCG(t, mi)

	idx, err := openCG(t, good)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ci, err := idx.GetIndexByHash(h3)
	if err != nil {
		t.Fatalf("lookup octopus: %v", err)
	}
	cd, err := idx.GetCommitDataByIndex(ci)
	if err != nil {
		t.Fatalf("commit data: %v", err)
	}
	if len(cd.ParentIndexes) != 3 {
		t.Fatalf("octopus parents=%v, want 3", cd.ParentIndexes)
	}
	idx.Close()

	// corrupt: clear terminator bits in the EDGE chunk
	bad := bytes.Clone(good)
	edgeOff := int64(-1)
	for i := 0; i < int(good[6]); i++ {
		if bytes.Equal(good[8+i*12:8+i*12+4], ExtraEdgeListChunk.Signature()) {
			edgeOff = tocOffset(good, i)
		}
	}
	if edgeOff < 0 {
		t.Fatal("no EDGE chunk emitted")
	}
	// clear high bits of every uint32 until the next chunk / trailer
	end := int64(len(bad)) - 20
	for i := 0; i < int(bad[6]); i++ {
		o := tocOffset(good, i)
		if o > edgeOff && o < end {
			end = o
		}
	}
	for off := edgeOff; off+4 <= end; off += 4 {
		bad[off] &= 0x7f
	}
	idx2, err := openCG(t, bad)
	if err != nil {
		t.Fatalf("open corrupted: %v", err)
	}
	defer idx2.Close()
	ci2, _ := idx2.GetIndexByHash(h3)
	if _, err := idx2.GetCommitDataByIndex(ci2); err == nil {
		t.Fatal("unterminated edge walk accepted")
	}
}

// TestDetail10: GenerationV2 = commit time + generation datum; the datum's
// high bit indexes the overflow chunk for a corrected 64-bit timestamp.
func TestDetail10(t *testing.T) {
	h0, h1 := cgHash(0x10), cgHash(0x20)
	mi := NewMemoryIndex()
	small := uint64(1700000000 + 42)
	big := uint64(1)<<33 + 7 // forces the overflow path
	mi.Add(h0, mkCommit(h0, nil, 1, small))
	mi.Add(h1, mkCommit(h1, []plumbing.Hash{h0}, 2, big))
	data := encodeCG(t, mi)

	idx, err := openCG(t, data)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer idx.Close()
	if !idx.HasGenerationV2() {
		t.Fatal("HasGenerationV2 false on genV2 file")
	}
	for _, want := range []struct {
		h     plumbing.Hash
		genV2 uint64
	}{{h0, small}, {h1, big}} {
		ci, err := idx.GetIndexByHash(want.h)
		if err != nil {
			t.Fatalf("lookup: %v", err)
		}
		cd, err := idx.GetCommitDataByIndex(ci)
		if err != nil {
			t.Fatalf("data: %v", err)
		}
		if cd.GenerationV2 != want.genV2 {
			t.Fatalf("GenerationV2=%d, want %d", cd.GenerationV2, want.genV2)
		}
	}
}

// TestDetail11: the chain file is newline-separated full object ids,
// oldest-to-newest; a malformed line fails the read and a final line with no
// trailing newline is dropped.
func TestDetail11(t *testing.T) {
	h1 := "01" + strings.Repeat("a", 38)
	h2 := "02" + strings.Repeat("b", 38)

	got, err := OpenChainFile(strings.NewReader(h1 + "\n" + h2 + "\n"))
	if err != nil || len(got) != 2 || got[0] != h1 || got[1] != h2 {
		t.Fatalf("two-line chain: %v %v", got, err)
	}

	got, err = OpenChainFile(strings.NewReader(h1 + "\n" + h2))
	if err != nil {
		t.Fatalf("unterminated tail errored: %v", err)
	}
	if len(got) != 1 || got[0] != h1 {
		t.Fatalf("trailing line not dropped: %v", got)
	}

	if _, err := OpenChainFile(strings.NewReader("not-a-hash\n")); err == nil {
		t.Fatal("malformed line accepted")
	}
}

// TestDetail12: chain-or-file prefers the single graph file and falls back to
// the chain only when the file cannot be opened — a file that opens but fails
// to parse is a hard error.
func TestDetail12(t *testing.T) {
	good, _, _ := threeCommitFile(t)
	chain := cgHash(0x33).String() + "\n"

	write := func(fs billy.Filesystem, path string, data []byte) {
		if err := writeFile(fs, path, data); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	// file only
	fs1 := memfs.New()
	write(fs1, "objects/info/commit-graph", good)
	idx, err := OpenChainOrFileIndex(fs1)
	if err != nil {
		t.Fatalf("file-only open: %v", err)
	}
	if idx.MaximumNumberOfHashes() != 3 {
		t.Fatalf("file-only hashes=%d", idx.MaximumNumberOfHashes())
	}
	idx.Close()

	// corrupt file (opens fine, fails to parse) + valid chain -> hard error,
	// no fallback
	fs2 := memfs.New()
	badVersion := bytes.Clone(good)
	badVersion[4] = 7
	write(fs2, "objects/info/commit-graph", badVersion)
	write(fs2, "objects/info/commit-graphs/commit-graph-chain", []byte(chain))
	write(fs2, "objects/info/commit-graphs/graph-"+cgHash(0x33).String()+".graph", good)
	if _, err := OpenChainOrFileIndex(fs2); err == nil {
		t.Fatal("corrupt file silently fell back to chain")
	}
}

func writeFile(fs billy.Filesystem, path string, data []byte) error {
	f, err := fs.Create(path)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	return f.Close()
}

// TestDetail13: closing a chained index also closes the parent; the parent's
// close error is reported only when the reader closed cleanly.
func TestDetail13(t *testing.T) {
	good, _, _ := threeCommitFile(t)

	parentErr := errors.New("parent close failed")
	cleanIdx, err := OpenFileIndexWithParent(
		rac{bytes.NewReader(good)}, &closeIdx{Index: NewMemoryIndex(), err: parentErr})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := cleanIdx.Close(); !errors.Is(err, parentErr) {
		t.Fatalf("clean close: %v, want parent error", err)
	}

	badIdx, err := OpenFileIndexWithParent(
		racNoSize{r: bytes.NewReader(good), ec: errReaderClose},
		&closeIdx{Index: NewMemoryIndex(), err: parentErr})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := badIdx.Close(); !errors.Is(err, errReaderClose) {
		t.Fatalf("dirty close: %v, want reader error", err)
	}
}

type closeIdx struct {
	Index
	err error
}

func (c *closeIdx) Close() error {
	c.Index.Close()
	return c.err
}

// TestDetail14: HasGenerationV2 is the AND of this file's chunk presence and
// the parent's flag.
func TestDetail14(t *testing.T) {
	h0 := cgHash(0x10)

	withV2 := NewMemoryIndex()
	withV2.Add(h0, mkCommit(h0, nil, 1, 1700000100))
	noV2 := NewMemoryIndex()
	noV2.Add(h0, mkCommit(h0, nil, 1, 0))

	fileWithV2 := encodeCG(t, withV2)
	fileNoV2 := encodeCG(t, noV2)

	parentWith := NewMemoryIndex()
	parentWith.Add(cgHash(0x05), mkCommit(cgHash(0x05), nil, 1, 1700000001))
	parentWithout := NewMemoryIndex()
	parentWithout.Add(cgHash(0x05), mkCommit(cgHash(0x05), nil, 1, 0))

	cases := []struct {
		name   string
		file   []byte
		parent Index
		want   bool
	}{
		{"v2 file + v2 parent", fileWithV2, parentWith, true},
		{"v2 file + no-v2 parent", fileWithV2, parentWithout, false},
		{"no-v2 file + v2 parent", fileNoV2, parentWith, false},
	}
	for _, c := range cases {
		idx, err := OpenFileIndexWithParent(rac{bytes.NewReader(c.file)}, c.parent)
		if err != nil {
			t.Fatalf("%s: open: %v", c.name, err)
		}
		if got := idx.HasGenerationV2(); got != c.want {
			t.Fatalf("%s: HasGenerationV2=%v want %v", c.name, got, c.want)
		}
		idx.Close()
	}
}
