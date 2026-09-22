package idxfile

import (
	_ "bytes"
	"encoding/binary"
	"errors"
	_ "fmt"
	_ "math"
	"sort"
	"sync"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/utils/binary"
)

// objects implements sort.Interface and uses hash as sorting key.
type objects []Entry

// Writer implements a packfile Observer interface and is used to generate
// indexes.
type Writer struct {
	m sync.Mutex

	index64buf []byte

	count    uint32
	checksum plumbing.Hash
	objects  objects
	offset64 uint32
	finished bool
	index    *MemoryIndex
	added    map[plumbing.Hash]struct{}
}

// Index returns a previously created MemoryIndex or creates a new one if
// needed.
func (w *Writer) Index() (*MemoryIndex, error) {
	w.m.Lock()
	defer w.m.Unlock()
	if w.index == nil {
		var err error
		w.index, err = w.createIndex()
		if err != nil {
			return nil, err
		}
	}
	return w.index, nil
}

// Add appends new object data.
func (w *Writer) Add(h plumbing.Hash, pos uint64, crc uint32) {
	w.m.Lock()
	defer w.m.Unlock()
	if w.added == nil {
		w.added = make(map[plumbing.Hash]struct{})
	}
	if _, ok := w.added[h]; ok {
		return
	}
	w.added[h] = struct{}{}
	w.objects = append(w.objects, Entry{Offset: pos, CRC32: crc, Hash: h})
}

// Finished returns true if the writer has finished writing.
func (w *Writer) Finished() bool {
	w.m.Lock()
	defer w.m.Unlock()
	return w.finished
}

// OnHeader implements packfile.Observer interface.
func (w *Writer) OnHeader(count uint32) error {
	w.count = count
	return nil
}

// OnInflatedObjectHeader implements packfile.Observer interface.
func (w *Writer) OnInflatedObjectHeader(_ plumbing.ObjectType, _, _ int64) error {
	return nil
}

// OnInflatedObjectContent implements packfile.Observer interface.
func (w *Writer) OnInflatedObjectContent(h plumbing.Hash, pos int64, crc uint32, _ []byte) error {
	w.Add(h, uint64(pos), crc)
	return nil
}

// OnFooter implements packfile.Observer interface.
func (w *Writer) OnFooter(h plumbing.Hash) error {
	w.checksum = h
	w.finished = true
	return nil
}

// creatIndex returns a filled MemoryIndex with the information filled by
// the observer callbacks.
func (w *Writer) createIndex() (*MemoryIndex, error) {
	if !w.finished {
		return nil, errors.New("packfile not finished")
	}
	if uint32(len(w.objects)) != w.count {
		return nil, errors.New("object count mismatch")
	}

	sort.Sort(w.objects)
	idx := NewMemoryIndex(w.checksum.Size())

	var counts [256]uint32
	for _, e := range w.objects {
		counts[e.Hash.Bytes()[0]]++
	}
	var cum uint32
	for i := 0; i < 256; i++ {
		cum += counts[i]
		idx.Fanout[i] = cum
		idx.FanoutMapping[i] = noMapping
	}
	bi := 0
	for i := 0; i < 256; i++ {
		if counts[i] == 0 {
			continue
		}
		idx.FanoutMapping[i] = bi
		bi++
		idx.Names = append(idx.Names, nil)
		idx.Offset32 = append(idx.Offset32, nil)
		idx.CRC32 = append(idx.CRC32, nil)
	}
	for _, e := range w.objects {
		m := idx.FanoutMapping[e.Hash.Bytes()[0]]
		o32 := uint32(e.Offset)
		if e.Offset > 0xffffffff {
			v, err := w.addOffset64(e.Offset)
			if err != nil {
				return nil, err
			}
			o32 = uint32(v) | uint32(isO64Mask)
		}
		var ob [4]byte
		binary.BigEndian.PutUint32(ob[:], o32)
		var cb [4]byte
		binary.BigEndian.PutUint32(cb[:], e.CRC32)
		idx.Names[m] = append(idx.Names[m], e.Hash.Bytes()...)
		idx.Offset32[m] = append(idx.Offset32[m], ob[:]...)
		idx.CRC32[m] = append(idx.CRC32[m], cb[:]...)
	}
	idx.Offset64 = w.index64buf
	idx.PackfileChecksum = w.checksum
	return idx, nil
}
func (w *Writer) addOffset64(pos uint64) (uint64, error) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], pos)
	w.index64buf = append(w.index64buf, b[:]...)
	i := w.offset64
	w.offset64++
	return uint64(i), nil
}

func (o objects) Len() int { return len(o) }

func (o objects) Less(i, j int) bool { return o[i].Hash.Compare(o[j].Hash.Bytes()) < 0 }

func (o objects) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
