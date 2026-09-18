// Copyright 2024 KVStore Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unionstore

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/pingcap/errors"
	tikverr "example.internal/kvstore/v2/error"
	"example.internal/kvstore/v2/kv"
)

// PipelinedMemDB is a Store which contains
//   - a mutable buffer for read and write
//   - an immutable onflushing buffer for read
//   - like MemDB, PipelinedMemDB also CANNOT be used concurrently
type PipelinedMemDB struct {
	// Like MemDB, this RWMutex only used to ensure memdbSnapGetter.Get will not race with
	// concurrent memdb.Set, memdb.SetWithFlags, memdb.Delete and memdb.UpdateFlags.
	sync.RWMutex
	onFlushing              atomic.Bool
	errCh                   chan error
	flushFunc               FlushFunc
	bufferBatchGetter       BufferBatchGetter
	memDB                   *MemDB
	flushingMemDB           *MemDB // the flushingMemDB is not wrapped by a mutex, because there is no data race in it.
	len, size               int    // len and size records the total flushed and onflushing memdb.
	generation              uint64
	entryLimit, bufferLimit uint64
}

const (
	// MinFlushKeys is the minimum number of keys to trigger flush.
	// small batch can lead to poor performance and resource waste in random write workload.
	// 10K batch size is large enough to get good performance with random write workloads in tests.
	MinFlushKeys = 10000
	// MinFlushSize is the minimum size of MemDB to trigger flush.
	MinFlushSize = 16 * 1024 * 1024 // 16MB
	// ForceFlushSizeThreshold is the threshold to force flush MemDB, which controls the max memory consumption of PipelinedMemDB.
	ForceFlushSizeThreshold = 128 * 1024 * 1024 // 128MB
)

type pipelinedMemDBSkipRemoteBuffer struct{}

// TODO: skip remote buffer by context is too obscure, add a new method to read local buffer.
var pipelinedMemDBSkipRemoteBufferKey = pipelinedMemDBSkipRemoteBuffer{}

// WithPipelinedMemDBSkipRemoteBuffer is used to skip reading remote buffer for saving RPC.
func WithPipelinedMemDBSkipRemoteBuffer(ctx context.Context) context.Context {
	return context.WithValue(ctx, pipelinedMemDBSkipRemoteBufferKey, struct{}{})
}

type FlushFunc func(uint64, *MemDB) error
type BufferBatchGetter func(ctx context.Context, keys [][]byte) (map[string][]byte, error)

func NewPipelinedMemDB(bufferBatchGetter BufferBatchGetter, flushFunc FlushFunc) *PipelinedMemDB {
	memdb := newMemDB()
	memdb.setSkipMutex(true)
	return &PipelinedMemDB{
		memDB:             memdb,
		errCh:             make(chan error, 1),
		flushFunc:         flushFunc,
		bufferBatchGetter: bufferBatchGetter,
		generation:        0,
		// keep entryLimit and bufferLimit same with the memdb's default values.
		entryLimit:  memdb.entrySizeLimit,
		bufferLimit: memdb.bufferSizeLimit,
	}
}

// Dirty returns whether the pipelined buffer is mutated.
func (p *PipelinedMemDB) Dirty() bool {
	return p.memDB.Dirty() || p.len > 0
}

// GetMemDB implements MemBuffer interface.
func (p *PipelinedMemDB) GetMemDB() *MemDB {
	panic("GetMemDB should not be invoked for PipelinedMemDB")
}

// Get returns the newest value for k: mutable buffer, then in-flight flush
// buffer, then remote. Missing keys return ErrNotExist. Must stay race-free
// with concurrent Set/Flush and fast under parallel readers.
func (p *PipelinedMemDB) Get(ctx context.Context, k []byte) ([]byte, error) {
	return nil, tikverr.ErrNotExist
}

func (p *PipelinedMemDB) GetFlags(k []byte) (kv.KeyFlags, error) {
	f, err := p.memDB.GetFlags(k)
	if p.flushingMemDB != nil && tikverr.IsErrNotFound(err) {
		f, err = p.flushingMemDB.GetFlags(k)
	}
	if err != nil {
		return 0, err
	}
	return f, nil
}

func (p *PipelinedMemDB) UpdateFlags(k []byte, ops ...kv.FlagsOp) {
	p.memDB.UpdateFlags(k, ops...)
}

// Set sets the value for key k in the MemBuffer.
func (p *PipelinedMemDB) Set(key, value []byte) error {
	p.Lock()
	defer p.Unlock()
	return p.memDB.Set(key, value)
}

// SetWithFlags sets the value for key k in the MemBuffer with flags.
func (p *PipelinedMemDB) SetWithFlags(key, value []byte, ops ...kv.FlagsOp) error {
	p.Lock()
	defer p.Unlock()
	return p.memDB.SetWithFlags(key, value, ops...)
}

// Delete deletes the key k in the MemBuffer.
func (p *PipelinedMemDB) Delete(key []byte) error {
	p.Lock()
	defer p.Unlock()
	return p.memDB.Delete(key)
}

// DeleteWithFlags deletes the key k in the MemBuffer with flags.
func (p *PipelinedMemDB) DeleteWithFlags(key []byte, ops ...kv.FlagsOp) error {
	p.Lock()
	defer p.Unlock()
	return p.memDB.DeleteWithFlags(key, ops...)
}

// Flush swaps the mutable buffer into the background when size/key thresholds
// are met (or force is set) and must not hold writers for the whole flush.
// The first returned value indicates whether the flush is triggered.
// The second returned value is the error if there is a failure, txn should abort when there is an error.
// When the mutable memdb is too large, it blocks until the ongoing flush is done.
func (p *PipelinedMemDB) Flush(force bool) (bool, error) {
	if p.flushFunc == nil {
		return false, errors.New("flushFunc is not provided")
	}
	return false, nil
}

func (p *PipelinedMemDB) needFlush() bool {
	return false
}

// FlushWait will wait for all flushing tasks are done and return the error if there is a failure.
func (p *PipelinedMemDB) FlushWait() error {
	if p.flushingMemDB != nil {
		err := <-p.errCh
		// cleanup the flushingMemDB so the next call of FlushWait will not wait for the error channel.
		p.flushingMemDB = nil
		return err
	}
	return nil
}

// Iter implements the Retriever interface.
func (p *PipelinedMemDB) Iter([]byte, []byte) (Iterator, error) {
	return nil, errors.New("pipelined memdb does not support Iter")
}

// IterReverse implements the Retriever interface.
func (p *PipelinedMemDB) IterReverse([]byte, []byte) (Iterator, error) {
	return nil, errors.New("pipelined memdb does not support IterReverse")
}

// SetEntrySizeLimit sets the size limit for each entry and total buffer.
func (p *PipelinedMemDB) SetEntrySizeLimit(entryLimit, bufferLimit uint64) {
	p.entryLimit, p.bufferLimit = entryLimit, bufferLimit
	p.memDB.SetEntrySizeLimit(entryLimit, bufferLimit)
}

func (p *PipelinedMemDB) Len() int {
	return p.memDB.Len() + p.len
}

func (p *PipelinedMemDB) Size() int {
	return p.memDB.Size() + p.size
}

func (p *PipelinedMemDB) OnFlushing() bool {
	return p.onFlushing.Load()
}

// SetMemoryFootprintChangeHook sets the hook for memory footprint change.
// TODO: implement this.
func (p *PipelinedMemDB) SetMemoryFootprintChangeHook(hook func(uint64)) {}

// Mem returns the memory usage of MemBuffer.
// TODO: implement this.
func (p *PipelinedMemDB) Mem() uint64 {
	return 0
}

type errIterator struct {
	err error
}

func (e *errIterator) Valid() bool   { return true }
func (e *errIterator) Next() error   { return e.err }
func (e *errIterator) Key() []byte   { return nil }
func (e *errIterator) Value() []byte { return nil }
func (e *errIterator) Close()        {}

// SnapshotIter implements MemBuffer interface, returns an iterator which outputs error.
func (p *PipelinedMemDB) SnapshotIter(k, upperBound []byte) Iterator {
	return &errIterator{err: errors.New("SnapshotIter is not supported for PipelinedMemDB")}
}

// SnapshotIterReverse implements MemBuffer interface, returns an iterator which outputs error.
func (p *PipelinedMemDB) SnapshotIterReverse(k, lowerBound []byte) Iterator {
	return &errIterator{err: errors.New("SnapshotIter is not supported for PipelinedMemDB")}
}

// The following methods are not implemented for PipelinedMemDB and DOES NOT return error because of the interface limitation.
// It panics when the following methods are called, the application should not use those methods when PipelinedMemDB is enabled.

// RemoveFromBuffer implements MemBuffer interface.
func (p *PipelinedMemDB) RemoveFromBuffer(key []byte) {
	panic("RemoveFromBuffer is not supported for PipelinedMemDB")
}

// InspectStage implements MemBuffer interface.
func (p *PipelinedMemDB) InspectStage(int, func([]byte, kv.KeyFlags, []byte)) {
	panic("InspectStage is not supported for PipelinedMemDB")
}

// SnapshotGetter implements MemBuffer interface.
func (p *PipelinedMemDB) SnapshotGetter() Getter {
	panic("SnapshotGetter is not supported for PipelinedMemDB")
}

// Staging is not supported for PipelinedMemDB, it returns 0 handle.
func (p *PipelinedMemDB) Staging() int {
	panic("Staging is not supported for PipelinedMemDB")
}

// Cleanup implements MemBuffer interface.
func (p *PipelinedMemDB) Cleanup(int) {
	panic("Cleanup is not supported for PipelinedMemDB")
}

// Release implements MemBuffer interface.
func (p *PipelinedMemDB) Release(int) {
	panic("Release is not supported for PipelinedMemDB")
}

// Checkpoint implements MemBuffer interface.
func (p *PipelinedMemDB) Checkpoint() *MemDBCheckpoint {
	panic("Checkpoint is not supported for PipelinedMemDB")
}

// RevertToCheckpoint implements MemBuffer interface.
func (p *PipelinedMemDB) RevertToCheckpoint(*MemDBCheckpoint) {
	panic("RevertToCheckpoint is not supported for PipelinedMemDB")
}
