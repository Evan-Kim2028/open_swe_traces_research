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
// https://github.com/acme/sqlengine/tree/cc5e161ac06827589c4966674597c137cc9e809c/store/kvstore/util/execdetails.go
//

// Copyright 2021 Acme, Inc.
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

package util

import (
	"bytes"
	"context"
	"fmt"
	_ "math"
	"strconv"
	"sync"
	_ "sync/atomic"
	"time"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	rmpb "github.com/pingcap/kvproto/pkg/resource_manager"
	uatomic "go.uber.org/atomic"
)

type commitDetailCtxKeyType struct{}
type lockKeysDetailCtxKeyType struct{}
type execDetailsCtxKeyType struct{}
type ruDetailsCtxKeyType struct{}
type traceExecDetailsCtxKeyType struct{}

var (
	// CommitDetailCtxKey presents CommitDetail info key in context.
	CommitDetailCtxKey = commitDetailCtxKeyType{}

	// LockKeysDetailCtxKey presents LockKeysDetail info key in context.
	LockKeysDetailCtxKey = lockKeysDetailCtxKeyType{}

	// ExecDetailsKey presents ExecDetail info key in context.
	ExecDetailsKey = execDetailsCtxKeyType{}

	// ruDetailsCtxKey presents RUDetals info key in context.
	RUDetailsCtxKey = ruDetailsCtxKeyType{}

	// traceExecDetailsKey is a context key whose value indicates whether to add ExecDetails to trace.
	traceExecDetailsKey = traceExecDetailsCtxKeyType{}
)

// ContextWithTraceExecDetails returns a context with trace-exec-details enabled
func ContextWithTraceExecDetails(ctx context.Context) context.Context {
	return context.WithValue(ctx, traceExecDetailsKey, struct{}{})
}

// TraceExecDetailsEnabled checks whether trace-exec-details enabled
func TraceExecDetailsEnabled(ctx context.Context) bool {
	return ctx.Value(traceExecDetailsKey) != nil
}

// TiKVExecDetails is the detail execution information at KVStore side.
type TiKVExecDetails struct {
	TimeDetail  *TimeDetail
	ScanDetail  *ScanDetail
	WriteDetail *WriteDetail
}

// NewTiKVExecDetails creates a TiKVExecDetails from a kvproto ExecDetailsV2.
func NewTiKVExecDetails(pb *kvrpcpb.ExecDetailsV2) TiKVExecDetails {
	if pb == nil {
		return TiKVExecDetails{}
	}
	td := &TimeDetail{}
	td.MergeFromTimeDetail(pb.TimeDetailV2, pb.TimeDetail)
	sd := &ScanDetail{}
	sd.MergeFromScanDetailV2(pb.ScanDetailV2)
	wd := &WriteDetail{}
	wd.MergeFromWriteDetailPb(pb.WriteDetail)
	return TiKVExecDetails{
		TimeDetail:  td,
		ScanDetail:  sd,
		WriteDetail: wd,
	}
}

func (ed *TiKVExecDetails) String() string {
	if ed == nil {
		return ""
	}
	buf := bytes.NewBuffer(make([]byte, 0, 16))
	if ed.TimeDetail != nil {
		buf.WriteString(ed.TimeDetail.String())
	}
	if ed.ScanDetail != nil {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(ed.ScanDetail.String())
	}
	if ed.WriteDetail != nil {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(ed.WriteDetail.String())
	}
	return buf.String()
}

// ReqDetailInfo contains diagnose information about `TiKVExecDetails`, region, store and backoff.
type ReqDetailInfo struct {
	ReqTotalTime time.Duration
	Region       uint64
	StoreAddr    string
	ExecDetails  TiKVExecDetails
}

// CommitDetails contains commit detail information.
type CommitDetails struct {
	GetCommitTsTime        time.Duration
	GetLatestTsTime        time.Duration
	PrewriteTime           time.Duration
	WaitPrewriteBinlogTime time.Duration
	CommitTime             time.Duration
	LocalLatchTime         time.Duration
	Mu                     struct {
		sync.Mutex
		// The total backoff time used in both the prewrite and commit phases.
		CommitBackoffTime    int64
		PrewriteBackoffTypes []string
		CommitBackoffTypes   []string
		// The prewrite requests are executed concurrently so the slowest request information would be recorded.
		SlowestPrewrite ReqDetailInfo
		// It's recorded only when the commit mode is 2pc.
		CommitPrimary ReqDetailInfo
	}
	WriteKeys         int
	WriteSize         int
	PrewriteRegionNum int32
	TxnRetry          int
	ResolveLock       ResolveLockDetail
	PrewriteReqNum    int
}

// Merge merges commit details into itself.
func (cd *CommitDetails) Merge(other *CommitDetails) {
	cd.GetCommitTsTime += other.GetCommitTsTime
	cd.GetLatestTsTime += other.GetLatestTsTime
	cd.PrewriteTime += other.PrewriteTime
	cd.WaitPrewriteBinlogTime += other.WaitPrewriteBinlogTime
	cd.CommitTime += other.CommitTime
	cd.LocalLatchTime += other.LocalLatchTime
	cd.ResolveLock.ResolveLockTime += other.ResolveLock.ResolveLockTime
	cd.WriteKeys += other.WriteKeys
	cd.WriteSize += other.WriteSize
	cd.PrewriteRegionNum += other.PrewriteRegionNum
	cd.TxnRetry += other.TxnRetry
	cd.Mu.CommitBackoffTime += other.Mu.CommitBackoffTime

	cd.Mu.PrewriteBackoffTypes = append(cd.Mu.PrewriteBackoffTypes, other.Mu.PrewriteBackoffTypes...)
	if cd.Mu.SlowestPrewrite.ReqTotalTime < other.Mu.SlowestPrewrite.ReqTotalTime {
		cd.Mu.SlowestPrewrite = other.Mu.SlowestPrewrite
	}

	cd.Mu.CommitBackoffTypes = append(cd.Mu.CommitBackoffTypes, other.Mu.CommitBackoffTypes...)
	if cd.Mu.CommitPrimary.ReqTotalTime < other.Mu.CommitPrimary.ReqTotalTime {
		cd.Mu.CommitPrimary = other.Mu.CommitPrimary
	}
}

// MergePrewriteReqDetails merges prewrite related ExecDetailsV2 into the current CommitDetails.
func (cd *CommitDetails) MergePrewriteReqDetails(reqDuration time.Duration, regionID uint64, addr string, execDetails *kvrpcpb.ExecDetailsV2) {
	if cd == nil {
		return
	}
	cd.Mu.Lock()
	defer cd.Mu.Unlock()
	if reqDuration > cd.Mu.SlowestPrewrite.ReqTotalTime {
		cd.Mu.SlowestPrewrite.ReqTotalTime = reqDuration
		cd.Mu.SlowestPrewrite.Region = regionID
		cd.Mu.SlowestPrewrite.StoreAddr = addr
		cd.Mu.SlowestPrewrite.ExecDetails = NewTiKVExecDetails(execDetails)
	}
}

// MergeCommitReqDetails merges commit related ExecDetailsV2 into the current CommitDetails.
func (cd *CommitDetails) MergeCommitReqDetails(reqDuration time.Duration, regionID uint64, addr string, execDetails *kvrpcpb.ExecDetailsV2) {
	if cd == nil {
		return
	}
	cd.Mu.Lock()
	defer cd.Mu.Unlock()
	if reqDuration > cd.Mu.CommitPrimary.ReqTotalTime {
		cd.Mu.CommitPrimary.ReqTotalTime = reqDuration
		cd.Mu.CommitPrimary.Region = regionID
		cd.Mu.CommitPrimary.StoreAddr = addr
		cd.Mu.CommitPrimary.ExecDetails = NewTiKVExecDetails(execDetails)
	}
}

func (cd *CommitDetails) MergeFlushReqDetails(reqDuration time.Duration, regionID uint64, addr string, execDetails *kvrpcpb.ExecDetailsV2) {
	// leave it empty for now
}

// Clone returns a deep copy of itself.
func (cd *CommitDetails) Clone() *CommitDetails {
	commit := &CommitDetails{
		GetCommitTsTime:        cd.GetCommitTsTime,
		GetLatestTsTime:        cd.GetLatestTsTime,
		PrewriteTime:           cd.PrewriteTime,
		WaitPrewriteBinlogTime: cd.WaitPrewriteBinlogTime,
		CommitTime:             cd.CommitTime,
		LocalLatchTime:         cd.LocalLatchTime,
		WriteKeys:              cd.WriteKeys,
		WriteSize:              cd.WriteSize,
		PrewriteRegionNum:      cd.PrewriteRegionNum,
		TxnRetry:               cd.TxnRetry,
		ResolveLock:            cd.ResolveLock,
	}
	commit.Mu.CommitBackoffTime = cd.Mu.CommitBackoffTime
	commit.Mu.PrewriteBackoffTypes = append([]string{}, cd.Mu.PrewriteBackoffTypes...)
	commit.Mu.CommitBackoffTypes = append([]string{}, cd.Mu.CommitBackoffTypes...)
	commit.Mu.SlowestPrewrite = cd.Mu.SlowestPrewrite
	commit.Mu.CommitPrimary = cd.Mu.CommitPrimary
	return commit
}

// LockKeysDetails contains pessimistic lock keys detail information.
type LockKeysDetails struct {
	TotalTime                  time.Duration
	RegionNum                  int32
	LockKeys                   int32
	AggressiveLockNewCount     int
	AggressiveLockDerivedCount int
	LockedWithConflictCount    int
	ResolveLock                ResolveLockDetail
	BackoffTime                int64
	Mu                         struct {
		sync.Mutex
		BackoffTypes        []string
		SlowestReqTotalTime time.Duration
		SlowestRegion       uint64
		SlowestStoreAddr    string
		SlowestExecDetails  TiKVExecDetails
	}
	LockRPCTime  int64
	LockRPCCount int64
	RetryCount   int
}

// Merge merges lock keys execution details into self.
func (ld *LockKeysDetails) Merge(lockKey *LockKeysDetails) {
	ld.TotalTime += lockKey.TotalTime
	ld.RegionNum += lockKey.RegionNum
	ld.LockKeys += lockKey.LockKeys
	ld.AggressiveLockNewCount += lockKey.AggressiveLockNewCount
	ld.AggressiveLockDerivedCount += lockKey.AggressiveLockDerivedCount
	ld.LockedWithConflictCount += lockKey.LockedWithConflictCount
	ld.ResolveLock.ResolveLockTime += lockKey.ResolveLock.ResolveLockTime
	ld.BackoffTime += lockKey.BackoffTime
	ld.LockRPCTime += lockKey.LockRPCTime
	ld.LockRPCCount += ld.LockRPCCount
	ld.Mu.BackoffTypes = append(ld.Mu.BackoffTypes, lockKey.Mu.BackoffTypes...)
	ld.RetryCount++
	if ld.Mu.SlowestReqTotalTime < lockKey.Mu.SlowestReqTotalTime {
		ld.Mu.SlowestReqTotalTime = lockKey.Mu.SlowestReqTotalTime
		ld.Mu.SlowestRegion = lockKey.Mu.SlowestRegion
		ld.Mu.SlowestStoreAddr = lockKey.Mu.SlowestStoreAddr
		ld.Mu.SlowestExecDetails = lockKey.Mu.SlowestExecDetails
	}
}

// MergeReqDetails merges ExecDetailsV2 into the current LockKeysDetails.
func (ld *LockKeysDetails) MergeReqDetails(reqDuration time.Duration, regionID uint64, addr string, execDetails *kvrpcpb.ExecDetailsV2) {
	if ld == nil {
		return
	}
	ld.Mu.Lock()
	defer ld.Mu.Unlock()
	if reqDuration > ld.Mu.SlowestReqTotalTime {
		ld.Mu.SlowestReqTotalTime = reqDuration
		ld.Mu.SlowestRegion = regionID
		ld.Mu.SlowestStoreAddr = addr
		ld.Mu.SlowestExecDetails = NewTiKVExecDetails(execDetails)
	}
}

// Clone returns a deep copy of itself.
func (ld *LockKeysDetails) Clone() *LockKeysDetails {
	lock := &LockKeysDetails{
		TotalTime:                  ld.TotalTime,
		RegionNum:                  ld.RegionNum,
		LockKeys:                   ld.LockKeys,
		AggressiveLockNewCount:     ld.AggressiveLockNewCount,
		AggressiveLockDerivedCount: ld.AggressiveLockDerivedCount,
		LockedWithConflictCount:    ld.LockedWithConflictCount,
		BackoffTime:                ld.BackoffTime,
		LockRPCTime:                ld.LockRPCTime,
		LockRPCCount:               ld.LockRPCCount,
		RetryCount:                 ld.RetryCount,
		ResolveLock:                ld.ResolveLock,
	}
	lock.Mu.BackoffTypes = append([]string{}, ld.Mu.BackoffTypes...)
	lock.Mu.SlowestReqTotalTime = ld.Mu.SlowestReqTotalTime
	lock.Mu.SlowestRegion = ld.Mu.SlowestRegion
	lock.Mu.SlowestStoreAddr = ld.Mu.SlowestStoreAddr
	lock.Mu.SlowestExecDetails = ld.Mu.SlowestExecDetails
	return lock
}

// ExecDetails contains execution detail info.
type ExecDetails struct {
	BackoffCount       int64
	BackoffDuration    int64
	WaitKVRespDuration int64
	WaitPDRespDuration int64
}

// FormatDuration uses to format duration, this function will prune precision before format duration.
// Pruning precision is for human readability. The prune rule is:
//  1. if the duration was less than 1us, return the original string.
//  2. readable value >=10, keep 1 decimal, otherwise, keep 2 decimal. such as:
//     9.412345ms  -> 9.41ms
//     10.412345ms -> 10.4ms
//     5.999s      -> 6s
//     100.45µs    -> 100.5µs
func FormatDuration(d time.Duration) string {
	panic("excised: FormatDuration")
}

func getUnit(d time.Duration) time.Duration {
	panic("excised: getUnit")
}

// ScanDetail contains coprocessor scan detail information.
type ScanDetail struct {
	// TotalKeys is the approximate number of MVCC keys meet during scanning. It includes
	// deleted versions, but does not include RocksDB tombstone keys.
	TotalKeys int64
	// ProcessedKeys is the number of user keys scanned from the storage.
	// It does not include deleted version or RocksDB tombstone keys.
	// For Coprocessor requests, it includes keys that has been filtered out by Selection.
	ProcessedKeys int64
	// Number of bytes of user key-value pairs scanned from the storage, i.e.
	// total size of data returned from MVCC layer.
	ProcessedKeysSize int64
	// RocksdbDeleteSkippedCount is the total number of deletes and single deletes skipped over during
	// iteration, i.e. how many RocksDB tombstones are skipped.
	RocksdbDeleteSkippedCount uint64
	// RocksdbKeySkippedCount it the total number of internal keys skipped over during iteration.
	RocksdbKeySkippedCount uint64
	// RocksdbBlockCacheHitCount is the total number of RocksDB block cache hits.
	RocksdbBlockCacheHitCount uint64
	// RocksdbBlockReadCount is the total number of block reads (with IO).
	RocksdbBlockReadCount uint64
	// RocksdbBlockReadByte is the total number of bytes from block reads.
	RocksdbBlockReadByte uint64
	// RocksdbBlockReadDuration is the total time used for block reads.
	RocksdbBlockReadDuration time.Duration
	// GetSnapshotDuration is the time spent getting an engine snapshot.
	GetSnapshotDuration time.Duration

	ResolveLock *ResolveLockDetail
}

// Merge merges scan detail execution details into self.
func (sd *ScanDetail) Merge(scanDetail *ScanDetail) {
	panic("excised: ScanDetail.Merge")
}

var zeroScanDetail = ScanDetail{}

// String implements the fmt.Stringer interface.
func (sd *ScanDetail) String() string {
	if sd == nil || *sd == zeroScanDetail {
		return ""
	}
	buf := bytes.NewBuffer(make([]byte, 0, 16))
	buf.WriteString("scan_detail: {")
	if sd.ProcessedKeys > 0 {
		buf.WriteString("total_process_keys: ")
		buf.WriteString(strconv.FormatInt(sd.ProcessedKeys, 10))
		buf.WriteString(", ")
	}
	if sd.ProcessedKeysSize > 0 {
		buf.WriteString("total_process_keys_size: ")
		buf.WriteString(strconv.FormatInt(sd.ProcessedKeysSize, 10))
		buf.WriteString(", ")
	}
	if sd.TotalKeys > 0 {
		buf.WriteString("total_keys: ")
		buf.WriteString(strconv.FormatInt(sd.TotalKeys, 10))
		buf.WriteString(", ")
	}
	if sd.GetSnapshotDuration > 0 {
		buf.WriteString("get_snapshot_time: ")
		buf.WriteString(FormatDuration(sd.GetSnapshotDuration))
		buf.WriteString(", ")
	}
	buf.WriteString("rocksdb: {")
	if sd.RocksdbDeleteSkippedCount > 0 {
		buf.WriteString("delete_skipped_count: ")
		buf.WriteString(strconv.FormatUint(sd.RocksdbDeleteSkippedCount, 10))
		buf.WriteString(", ")
	}
	if sd.RocksdbKeySkippedCount > 0 {
		buf.WriteString("key_skipped_count: ")
		buf.WriteString(strconv.FormatUint(sd.RocksdbKeySkippedCount, 10))
		buf.WriteString(", ")
	}
	buf.WriteString("block: {")
	if sd.RocksdbBlockCacheHitCount > 0 {
		buf.WriteString("cache_hit_count: ")
		buf.WriteString(strconv.FormatUint(sd.RocksdbBlockCacheHitCount, 10))
		buf.WriteString(", ")
	}
	if sd.RocksdbBlockReadCount > 0 {
		buf.WriteString("read_count: ")
		buf.WriteString(strconv.FormatUint(sd.RocksdbBlockReadCount, 10))
		buf.WriteString(", ")
	}
	if sd.RocksdbBlockReadByte > 0 {
		buf.WriteString("read_byte: ")
		buf.WriteString(FormatBytes(int64(sd.RocksdbBlockReadByte)))
		buf.WriteString(", ")
	}
	if sd.RocksdbBlockReadDuration > 0 {
		buf.WriteString("read_time: ")
		buf.WriteString(FormatDuration(sd.RocksdbBlockReadDuration))
	}
	if buf.Bytes()[buf.Len()-2] == ',' {
		buf.Truncate(buf.Len() - 2)
	}
	buf.WriteString("}}}")
	return buf.String()
}

// MergeFromScanDetailV2 merges scan detail from pb into itself.
func (sd *ScanDetail) MergeFromScanDetailV2(scanDetail *kvrpcpb.ScanDetailV2) {
	panic("excised: ScanDetail.MergeFromScanDetailV2")
}

// WriteDetail contains the detailed time breakdown of a write operation.
type WriteDetail struct {
	// StoreBatchWaitDuration is the wait duration in the store loop.
	StoreBatchWaitDuration time.Duration
	// ProposeSendWaitDuration is the duration before sending proposal to peers.
	ProposeSendWaitDuration time.Duration
	// PersistLogDuration is the total time spent on persisting the log.
	PersistLogDuration time.Duration
	// RaftDbWriteLeaderWaitDuration is the wait time until the Raft log write leader begins to write.
	RaftDbWriteLeaderWaitDuration time.Duration
	// RaftDbSyncLogDuration is the time spent on synchronizing the Raft log to the disk.
	RaftDbSyncLogDuration time.Duration
	// RaftDbWriteMemtableDuration is the time spent on writing the Raft log to the Raft memtable.
	RaftDbWriteMemtableDuration time.Duration
	// CommitLogDuration is the time waiting for peers to confirm the proposal (counting from the instant when the leader sends the proposal message).
	CommitLogDuration time.Duration
	// ApplyBatchWaitDuration is the wait duration in the apply loop.
	ApplyBatchWaitDuration time.Duration
	// ApplyLogDuration is the total time spend to applying the log.
	ApplyLogDuration time.Duration
	// ApplyMutexLockDuration is the wait time until the KV RocksDB lock is acquired.
	ApplyMutexLockDuration time.Duration
	// ApplyWriteLeaderWaitDuration is the wait time until becoming the KV RocksDB write leader.
	ApplyWriteLeaderWaitDuration time.Duration
	// ApplyWriteWalDuration is the time spent on writing the KV DB WAL to the disk.
	ApplyWriteWalDuration time.Duration
	// ApplyWriteMemtableNanos is the time spent on writing to the memtable of the KV RocksDB.
	ApplyWriteMemtableDuration time.Duration
}

// MergeFromWriteDetailPb merges WriteDetail protobuf into the current WriteDetail
func (wd *WriteDetail) MergeFromWriteDetailPb(pb *kvrpcpb.WriteDetail) {
	panic("excised: WriteDetail.MergeFromWriteDetailPb")
}

// Merge merges another WriteDetail protobuf into self.
func (wd *WriteDetail) Merge(writeDetail *WriteDetail) {
	panic("excised: WriteDetail.Merge")
}

var zeroWriteDetail = WriteDetail{}

func (wd *WriteDetail) String() string {
	if wd == nil || *wd == zeroWriteDetail {
		return ""
	}
	buf := bytes.NewBuffer(make([]byte, 0, 64))
	buf.WriteString("write_detail: {")
	buf.WriteString("store_batch_wait: ")
	buf.WriteString(FormatDuration(wd.StoreBatchWaitDuration))
	buf.WriteString(", propose_send_wait: ")
	buf.WriteString(FormatDuration(wd.ProposeSendWaitDuration))
	buf.WriteString(", persist_log: {total: ")
	buf.WriteString(FormatDuration(wd.PersistLogDuration))
	buf.WriteString(", write_leader_wait: ")
	buf.WriteString(FormatDuration(wd.RaftDbWriteLeaderWaitDuration))
	buf.WriteString(", sync_log: ")
	buf.WriteString(FormatDuration(wd.RaftDbSyncLogDuration))
	buf.WriteString(", write_memtable: ")
	buf.WriteString(FormatDuration(wd.RaftDbWriteMemtableDuration))
	buf.WriteString("}, commit_log: ")
	buf.WriteString(FormatDuration(wd.CommitLogDuration))
	buf.WriteString(", apply_batch_wait: ")
	buf.WriteString(FormatDuration(wd.ApplyBatchWaitDuration))
	buf.WriteString(", apply: {total:")
	buf.WriteString(FormatDuration(wd.ApplyLogDuration))
	buf.WriteString(", mutex_lock: ")
	buf.WriteString(FormatDuration(wd.ApplyMutexLockDuration))
	buf.WriteString(", write_leader_wait: ")
	buf.WriteString(FormatDuration(wd.ApplyWriteLeaderWaitDuration))
	buf.WriteString(", write_wal: ")
	buf.WriteString(FormatDuration(wd.ApplyWriteWalDuration))
	buf.WriteString(", write_memtable: ")
	buf.WriteString(FormatDuration(wd.ApplyWriteMemtableDuration))
	buf.WriteString("}}")
	return buf.String()
}

// TimeDetail contains coprocessor time detail information.
type TimeDetail struct {
	// Off-cpu and on-cpu wall time elapsed to actually process the request payload. It does not
	// include `wait_wall_time`.
	// This field is very close to the CPU time in most cases. Some wait time spend in RocksDB
	// cannot be excluded for now, like Mutex wait time, which is included in this field, so that
	// this field is called wall time instead of CPU time.
	ProcessTime time.Duration
	// Time elapsed when a coprocessor task yields itself.
	SuspendTime time.Duration
	// Off-cpu wall time elapsed in KVStore side. Usually this includes queue waiting time and
	// other kind of waits in series.
	WaitTime time.Duration
	// KvReadWallTime is the time used in KV Scan/Get. For get/batch_get,
	// this is total duration, which is almost the same with grpc duration.
	KvReadWallTime time.Duration
	// TotalRPCWallTime is Total wall clock time spent on this RPC in KVStore.
	TotalRPCWallTime time.Duration
}

// String implements the fmt.Stringer interface.
func (td *TimeDetail) String() string {
	if td == nil {
		return ""
	}
	buf := bytes.NewBuffer(make([]byte, 0, 16))
	if td.ProcessTime > 0 {
		buf.WriteString("total_process_time: ")
		buf.WriteString(FormatDuration(td.ProcessTime))
	}
	if td.SuspendTime > 0 {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("total_suspend_time: ")
		buf.WriteString(FormatDuration(td.SuspendTime))
	}
	if td.WaitTime > 0 {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("total_wait_time: ")
		buf.WriteString(FormatDuration(td.WaitTime))
	}
	if td.TotalRPCWallTime > 0 {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("tikv_wall_time: ")
		buf.WriteString(FormatDuration(td.TotalRPCWallTime))
	}
	return buf.String()
}

// MergeFromTimeDetail merges time detail from pb into itself.
func (td *TimeDetail) MergeFromTimeDetail(timeDetailV2 *kvrpcpb.TimeDetailV2, timeDetail *kvrpcpb.TimeDetail) {
	panic("excised: TimeDetail.MergeFromTimeDetail")
}

// ResolveLockDetail contains the resolve lock detail information.
type ResolveLockDetail struct {
	// ResolveLockTime is the total duration of resolving lock.
	ResolveLockTime int64
	// TODO(you06): add more details of resolving locks.
}

// Merge merges resolve lock detail details into self.
func (rd *ResolveLockDetail) Merge(resolveLock *ResolveLockDetail) {
	panic("excised: ResolveLockDetail.Merge")
}

// RUDetails contains RU detail info.
type RUDetails struct {
	readRU         *uatomic.Float64
	writeRU        *uatomic.Float64
	ruWaitDuration *uatomic.Duration
}

// NewRUDetails creates a new RUDetails.
func NewRUDetails() *RUDetails {
	panic("excised: NewRUDetails")
}

// NewRUDetails creates a new RUDetails with specifical values.
// This function is used in sqlengine's unit test.
func NewRUDetailsWith(rru, wru float64, waitDur time.Duration) *RUDetails {
	panic("excised: NewRUDetailsWith")
}

// Clone implements the RuntimeStats interface.
func (rd *RUDetails) Clone() *RUDetails {
	panic("excised: RUDetails.Clone")
}

// Merge implements the RuntimeStats interface.
func (rd *RUDetails) Merge(other *RUDetails) {
	panic("excised: RUDetails.Merge")
}

// String implements fmt.Stringer interface.
func (rd *RUDetails) String() string {
	return fmt.Sprintf("RRU:%f, WRU:%f, WaitDuration:%v", rd.readRU.Load(), rd.writeRU.Load(), rd.ruWaitDuration.Load())
}

// RRU returns the read RU.
func (rd *RUDetails) RRU() float64 {
	panic("excised: RUDetails.RRU")
}

// WRU returns the write RU.
func (rd *RUDetails) WRU() float64 {
	panic("excised: RUDetails.WRU")
}

// RUWaitDuration returns the time duration waiting for available RU.
func (rd *RUDetails) RUWaitDuration() time.Duration {
	panic("excised: RUDetails.RUWaitDuration")
}

// Update updates the RU runtime stats with the given consumption info.
func (rd *RUDetails) Update(consumption *rmpb.Consumption, waitDuration time.Duration) {
	panic("excised: RUDetails.Update")
}
