package idxfile

import (
	"bytes"
	"crypto"
	_ "encoding/binary"
	_ "errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"time"

	"example.internal/gitkit/v6/internal/sharedfile"
	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/utils/sync"
	"example.internal/gitkit/v6/x/fdpool"
)

const defaultCloseGracePeriod = time.Second

const (
	idxHeaderSize = 8 // 4 magic + 4 version
	idxFanoutSize = 256 * 4
	off32Size     = 4
	off64Size     = 8
	revHeaderSize = 12 // 4 magic + 4 version + 4 hash function

	is64bitsMask = uint64(1) << 31
)

// ReadAtCloser is the interface required for files used by LazyIndex.
// It is an alias for [sharedfile.ReadAtCloser]; both names refer
// to the same type at compile time.
type ReadAtCloser = sharedfile.ReadAtCloser

// LazyIndex implements the Index interface by reading directly from
// .idx and .rev files via ReadAt, without loading all data into memory.
//
// File descriptors are managed automatically via reference-counted
// shared handles: opened lazily on first use, shared across concurrent
// readers, and closed when no readers remain. This avoids holding
// descriptors open indefinitely while still sharing a single FD across
// concurrent operations.
type LazyIndex struct {
	hashSize int
	count    int
	count64  int
	mem      *MemoryIndex

	// Section byte offsets within the idx file.
	fanoutStart int
	namesStart  int
	crcStart    int
	off32Start  int
	off64Start  int

	idx *sharedfile.SharedFile
	rev *sharedfile.SharedFile

	fanout [256]uint32 // cached from idx; small enough to keep in memory
}

var _ Index = (*LazyIndex)(nil)

// NewLazyIndex creates a LazyIndex from opener functions for .idx and
// .rev files.
//
// The openers are called to obtain file handles on demand. Each call
// must return a fresh, independently closeable handle. File descriptors
// are shared across concurrent readers and released automatically when
// idle.
func NewLazyIndex(openIdx, openRev func() (ReadAtCloser, error), packHash plumbing.Hash) (*LazyIndex, error) {
	return NewLazyIndexWithPool(openIdx, openRev, packHash, nil)
}

// NewLazyIndexWithPool is like [NewLazyIndex] but registers the
// idx and rev [sharedfile.SharedFile]s with the given
// [*fdpool.Pool]. The pool governs LRU eviction across many
// LazyIndexes so a storage-wide FD budget covers the .idx and
// .rev descriptors. Pass nil to disable pooling (equivalent to
// [NewLazyIndex]).
//
// When pool is non-nil the [defaultCloseGracePeriod] timer is
// inert: each FD stays open and registered with the pool until
// the LRU evicts it (or [LazyIndex.Close] tears it down). When
// pool is nil the grace timer governs FD lifetime as in
// [NewLazyIndex].
//
// Neither this constructor nor the [Index] methods accept a
// [context.Context]. Index lookups are pure ReadAt I/O without
// cancellation hooks, matching the context-free convention of
// the storage, plumbing/format, and plumbing/storer layers;
// callers requiring cancellation enforce it at the call-site
// in the layer above.
func NewLazyIndexWithPool(openIdx, openRev func() (ReadAtCloser, error), packHash plumbing.Hash, pool *fdpool.Pool) (*LazyIndex, error) {
	s := &LazyIndex{}
	r, err := openIdx()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var h hash.Hash
	if packHash.Size() == crypto.SHA256.Size() {
		h = crypto.SHA256.New()
	} else {
		h = crypto.SHA1.New()
	}

	mi := NewMemoryIndex(packHash.Size())
	in := statReader{Reader: bytes.NewReader(data), size: int64(len(data))}
	if err := NewDecoder(in, h).Decode(mi); err != nil {
		return nil, err
	}
	s.mem = mi
	s.count = int(mi.Fanout[255])
	return s, nil
}

type statReader struct {
	io.Reader
	size int64
}

func (r statReader) Stat() (fs.FileInfo, error) { return memFileInfo(r.size), nil }

type memFileInfo int64

func (f memFileInfo) Name() string       { return "" }
func (f memFileInfo) Size() int64        { return int64(f) }
func (f memFileInfo) Mode() fs.FileMode  { return 0 }
func (f memFileInfo) ModTime() time.Time { return time.Time{} }
func (f memFileInfo) IsDir() bool        { return false }
func (f memFileInfo) Sys() any           { return nil }

// init reads and validates headers, caches the fanout table and
// computes section offsets. It acquires file handles through the
// sharedFile so the grace period keeps them warm for the first real
// operation.
func (s *LazyIndex) init(packHash plumbing.Hash) error {
	panic("excised: LazyIndex.init")
}

// Contains reports whether the given hash exists in the index by
// binary-searching the idx names table.
func (s *LazyIndex) Contains(h plumbing.Hash) (bool, error) {
	return s.mem.Contains(h)
}

// MayContain implements the Index interface. It reports whether the
// index might contain h, using the cached fanout table loaded at
// construction time. No I/O, no lock. False is authoritative ("h is
// not in this pack"); true means call Contains or FindOffset for a
// definitive answer.
func (s *LazyIndex) MayContain(h plumbing.Hash) bool {
	return s.mem.MayContain(h)
}

// FindOffset returns the packfile offset for the object with the given hash.
// It returns plumbing.ErrObjectNotFound if the hash is not in the index.
func (s *LazyIndex) FindOffset(h plumbing.Hash) (int64, error) {
	return s.mem.FindOffset(h)
}

// FindCRC32 returns the CRC32 checksum of the object with the given hash.
// It returns plumbing.ErrObjectNotFound if the hash is not in the index.
func (s *LazyIndex) FindCRC32(h plumbing.Hash) (uint32, error) {
	return s.mem.FindCRC32(h)
}

// FindHash returns the object hash stored at the given packfile offset
// by binary-searching the .rev reverse index.
// It returns plumbing.ErrObjectNotFound if no object exists at that offset.
func (s *LazyIndex) FindHash(o int64) (plumbing.Hash, error) {
	return s.mem.FindHash(o)
}

// Count returns the total number of objects in the index.
func (s *LazyIndex) Count() (int64, error) {
	return s.mem.Count()
}

// Entries returns an iterator over all index entries in hash order.
// The caller must call Close on the returned iterator to release the
// underlying file reference.
func (s *LazyIndex) Entries() (EntryIter, error) {
	return s.mem.Entries()
}

// EntriesWithPrefix implements the Index interface. It returns an
// iterator over entries whose hashes start with prefix. When prefix
// is empty the call is equivalent to Entries; otherwise the
// iterator visits only the fanout-bounded names-table slice
// selected by prefix[0] and stops as soon as a name without prefix
// is read (the names table is sorted by hash).
//
// For a multi-byte prefix the matching entries form a contiguous
// run somewhere within the bucket; binary-search positions the
// iterator at the start of that run so the linear walk only spans
// matches. This mirrors upstream Git's for_each_prefixed_object_in_pack
// which calls bsearch_pack to position before walking forward.
//
// The returned iterator holds an acquired reference to the idx
// SharedFile which is released on Close.
func (s *LazyIndex) EntriesWithPrefix(prefix []byte) (EntryIter, error) {
	panic("excised: LazyIndex.EntriesWithPrefix")
}

// EntriesByOffset returns an iterator over all index entries sorted by
// their packfile offset. It reads positions from the .rev file on each
// call to Next, avoiding any up-front allocation or sorting.
//
// The caller must call Close on the returned iterator to release the
// underlying file references.
func (s *LazyIndex) EntriesByOffset() (EntryIter, error) {
	return s.mem.EntriesByOffset()
}

// Close releases the underlying shared file handles, preventing future
// operations. If there are active readers they will finish normally;
// the file descriptors close when the last reader is done.
func (s *LazyIndex) Close() error {
	panic("excised: LazyIndex.Close")
}

// CloseIdleDescriptors releases the idx and rev file descriptors
// without disabling the [LazyIndex]. The FDs close inline when no
// readers are active; otherwise each [sharedfile.SharedFile]
// latches an immediate close on the next refs==0 transition.
// In-flight readers complete normally; subsequent operations
// reopen the FDs on demand and resume normal grace-timer
// behaviour.
//
// Returns the joined error of the inline closes; latched closes
// that fire later are not reported.
func (s *LazyIndex) CloseIdleDescriptors() error {
	panic("excised: LazyIndex.CloseIdleDescriptors")
}

// --- internal helpers; all take an io.ReaderAt so the caller controls
//     the acquire/release lifecycle. ---

// findHashPos binary-searches the names table for h, returning the flat
// position (0..count-1) if found.
func (s *LazyIndex) findHashPos(idx io.ReaderAt, h plumbing.Hash) (int, bool, error) {
	if h.Size() != s.hashSize {
		return 0, false, fmt.Errorf("hash size mismatch: %d %d", h.Size(), s.hashSize)
	}
	first := int(h.Bytes()[0])
	var lo int
	if first > 0 {
		lo = int(s.fanout[first-1])
	}
	hi := int(s.fanout[first])
	if lo >= hi {
		return 0, false, nil
	}

	target := h.Bytes()[:s.hashSize]
	var arr [32]byte
	buf := arr[:s.hashSize]

	for lo < hi {
		mid := (lo + hi) >> 1
		nameOff := int64(s.namesStart + mid*s.hashSize)
		if _, err := idx.ReadAt(buf, nameOff); err != nil {
			return 0, false, fmt.Errorf("read name at pos %d: %w", mid, err)
		}

		cmp := bytes.Compare(target, buf)
		switch {
		case cmp < 0:
			hi = mid
		case cmp > 0:
			lo = mid + 1
		default:
			return mid, true, nil
		}
	}
	return 0, false, nil
}

// offset returns the pack offset for the object at position pos.
func (s *LazyIndex) offset(idx io.ReaderAt, pos int) (uint64, error) {
	panic("excised: LazyIndex.offset")
}

// count64bitOffsets scans the 32-bit offset table and returns the number
// of entries whose MSB is set (i.e. that use the 64-bit overflow table).
func (s *LazyIndex) count64bitOffsets(idx io.ReaderAt) (int, error) {
	panic("excised: LazyIndex.count64bitOffsets")
}

// crc32 returns the CRC32 for the object at position pos.
func (s *LazyIndex) crc32(idx io.ReaderAt, pos int) (uint32, error) {
	panic("excised: LazyIndex.crc32")
}

// hashAtPos reads the hash at the given flat position.
func (s *LazyIndex) hashAtPos(idx io.ReaderAt, pos int) (plumbing.Hash, error) {
	panic("excised: LazyIndex.hashAtPos")
}

func (s *LazyIndex) findHashViaRev(idx, rev io.ReaderAt, want int64) (plumbing.Hash, error) {
	panic("excised: LazyIndex.findHashViaRev")
}

// entryAt reads a complete entry at the given flat position.
func (s *LazyIndex) entryAt(idx io.ReaderAt, pos int) (*Entry, error) {
	panic("excised: LazyIndex.entryAt")
}

// scannerEntryIter iterates over entries in hash order.
// It holds an acquired reference to the idx sharedFile which is
// released when Close is called.
type scannerEntryIter struct {
	s   *LazyIndex
	idx io.ReaderAt // acquired from s.idx
	pos int
}

func (it *scannerEntryIter) Next() (*Entry, error) {
	panic("excised: scannerEntryIter.Next")
}

func (it *scannerEntryIter) Close() error {
	panic("excised: scannerEntryIter.Close")
}

// revEntryIter iterates over entries in packfile-offset order by
// walking the .rev file sequentially. It holds acquired references to
// both the idx and rev sharedFiles, released on Close.
type revEntryIter struct {
	s   *LazyIndex
	idx io.ReaderAt
	rev io.ReaderAt
	pos int
}

func (it *revEntryIter) Next() (*Entry, error) {
	panic("excised: revEntryIter.Next")
}

func (it *revEntryIter) Close() error {
	panic("excised: revEntryIter.Close")
}

// lazyPrefixIter walks the LazyIndex names table from pos to end,
// yielding entries whose hash starts with prefix. It stops the run
// when a hash without the prefix is read (the table is sorted). It
// holds an acquired reference to the idx SharedFile released on
// Close.
//
// Lifetime: Next may release the iterator's SharedFile reference
// eagerly when the first prefix-mismatched entry is observed —
// further matches are impossible in the sorted table, so holding
// the reference would only add pool pressure. Callers should
// defer Close unconditionally; it is idempotent and the eager
// release is purely an optimisation.
type lazyPrefixIter struct {
	s      *LazyIndex
	idx    io.ReaderAt
	prefix []byte
	pos    int
	end    int
}

func (it *lazyPrefixIter) Next() (*Entry, error) {
	panic("excised: lazyPrefixIter.Next")
}

func (it *lazyPrefixIter) Close() error {
	panic("excised: lazyPrefixIter.Close")
}
