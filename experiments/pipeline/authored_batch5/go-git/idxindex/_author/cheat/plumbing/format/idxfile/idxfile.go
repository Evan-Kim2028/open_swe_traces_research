package idxfile

import (
	_ "bytes"
	_ "crypto"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"sync"

	"example.internal/gitkit/v6/plumbing"
)

const (
	// VersionSupported is the only idx version supported.
	VersionSupported = 2

	noMapping = -1
)

var idxHeader = []byte{255, 't', 'O', 'c'}

// Index represents an index of a packfile.
//
// Implementations satisfy a [io.Closer] contract via [Index.Close]:
// on-disk implementations release file descriptors, pure
// in-memory implementations return nil. Downstream callers
// holding their own concrete [Index] implementations must
// supply a [Close] method to satisfy this interface; a no-op
// `func (*MyIndex) Close() error { return nil }` is sufficient
// for in-memory backends.
type Index interface {
	// Contains checks whether the given hash is in the index.
	Contains(h plumbing.Hash) (bool, error)
	// FindOffset finds the offset in the packfile for the object with
	// the given hash.
	FindOffset(h plumbing.Hash) (int64, error)
	// FindCRC32 finds the CRC32 of the object with the given hash.
	FindCRC32(h plumbing.Hash) (uint32, error)
	// FindHash finds the hash for the object with the given offset.
	FindHash(o int64) (plumbing.Hash, error)
	// Count returns the number of entries in the index.
	Count() (int64, error)
	// Entries returns an iterator to retrieve all index entries.
	Entries() (EntryIter, error)
	// EntriesByOffset returns an iterator to retrieve all index entries ordered
	// by offset.
	EntriesByOffset() (EntryIter, error)
	// EntriesWithPrefix returns an iterator over index entries whose
	// hashes start with prefix. Implementations use the fanout table
	// to bound the search when len(prefix) >= 1; an empty prefix
	// returns all entries (equivalent to Entries). The returned
	// iterator must be Closed by the caller to release any held
	// resources.
	EntriesWithPrefix(prefix []byte) (EntryIter, error)
	// MayContain reports whether the index might contain h. A false
	// return is authoritative ("h is definitely not in this pack")
	// based on the idx fanout table; true means the caller should
	// call Contains or FindOffset for a definitive answer.
	//
	// Implementations must be O(1) and I/O-free. Callers route
	// every read through MayContain to gate further index work
	// (see storage/filesystem.ObjectStorage.findObjectInPackfile);
	// an implementation that performs I/O or scales with index
	// size silently regresses every storage-level read.
	MayContain(h plumbing.Hash) bool
	// Close releases any resources held by the index. Implementations
	// backed by on-disk files must close their file descriptors; pure
	// in-memory implementations must return nil. Close is idempotent.
	Close() error
}

// MemoryIndex is the in memory representation of an idx file.
//
// The use of MemoryIndex for large repositories is discouraged.
// Use [LazyIndex] instead.
type MemoryIndex struct {
	// Version is the version of the index file.
	Version uint32
	// Fanout is a table where the Nth entry is the cumulative count of objects with the first byte of their name <= N.
	Fanout [256]uint32
	// FanoutMapping maps the position in the fanout table to the position
	// in the Names, Offset32 and CRC32 slices. This improves the memory
	// usage by not needing an array with unnecessary empty slots.
	FanoutMapping [256]int
	// Names is the list of object names.
	Names [][]byte
	// Offset32 is the list of 32-bit offsets.
	Offset32 [][]byte
	// CRC32 is the list of CRC32 checksums.
	CRC32 [][]byte
	// Offset64 is the list of 64-bit offsets.
	Offset64 []byte
	// PackfileChecksum is the checksum of the packfile.
	PackfileChecksum plumbing.Hash
	// IdxChecksum is the checksum of the index file.
	IdxChecksum plumbing.Hash

	offsetHash      map[int64]plumbing.Hash
	offsetBuildOnce sync.Once
	mu              sync.RWMutex

	objectIDSize int
}

var _ Index = (*MemoryIndex)(nil)

// Close is a no-op. MemoryIndex holds no external resources.
func (idx *MemoryIndex) Close() error {
	panic("excised: MemoryIndex.Close")
}

// NewMemoryIndex returns an instance of a new MemoryIndex.
func NewMemoryIndex(objectIDSize int) *MemoryIndex {
	return &MemoryIndex{
		offsetHash:   make(map[int64]plumbing.Hash),
		objectIDSize: objectIDSize,
	}
}

func (idx *MemoryIndex) findHashIndex(h plumbing.Hash) (int, bool) {
	// Linear scan over all fanout buckets.
	pos := 0
	for b := 0; b < 256; b++ {
		mapped := idx.FanoutMapping[b]
		if mapped < 0 {
			continue
		}
		names := idx.Names[mapped]
		for off := 0; off+idx.objectIDSize <= len(names); off += idx.objectIDSize {
			if h.Compare(names[off:off+idx.objectIDSize]) == 0 {
				return pos, true
			}
			pos++
		}
	}
	return -1, false
}

// MayContain implements the Index interface. It reports whether the
// index might contain h using the in-memory fanout mapping. Returns
// false iff h's first byte falls in an empty fanout bucket.
func (idx *MemoryIndex) MayContain(h plumbing.Hash) bool {
	_, ok := idx.findHashIndex(h)
	return ok
}

// Contains implements the Index interface.
func (idx *MemoryIndex) Contains(h plumbing.Hash) (bool, error) {
	_, ok := idx.findHashIndex(h)
	return ok, nil
}

// FindOffset implements the Index interface.
func (idx *MemoryIndex) FindOffset(h plumbing.Hash) (int64, error) {
	i, ok := idx.findHashIndex(h)
	if !ok {
		return 0, plumbing.ErrObjectNotFound
	}
	pos := 0
	for b := 0; b < 256; b++ {
		mapped := idx.FanoutMapping[b]
		if mapped < 0 {
			continue
		}
		cnt := int(idx.Fanout[b])
		prev := uint32(0)
		if b > 0 {
			prev = idx.Fanout[b-1]
		}
		cnt -= int(prev)
		if i < pos+cnt {
			off, err := idx.getOffset(mapped, i-pos)
			return int64(off), err
		}
		pos += cnt
	}
	return 0, plumbing.ErrObjectNotFound
}

const isO64Mask = uint64(1) << 31

func (idx *MemoryIndex) getOffset(firstLevel, secondLevel int) (uint64, error) {
	b := idx.Offset32[firstLevel][secondLevel*offset32Len:]
	off := uint64(binary.BigEndian.Uint32(b))
	if off&isO64Mask != 0 {
		p := int(off &^ isO64Mask)
		return binary.BigEndian.Uint64(idx.Offset64[p*offset64Len:]), nil
	}
	return off, nil
}

// FindCRC32 implements the Index interface.
func (idx *MemoryIndex) FindCRC32(h plumbing.Hash) (uint32, error) {
	i, ok := idx.findHashIndex(h)
	if !ok {
		return 0, plumbing.ErrObjectNotFound
	}
	pos := 0
	for b := 0; b < 256; b++ {
		mapped := idx.FanoutMapping[b]
		if mapped < 0 {
			continue
		}
		cnt := int(idx.Fanout[b])
		prev := uint32(0)
		if b > 0 {
			prev = idx.Fanout[b-1]
		}
		cnt -= int(prev)
		if i < pos+cnt {
			return idx.getCRC32(mapped, i-pos), nil
		}
		pos += cnt
	}
	return 0, plumbing.ErrObjectNotFound
}

func (idx *MemoryIndex) getCRC32(firstLevel, secondLevel int) uint32 {
	return binary.BigEndian.Uint32(idx.CRC32[firstLevel][secondLevel*crc32Len:])
}

// FindHash implements the Index interface.
func (idx *MemoryIndex) FindHash(o int64) (plumbing.Hash, error) {
	idx.offsetBuildOnce.Do(func() {
		_ = idx.genOffsetHash()
	})
	if h, ok := idx.offsetHash[o]; ok {
		return h, nil
	}
	return plumbing.ZeroHash, plumbing.ErrObjectNotFound
}

// genOffsetHash generates the offset/hash mapping for reverse search.
func (idx *MemoryIndex) genOffsetHash() error {
	// Flatten names and offsets into the reverse map.
	for b := 0; b < 256; b++ {
		mapped := idx.FanoutMapping[b]
		if mapped < 0 {
			continue
		}
		names := idx.Names[mapped]
		for off := 0; off+idx.objectIDSize <= len(names); off += idx.objectIDSize {
			o, err := idx.getOffset(mapped, off/idx.objectIDSize)
			if err != nil {
				return err
			}
			var h plumbing.Hash
			h.ResetBySize(idx.objectIDSize)
			h.Write(names[off : off+idx.objectIDSize])
			idx.offsetHash[int64(o)] = h
		}
	}
	return nil
}

// Count implements the Index interface.
func (idx *MemoryIndex) Count() (int64, error) {
	return int64(idx.Fanout[255]), nil
}

// Entries implements the Index interface.
func (idx *MemoryIndex) Entries() (EntryIter, error) {
	return &idxfileEntryIter{idx: idx}, nil
}

// EntriesWithPrefix implements the Index interface. It returns an
// iterator over entries whose hashes start with prefix. When prefix
// is empty the call is equivalent to Entries; otherwise the
// iterator visits only the fanout bucket selected by prefix[0] and
// stops as soon as the sorted-by-hash bucket walks past prefix.
//
// For a multi-byte prefix the matching entries form a contiguous
// run somewhere within the bucket; binary-search positions the
// iterator at the start of that run so the linear walk only spans
// matches. This mirrors upstream Git's for_each_prefixed_object_in_pack
// which calls bsearch_pack to position before walking forward.
func (idx *MemoryIndex) EntriesWithPrefix(prefix []byte) (EntryIter, error) {
	panic("excised: MemoryIndex.EntriesWithPrefix")
}

// EntriesByOffset implements the Index interface.
func (idx *MemoryIndex) EntriesByOffset() (EntryIter, error) {
	it, err := idx.Entries()
	if err != nil {
		return nil, err
	}
	var all entriesByOffset
	for {
		e, err := it.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		all = append(all, e)
	}
	sort.Sort(all)
	return &idxfileEntryOffsetIter{entries: all}, nil
}

func (idx *MemoryIndex) idSize() int {
	return idx.objectIDSize
}

// EntryIter is an iterator that will return the entries in a packfile index.
type EntryIter interface {
	// Next returns the next entry in the packfile index.
	Next() (*Entry, error)
	// Close closes the iterator.
	Close() error
}

type idxfileEntryIter struct {
	idx                     *MemoryIndex
	total                   int
	firstLevel, secondLevel int
}

func (i *idxfileEntryIter) Next() (*Entry, error) {
	for {
		if i.firstLevel >= fanout {
			return nil, io.EOF
		}

		if i.total >= int(i.idx.Fanout[i.firstLevel]) {
			i.firstLevel++
			i.secondLevel = 0
			continue
		}

		mappedFirstLevel := i.idx.FanoutMapping[i.firstLevel]
		entry := new(Entry)
		entry.Hash.ResetBySize(i.idx.idSize())
		_, err := entry.Hash.Write(i.idx.Names[mappedFirstLevel][i.secondLevel*i.idx.idSize():])
		if err != nil {
			return nil, fmt.Errorf("cannot write entry hash: %w", err)
		}

		entry.Offset, err = i.idx.getOffset(mappedFirstLevel, i.secondLevel)
		if err != nil {
			return nil, err
		}
		entry.CRC32 = i.idx.getCRC32(mappedFirstLevel, i.secondLevel)

		i.secondLevel++
		i.total++

		return entry, nil
	}
}

func (i *idxfileEntryIter) Close() error {
	i.firstLevel = fanout
	return nil
}

// idxfilePrefixIter walks a single fanout bucket, yielding entries
// whose hash starts with prefix. The bucket is sorted by hash, so
// once a name is read whose first bytes do not match prefix the
// iterator stops.
//
// The iterator references the bucket's per-slot slices directly
// (names, offset32, crc32) plus the shared offset64 table, so it
// does not retain a reference to the parent MemoryIndex. This keeps
// the iterator footprint to just the cursor state and the slice
// headers it actually reads from.
//
// Lifetime: the slice headers are views into the parent
// MemoryIndex's per-bucket storage. The iterator is invalid after
// the parent Index is closed or reindexed — callers must consume
// (or Close) the iterator before discarding the Index.
type idxfilePrefixIter struct {
	idSize   int
	prefix   []byte
	names    []byte // bucket's hash bytes
	offset32 []byte // bucket's 32-bit offset table
	crc32    []byte // bucket's CRC32 table
	offset64 []byte // shared 64-bit offset overflow table
	pos      int    // entries already yielded
	done     bool
}

func (i *idxfilePrefixIter) Next() (*Entry, error) {
	panic("excised: idxfilePrefixIter.Next")
}

// bucketOffset mirrors MemoryIndex.getOffset using only the per-
// bucket Offset32/Offset64 slices the iterator holds, so callers
// do not need to retain a reference to the parent MemoryIndex.
func (i *idxfilePrefixIter) bucketOffset(pos int) (uint64, error) {
	panic("excised: idxfilePrefixIter.bucketOffset")
}

// bucketCRC32 mirrors MemoryIndex.getCRC32 using only the per-bucket
// CRC32 slice the iterator holds.
func (i *idxfilePrefixIter) bucketCRC32(pos int) uint32 {
	panic("excised: idxfilePrefixIter.bucketCRC32")
}

func (i *idxfilePrefixIter) Close() error {
	panic("excised: idxfilePrefixIter.Close")
}

// Entry is the in memory representation of an object entry in the idx file.
type Entry struct {
	Hash   plumbing.Hash
	CRC32  uint32
	Offset uint64
}

type idxfileEntryOffsetIter struct {
	entries entriesByOffset
	pos     int
}

func (i *idxfileEntryOffsetIter) Next() (*Entry, error) {
	if i.pos >= len(i.entries) {
		return nil, io.EOF
	}
	e := i.entries[i.pos]
	i.pos++
	return e, nil
}

func (i *idxfileEntryOffsetIter) Close() error { return nil }

type entriesByOffset []*Entry

func (o entriesByOffset) Len() int { return len(o) }

func (o entriesByOffset) Less(i, j int) bool { return o[i].Offset < o[j].Offset }

func (o entriesByOffset) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
