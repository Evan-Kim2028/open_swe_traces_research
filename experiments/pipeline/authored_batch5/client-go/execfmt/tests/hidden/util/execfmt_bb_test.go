package util

import (
	"sync"
	"testing"
	"time"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	rmpb "github.com/pingcap/kvproto/pkg/resource_manager"
)

// Hidden suite for unit execfmt. One TestDetailNN per DETAILS.md line.

// TestDetail01: FormatDuration prunes precision per the documented rule.
func TestDetail01(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{999 * time.Nanosecond, "999ns"},
		{time.Microsecond, "1µs"},
		{9412345 * time.Nanosecond, "9.41ms"},
		{10412345 * time.Nanosecond, "10.4ms"},
		{100450 * time.Nanosecond, "100.5µs"},
		{5999 * time.Millisecond, "6s"},
	}
	for _, c := range cases {
		if got := FormatDuration(c.d); got != c.want {
			t.Fatalf("FormatDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

// TestDetail02: MergeFromTimeDetail prefers V2; the V1 path uses its own
// fields and never sets SuspendTime.
func TestDetail02(t *testing.T) {
	td := &TimeDetail{}
	td.MergeFromTimeDetail(&kvrpcpb.TimeDetailV2{
		ProcessWallTimeNs:        1000,
		ProcessSuspendWallTimeNs: 500,
		WaitWallTimeNs:           2000,
		KvReadWallTimeNs:         3000,
		TotalRpcWallTimeNs:       9000,
	}, &kvrpcpb.TimeDetail{
		ProcessWallTimeMs:  999,
		WaitWallTimeMs:     999,
		KvReadWallTimeMs:   999,
		TotalRpcWallTimeNs: 999,
	})
	if td.ProcessTime != time.Microsecond || td.SuspendTime != 500*time.Nanosecond ||
		td.WaitTime != 2*time.Microsecond || td.KvReadWallTime != 3*time.Microsecond ||
		td.TotalRPCWallTime != 9*time.Microsecond {
		t.Fatalf("V2 not preferred: %+v", td)
	}
	td = &TimeDetail{}
	td.MergeFromTimeDetail(nil, &kvrpcpb.TimeDetail{
		WaitWallTimeMs:     5,
		ProcessWallTimeMs:  3,
		KvReadWallTimeMs:   7,
		TotalRpcWallTimeNs: 2_000_000_000,
	})
	if td.WaitTime != 5*time.Millisecond || td.ProcessTime != 3*time.Millisecond ||
		td.KvReadWallTime != 7*time.Millisecond || td.TotalRPCWallTime != 2*time.Second {
		t.Fatalf("V1 path wrong units: %+v", td)
	}
	if td.SuspendTime != 0 {
		t.Fatalf("V1 path set SuspendTime = %v", td.SuspendTime)
	}
}

// TestDetail03: MergeFromScanDetailV2 is nil-safe, renames the version fields
// to key fields, and accumulates counters and durations.
func TestDetail03(t *testing.T) {
	sd := &ScanDetail{}
	sd.MergeFromScanDetailV2(nil) // must not panic
	pb := &kvrpcpb.ScanDetailV2{
		TotalVersions:             10,
		ProcessedVersions:         4,
		ProcessedVersionsSize:     99,
		RocksdbDeleteSkippedCount: 1,
		RocksdbKeySkippedCount:    2,
		RocksdbBlockCacheHitCount: 3,
		RocksdbBlockReadCount:     4,
		RocksdbBlockReadByte:      5,
		RocksdbBlockReadNanos:     60,
		GetSnapshotNanos:          70,
	}
	sd.MergeFromScanDetailV2(pb)
	sd.MergeFromScanDetailV2(pb)
	if sd.TotalKeys != 20 || sd.ProcessedKeys != 8 || sd.ProcessedKeysSize != 198 {
		t.Fatalf("renamed fields: %+v", sd)
	}
	if sd.RocksdbDeleteSkippedCount != 2 || sd.RocksdbKeySkippedCount != 4 ||
		sd.RocksdbBlockCacheHitCount != 6 || sd.RocksdbBlockReadCount != 8 ||
		sd.RocksdbBlockReadByte != 10 {
		t.Fatalf("rocksdb counters: %+v", sd)
	}
	if sd.RocksdbBlockReadDuration != 120*time.Nanosecond || sd.GetSnapshotDuration != 140*time.Nanosecond {
		t.Fatalf("durations: %+v", sd)
	}
}

// TestDetail04: Merge accumulates every field and is concurrency-safe.
func TestDetail04(t *testing.T) {
	sd := &ScanDetail{}
	sd.Merge(&ScanDetail{
		TotalKeys: 1, ProcessedKeys: 2, ProcessedKeysSize: 3,
		RocksdbDeleteSkippedCount: 4, RocksdbKeySkippedCount: 5,
		RocksdbBlockCacheHitCount: 6, RocksdbBlockReadCount: 7,
		RocksdbBlockReadByte: 8, RocksdbBlockReadDuration: 9,
		GetSnapshotDuration: 10,
	})
	sd.Merge(&ScanDetail{TotalKeys: 10, ProcessedKeys: 20, ProcessedKeysSize: 30})
	if sd.TotalKeys != 11 || sd.ProcessedKeys != 22 || sd.ProcessedKeysSize != 33 ||
		sd.RocksdbDeleteSkippedCount != 4 || sd.RocksdbBlockReadDuration != 9 ||
		sd.GetSnapshotDuration != 10 {
		t.Fatalf("ScanDetail.Merge: %+v", sd)
	}
	wd := &WriteDetail{}
	wd.Merge(&WriteDetail{StoreBatchWaitDuration: 1, PersistLogDuration: 2, ApplyLogDuration: 3})
	wd.Merge(&WriteDetail{StoreBatchWaitDuration: 10, PersistLogDuration: 20, ApplyLogDuration: 30})
	if wd.StoreBatchWaitDuration != 11 || wd.PersistLogDuration != 22 || wd.ApplyLogDuration != 33 {
		t.Fatalf("WriteDetail.Merge: %+v", wd)
	}
	rd := &ResolveLockDetail{}
	rd.Merge(&ResolveLockDetail{ResolveLockTime: 5})
	rd.Merge(&ResolveLockDetail{ResolveLockTime: 7})
	if rd.ResolveLockTime != 12 {
		t.Fatalf("ResolveLockDetail.Merge: %d", rd.ResolveLockTime)
	}
	// Concurrent merges must not lose updates.
	c := &ScanDetail{}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Merge(&ScanDetail{ProcessedKeys: 1})
		}()
	}
	wg.Wait()
	if c.ProcessedKeys != 50 {
		t.Fatalf("concurrent Merge lost updates: %d", c.ProcessedKeys)
	}
}

// TestDetail05: RUDetails accumulates consumption; nil receiver or nil
// consumption is a no-op; Clone/Merge snapshot and sum.
func TestDetail05(t *testing.T) {
	rd := NewRUDetails()
	if rd.RRU() != 0 || rd.WRU() != 0 || rd.RUWaitDuration() != 0 {
		t.Fatalf("fresh RUDetails not zero")
	}
	rd.Update(&rmpb.Consumption{RRU: 1.5, WRU: 2.5}, 3*time.Nanosecond)
	rd.Update(&rmpb.Consumption{RRU: 1.5, WRU: 2.5}, 3*time.Nanosecond)
	if rd.RRU() != 3 || rd.WRU() != 5 || rd.RUWaitDuration() != 6*time.Nanosecond {
		t.Fatalf("Update did not accumulate: %v %v %v", rd.RRU(), rd.WRU(), rd.RUWaitDuration())
	}
	rd.Update(nil, time.Second) // must not panic or change
	if rd.RUWaitDuration() != 6*time.Nanosecond {
		t.Fatalf("nil consumption changed state")
	}
	var nilRD *RUDetails
	nilRD.Update(&rmpb.Consumption{RRU: 1}, time.Second) // must not panic
	w := NewRUDetailsWith(4, 6, 8*time.Nanosecond)
	if w.RRU() != 4 || w.WRU() != 6 || w.RUWaitDuration() != 8*time.Nanosecond {
		t.Fatalf("NewRUDetailsWith: %v %v %v", w.RRU(), w.WRU(), w.RUWaitDuration())
	}
	cl := w.Clone()
	if cl.RRU() != 4 || cl.WRU() != 6 || cl.RUWaitDuration() != 8*time.Nanosecond {
		t.Fatalf("Clone: %v %v %v", cl.RRU(), cl.WRU(), cl.RUWaitDuration())
	}
	m := NewRUDetailsWith(1, 1, time.Nanosecond)
	m.Merge(w)
	if m.RRU() != 5 || m.WRU() != 7 || m.RUWaitDuration() != 9*time.Nanosecond {
		t.Fatalf("Merge: %v %v %v", m.RRU(), m.WRU(), m.RUWaitDuration())
	}
}

// TestDetail06: MergeFromWriteDetailPb maps every *Nanos field to a Duration;
// nil pb is a no-op.
func TestDetail06(t *testing.T) {
	wd := &WriteDetail{}
	wd.MergeFromWriteDetailPb(nil) // must not panic
	wd.MergeFromWriteDetailPb(&kvrpcpb.WriteDetail{
		StoreBatchWaitNanos:        1,
		ProposeSendWaitNanos:       2,
		PersistLogNanos:            3,
		RaftDbWriteLeaderWaitNanos: 4,
		RaftDbSyncLogNanos:         5,
		RaftDbWriteMemtableNanos:   6,
		CommitLogNanos:             7,
		ApplyBatchWaitNanos:        8,
		ApplyLogNanos:              9,
		ApplyMutexLockNanos:        10,
		ApplyWriteLeaderWaitNanos:  11,
		ApplyWriteWalNanos:         12,
		ApplyWriteMemtableNanos:    13,
	})
	want := []struct {
		got  time.Duration
		want int64
	}{
		{wd.StoreBatchWaitDuration, 1}, {wd.ProposeSendWaitDuration, 2},
		{wd.PersistLogDuration, 3}, {wd.RaftDbWriteLeaderWaitDuration, 4},
		{wd.RaftDbSyncLogDuration, 5}, {wd.RaftDbWriteMemtableDuration, 6},
		{wd.CommitLogDuration, 7}, {wd.ApplyBatchWaitDuration, 8},
		{wd.ApplyLogDuration, 9}, {wd.ApplyMutexLockDuration, 10},
		{wd.ApplyWriteLeaderWaitDuration, 11}, {wd.ApplyWriteWalDuration, 12},
		{wd.ApplyWriteMemtableDuration, 13},
	}
	for i, c := range want {
		if c.got != time.Duration(c.want) {
			t.Fatalf("field %d = %v, want %dns", i, c.got, c.want)
		}
	}
}
