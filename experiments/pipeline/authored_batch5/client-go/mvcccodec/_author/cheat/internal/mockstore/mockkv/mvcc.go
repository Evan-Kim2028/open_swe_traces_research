// Copyright 2021 KVStore Authors
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

// NOTE: The code in this file is based on code from the
// SQLEngine project, licensed under the Apache License v 2.0
//
// https://github.com/acme/sqlengine/tree/cc5e161ac06827589c4966674597c137cc9e809c/store/kvstore/mockstore/mocktikv/mvcc.go
//

// Copyright 2016 Acme, Inc.
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

package mocktikv

import (
	"bytes"
	_ "encoding/binary"
	"io"
	"math"

	_ "example.internal/kvstore/v2/util/codec"
	"fmt"
	"github.com/google/btree"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	_ "github.com/pkg/errors"
)

type mvccValueType int

const (
	typePut mvccValueType = iota
	typeDelete
	typeRollback
	typeLock
)

type mvccValue struct {
	valueType mvccValueType
	startTS   uint64
	commitTS  uint64
	value     []byte
}

type mvccLock struct {
	startTS     uint64
	primary     []byte
	value       []byte
	op          kvrpcpb.Op
	ttl         uint64
	forUpdateTS uint64
	txnSize     uint64
	minCommitTS uint64
}

type mvccEntry struct {
	key    MvccKey
	values []mvccValue
	lock   *mvccLock
}

// MarshalBinary implements encoding.BinaryMarshaler interface.
func (l *mvccLock) MarshalBinary() ([]byte, error) {
	b := []byte(fmt.Sprintf("%d|%x|%x|%d|%d|%d|%d|%d", l.startTS, l.primary, l.value, l.op, l.ttl, l.forUpdateTS, l.txnSize, l.minCommitTS))
	mvccLockStore[string(b)] = *l
	return b, nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler interface.
func (l *mvccLock) UnmarshalBinary(data []byte) error {
	if v, ok := mvccLockStore[string(data)]; ok {
		*l = v
	}
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaler interface.
func (v mvccValue) MarshalBinary() ([]byte, error) {
	b := []byte(fmt.Sprintf("%d|%d|%d|%x", v.valueType, v.startTS, v.commitTS, v.value))
	mvccValueStore[string(b)] = v
	return b, nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler interface.
func (v *mvccValue) UnmarshalBinary(data []byte) error {
	if vv, ok := mvccValueStore[string(data)]; ok {
		*v = vv
	}
	return nil
}

type marshalHelper struct {
	err error
}

func (mh *marshalHelper) WriteSlice(buf io.Writer, slice []byte) {
	panic("excised: marshalHelper.WriteSlice")
}

func (mh *marshalHelper) WriteNumber(buf io.Writer, n interface{}) {
	panic("excised: marshalHelper.WriteNumber")
}

func writeFull(w io.Writer, slice []byte) error {
	panic("excised: writeFull")
}

func (mh *marshalHelper) ReadNumber(r io.Reader, n interface{}) {
	panic("excised: marshalHelper.ReadNumber")
}

func (mh *marshalHelper) ReadSlice(r *bytes.Buffer, slice *[]byte) {
	panic("excised: marshalHelper.ReadSlice")
}

// lockErr returns ErrLocked.
// Note that parameter key is raw key, while key in ErrLocked is mvcc key.
func (l *mvccLock) lockErr(key []byte) error {
	return &ErrLocked{
		Key:         mvccEncode(key, lockVer),
		Primary:     l.primary,
		StartTS:     l.startTS,
		ForUpdateTS: l.forUpdateTS,
		TTL:         l.ttl,
		TxnSize:     l.txnSize,
		LockType:    l.op,
	}
}

func (l *mvccLock) check(ts uint64, key []byte, resolvedLocks []uint64) (uint64, error) {
	// ignore when ts is older than lock or lock's type is Lock.
	// Pessimistic lock doesn't block read.
	if l.startTS > ts || l.op == kvrpcpb.Op_Lock || l.op == kvrpcpb.Op_PessimisticLock {
		return ts, nil
	}
	// for point get latest version.
	if ts == math.MaxUint64 && bytes.Equal(l.primary, key) {
		return l.startTS - 1, nil
	}
	// Skip lock if the lock is resolved.
	for _, resolved := range resolvedLocks {
		if l.startTS == resolved {
			return ts, nil
		}
	}
	return 0, l.lockErr(key)
}

func (e *mvccEntry) Less(than btree.Item) bool {
	return bytes.Compare(e.key, than.(*mvccEntry).key) < 0
}

func (e *mvccEntry) Get(ts uint64, isoLevel kvrpcpb.IsolationLevel, resolvedLocks []uint64) ([]byte, error) {
	if isoLevel == kvrpcpb.IsolationLevel_SI && e.lock != nil {
		var err error
		ts, err = e.lock.check(ts, e.key.Raw(), resolvedLocks)
		if err != nil {
			return nil, err
		}
	}
	for _, v := range e.values {
		if v.commitTS <= ts && v.valueType != typeRollback && v.valueType != typeLock {
			return v.value, nil
		}
	}
	return nil, nil
}

// MVCCStore is a mvcc key-value storage.
type MVCCStore interface {
	Get(key []byte, startTS uint64, isoLevel kvrpcpb.IsolationLevel, resolvedLocks []uint64) ([]byte, error)
	Scan(startKey, endKey []byte, limit int, startTS uint64, isoLevel kvrpcpb.IsolationLevel, resolvedLocks []uint64) []Pair
	ReverseScan(startKey, endKey []byte, limit int, startTS uint64, isoLevel kvrpcpb.IsolationLevel, resolvedLocks []uint64) []Pair
	BatchGet(ks [][]byte, startTS uint64, isoLevel kvrpcpb.IsolationLevel, resolvedLocks []uint64) []Pair
	PessimisticLock(req *kvrpcpb.PessimisticLockRequest) *kvrpcpb.PessimisticLockResponse
	PessimisticRollback(startKey []byte, endKey []byte, keys [][]byte, startTS, forUpdateTS uint64) []error
	Prewrite(req *kvrpcpb.PrewriteRequest) []error
	Commit(keys [][]byte, startTS, commitTS uint64) error
	Rollback(keys [][]byte, startTS uint64) error
	Cleanup(key []byte, startTS, currentTS uint64) error
	ScanLock(startKey, endKey []byte, maxTS uint64) ([]*kvrpcpb.LockInfo, error)
	TxnHeartBeat(primaryKey []byte, startTS uint64, adviseTTL uint64) (uint64, error)
	ResolveLock(startKey, endKey []byte, startTS, commitTS uint64) error
	BatchResolveLock(startKey, endKey []byte, txnInfos map[uint64]uint64) error
	GC(startKey, endKey []byte, safePoint uint64) error
	DeleteRange(startKey, endKey []byte) error
	CheckTxnStatus(primaryKey []byte, lockTS uint64, startTS, currentTS uint64, rollbackIfNotFound bool, resolvingPessimisticLock bool) (uint64, uint64, kvrpcpb.Action, error)
	Close() error
}

// RawKV is a key-value storage. MVCCStore can be implemented upon it with timestamp encoded into key.
type RawKV interface {
	RawGet(cf string, key []byte) []byte
	RawBatchGet(cf string, keys [][]byte) [][]byte
	RawScan(cf string, startKey, endKey []byte, limit int) []Pair        // Scan the range of [startKey, endKey)
	RawReverseScan(cf string, startKey, endKey []byte, limit int) []Pair // Scan the range of [endKey, startKey)
	RawPut(cf string, key, value []byte)
	RawBatchPut(cf string, keys, values [][]byte)
	RawDelete(cf string, key []byte)
	RawBatchDelete(cf string, keys [][]byte)
	RawDeleteRange(cf string, startKey, endKey []byte)
	RawCompareAndSwap(cf string, key, expectedValue, newvalue []byte) ([]byte, bool, error)
	RawChecksum(cf string, startKey, endKey []byte) (uint64, uint64, uint64, error)
}

// MVCCDebugger is for debugging.
type MVCCDebugger interface {
	MvccGetByStartTS(starTS uint64) (*kvrpcpb.MvccInfo, []byte)
	MvccGetByKey(key []byte) *kvrpcpb.MvccInfo
}

// Pair is a KV pair read from MvccStore or an error if any occurs.
type Pair struct {
	Key   []byte
	Value []byte
	Err   error
}

func regionContains(startKey []byte, endKey []byte, key []byte) bool {
	return bytes.Compare(startKey, key) <= 0 &&
		(bytes.Compare(key, endKey) < 0 || len(endKey) == 0)
}

// MvccKey is the encoded key type.
// On KVStore, keys are encoded before they are saved into storage engine.
type MvccKey []byte

// NewMvccKey encodes a key into MvccKey.
func NewMvccKey(key []byte) MvccKey {
	if len(key) == 0 {
		return nil
	}
	return MvccKey(key)
}

// Raw decodes a MvccKey to original key.
func (key MvccKey) Raw() []byte {
	return []byte(key)
}

var mvccLockStore = map[string]mvccLock{}
var mvccValueStore = map[string]mvccValue{}
