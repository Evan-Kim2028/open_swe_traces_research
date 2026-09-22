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
// https://github.com/acme/sqlengine/tree/cc5e161ac06827589c4966674597c137cc9e809c/store/kvstore/error/error.go
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

package error

import (
	_ "encoding/hex"
	"fmt"
	"time"

	_ "example.internal/kvstore/v2/internal/logutil"
	_ "example.internal/kvstore/v2/util"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	// ErrBodyMissing response body is missing error
	ErrBodyMissing = errors.New("response body is missing")
	// ErrTiDBShuttingDown is returned when SQLEngine is closing and send request to kvstore fail, do not retry.
	ErrTiDBShuttingDown = errors.New("sqlengine server shutting down")
	// ErrNotExist means the related data not exist.
	ErrNotExist = errors.New("not exist")
	// ErrCannotSetNilValue is the error when sets an empty value.
	ErrCannotSetNilValue = errors.New("can not set nil value")
	// ErrInvalidTxn is the error when commits or rollbacks in an invalid transaction.
	ErrInvalidTxn = errors.New("invalid transaction")
	// ErrTiKVServerTimeout is the error when kvstore server is timeout.
	ErrTiKVServerTimeout = errors.New("kvstore server timeout")
	// ErrTiFlashServerTimeout is the error when tiflash server is timeout.
	ErrTiFlashServerTimeout = errors.New("tiflash server timeout")
	// ErrQueryInterrupted is the error when the query is interrupted.
	// This is deprecated. Keep it only to pass CI :-(. We can remove this later.
	ErrQueryInterrupted = errors.New("query interrupted")
	// ErrTiKVStaleCommand is the error that the command is stale in kvstore.
	ErrTiKVStaleCommand = errors.New("kvstore stale command")
	// ErrTiKVMaxTimestampNotSynced is the error that kvstore's max timestamp is not synced.
	ErrTiKVMaxTimestampNotSynced = errors.New("kvstore max timestamp not synced")
	// ErrLockAcquireFailAndNoWaitSet is the error that acquire the lock failed while no wait is setted.
	ErrLockAcquireFailAndNoWaitSet = errors.New("lock acquired failed and no wait is set")
	// ErrResolveLockTimeout is the error that resolve lock timeout.
	ErrResolveLockTimeout = errors.New("resolve lock timeout")
	// ErrLockWaitTimeout is the error that wait for the lock is timeout.
	ErrLockWaitTimeout = errors.New("lock wait timeout")
	// ErrTiKVServerBusy is the error when kvstore server is busy.
	ErrTiKVServerBusy = errors.New("kvstore server busy")
	// ErrTiFlashServerBusy is the error that tiflash server is busy.
	ErrTiFlashServerBusy = errors.New("tiflash server busy")
	// ErrRegionUnavailable is the error when region is not available.
	ErrRegionUnavailable = errors.New("region unavailable")
	// ErrRegionDataNotReady is the error when region's data is not ready when querying it with safe_ts
	ErrRegionDataNotReady = errors.New("region data not ready")
	// ErrRegionNotInitialized is error when region is not initialized
	ErrRegionNotInitialized = errors.New("region not Initialized")
	// ErrTiKVDiskFull is the error when kvstore server disk usage is full.
	ErrTiKVDiskFull = errors.New("kvstore disk full")
	// ErrRegionRecoveryInProgress is the error when region is recovering.
	ErrRegionRecoveryInProgress = errors.New("region is being online unsafe recovered")
	// ErrRegionFlashbackInProgress is the error when a region in the flashback progress receive any other request.
	ErrRegionFlashbackInProgress = errors.New("region is in the flashback progress")
	// ErrRegionFlashbackNotPrepared is the error when a region is not prepared for the flashback first.
	ErrRegionFlashbackNotPrepared = errors.New("region is not prepared for the flashback")
	// ErrIsWitness is the error when a request is send to a witness.
	ErrIsWitness = errors.New("peer is witness")
	// ErrUnknown is the unknow error.
	ErrUnknown = errors.New("unknown")
	// ErrResultUndetermined is the error when execution result is unknown.
	ErrResultUndetermined = errors.New("execution result undetermined")
)

type ErrQueryInterruptedWithSignal struct {
	Signal uint32
}

func (e ErrQueryInterruptedWithSignal) Error() string {
	panic("excised: ErrQueryInterruptedWithSignal.Error")
}

// MismatchClusterID represents the message that the cluster ID of the Meta client does not match the Meta.
const MismatchClusterID = "mismatch cluster id"

// IsErrNotFound checks if err is a kind of NotFound error.
func IsErrNotFound(err error) bool {
	return errors.Is(err, ErrNotExist)
}

// ErrDeadlock wraps *kvrpcpb.Deadlock to implement the error interface.
// It also marks if the deadlock is retryable.
type ErrDeadlock struct {
	*kvrpcpb.Deadlock
	IsRetryable bool
}

func (d *ErrDeadlock) Error() string {
	panic("excised: ErrDeadlock.Error")
}

// PDError wraps *pdpb.Error to implement the error interface.
type PDError struct {
	Err *pdpb.Error
}

func (d *PDError) Error() string {
	panic("excised: PDError.Error")
}

// ErrKeyExist wraps *pdpb.AlreadyExist to implement the error interface.
type ErrKeyExist struct {
	*kvrpcpb.AlreadyExist
}

func (k *ErrKeyExist) Error() string {
	panic("excised: ErrKeyExist.Error")
}

// IsErrKeyExist returns true if it is ErrKeyExist.
func IsErrKeyExist(err error) bool {
	panic("excised: IsErrKeyExist")
}

// ErrWriteConflict wraps *kvrpcpb.ErrWriteConflict to implement the error interface.
type ErrWriteConflict struct {
	*kvrpcpb.WriteConflict
}

func (k *ErrWriteConflict) Error() string {
	return fmt.Sprintf("write conflict { %s }", k.WriteConflict.String())
}

// IsErrWriteConflict returns true if it is ErrWriteConflict.
func IsErrWriteConflict(err error) bool {
	var e *ErrWriteConflict
	return errors.As(err, &e)
}

// NewErrWriteConflictWithArgs generates an ErrWriteConflict with args.
func NewErrWriteConflictWithArgs(startTs, conflictTs, conflictCommitTs uint64, key []byte, reason kvrpcpb.WriteConflict_Reason) *ErrWriteConflict {
	conflict := kvrpcpb.WriteConflict{
		StartTs:          startTs,
		ConflictTs:       conflictTs,
		Key:              key,
		ConflictCommitTs: conflictCommitTs,
		Reason:           reason,
	}
	return &ErrWriteConflict{WriteConflict: &conflict}
}

// ErrWriteConflictInLatch is the error when the commit meets an write conflict error when local latch is enabled.
type ErrWriteConflictInLatch struct {
	StartTS uint64
}

func (e *ErrWriteConflictInLatch) Error() string {
	panic("excised: ErrWriteConflictInLatch.Error")
}

// ErrRetryable wraps *kvrpcpb.Retryable to implement the error interface.
type ErrRetryable struct {
	Retryable string
}

func (k *ErrRetryable) Error() string {
	return k.Retryable
}

// ErrTxnTooLarge is the error when transaction is too large, lock time reached the maximum value.
type ErrTxnTooLarge struct {
	Size int
}

func (e *ErrTxnTooLarge) Error() string {
	return fmt.Sprintf("txn too large, size: %v.", e.Size)
}

// ErrEntryTooLarge is the error when a key value entry is too large.
type ErrEntryTooLarge struct {
	Limit uint64
	Size  uint64
}

func (e *ErrEntryTooLarge) Error() string {
	panic("excised: ErrEntryTooLarge.Error")
}

// ErrPDServerTimeout is the error when meta server is timeout.
type ErrPDServerTimeout struct {
	msg string
}

// NewErrPDServerTimeout creates an ErrPDServerTimeout.
func NewErrPDServerTimeout(msg string) error {
	panic("excised: NewErrPDServerTimeout")
}

func (e *ErrPDServerTimeout) Error() string {
	panic("excised: ErrPDServerTimeout.Error")
}

// ErrGCTooEarly is the error that GC life time is shorter than transaction duration
type ErrGCTooEarly struct {
	TxnStartTS  time.Time
	GCSafePoint time.Time
}

func (e *ErrGCTooEarly) Error() string {
	panic("excised: ErrGCTooEarly.Error")
}

// ErrTokenLimit is the error that token is up to the limit.
type ErrTokenLimit struct {
	StoreID uint64
}

func (e *ErrTokenLimit) Error() string {
	panic("excised: ErrTokenLimit.Error")
}

// ErrAssertionFailed is the error that assertion on data failed.
type ErrAssertionFailed struct {
	*kvrpcpb.AssertionFailed
}

// ErrLockOnlyIfExistsNoReturnValue is used when the flag `LockOnlyIfExists` of `LockCtx` is set, but `ReturnValues` is not.
type ErrLockOnlyIfExistsNoReturnValue struct {
	StartTS     uint64
	ForUpdateTs uint64
	LockKey     []byte
}

// ErrLockOnlyIfExistsNoPrimaryKey is used when the flag `LockOnlyIfExists` of `LockCtx` is set, but primary key of current transaction is not.
type ErrLockOnlyIfExistsNoPrimaryKey struct {
	StartTS     uint64
	ForUpdateTs uint64
	LockKey     []byte
}

func (e *ErrAssertionFailed) Error() string {
	panic("excised: ErrAssertionFailed.Error")
}

func (e *ErrLockOnlyIfExistsNoReturnValue) Error() string {
	panic("excised: ErrLockOnlyIfExistsNoReturnValue.Error")
}

func (e *ErrLockOnlyIfExistsNoPrimaryKey) Error() string {
	panic("excised: ErrLockOnlyIfExistsNoPrimaryKey.Error")
}

// ExtractKeyErr extracts a KeyError.
func ExtractKeyErr(keyErr *kvrpcpb.KeyError) error {
	if keyErr.Conflict != nil {
		return errors.WithStack(&ErrWriteConflict{WriteConflict: keyErr.GetConflict()})
	}
	if keyErr.Retryable != "" {
		return errors.WithStack(&ErrRetryable{Retryable: keyErr.Retryable})
	}
	return errors.Errorf("unexpected KeyError: %s", keyErr.String())
}

// IsErrorUndetermined checks if the error is undetermined error.
func IsErrorUndetermined(err error) bool {
	panic("excised: IsErrorUndetermined")
}

// Log logs the error if it is not nil.
func Log(err error) {
	if err != nil {
		log.Error("encountered error", zap.Error(err), zap.Stack("stack"))
	}
}
