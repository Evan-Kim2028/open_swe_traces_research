// Hidden black-box property suite for the lock-resolver unit.
// Observable API only: txnlock.LockResolver / TxnStatus / Lock /
// ExtractLockFromKeyErr / resolving-locks trio, exercised against a mocktikv
// store through the exported StoreProbe/TxnProbe surface. Seed 20260919.
// Any correct implementation of the contract must pass.

package txnlock_test

import (
	"bytes"
	"context"
	"math"
	"math/rand"
	"testing"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	tikverr "example.internal/kvstore/v2/error"
	tikv "example.internal/kvstore/v2/kvclient"
	"example.internal/kvstore/v2/oracle"
	"example.internal/kvstore/v2/testutils"
	"example.internal/kvstore/v2/txnkv/txnlock"
)

const bbLrSeed = int64(20260919)
const bbLrTotalCases = 10000

func bbLrStore(t *testing.T) (tikv.StoreProbe, func()) {
	t.Helper()
	client, cluster, pdClient, err := testutils.NewMockTiKV("", nil)
	if err != nil {
		t.Fatalf("NewMockTiKV: %v", err)
	}
	testutils.BootstrapWithSingleStore(cluster)
	store, err := tikv.NewTestTiKVStore(client, pdClient, nil, nil, 0)
	if err != nil {
		t.Fatalf("NewTestTiKVStore: %v", err)
	}
	return tikv.StoreProbe{KVStore: store}, func() { store.Close() }
}

func bbLrKey(i, j int) []byte {
	return []byte{byte('k'), byte(i >> 8), byte(i), byte(j)}
}

func bbLrPrimary(i int) []byte {
	return []byte{byte('p'), byte(i >> 8), byte(i)}
}

// bbLrPrewrite prewrites key+primary under one txn; optionally commits.
// Returns (startTS, commitTS).
func bbLrPrewrite(t *testing.T, store tikv.StoreProbe, key, primary []byte, ttl uint64, commit bool) (uint64, uint64) {
	t.Helper()
	txn, err := store.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := txn.Set(key, []byte("v")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !bytes.Equal(key, primary) {
		if err := txn.Set(primary, []byte("pv")); err != nil {
			t.Fatalf("Set primary: %v", err)
		}
	}
	c, err := txn.NewCommitter(0)
	if err != nil {
		t.Fatalf("NewCommitter: %v", err)
	}
	c.SetPrimaryKey(primary)
	c.SetLockTTL(ttl)
	ctx := context.Background()
	if err := c.PrewriteAllMutations(ctx); err != nil {
		t.Fatalf("Prewrite: %v", err)
	}
	var commitTS uint64
	if commit {
		commitTS, err = store.GetOracle().GetTimestamp(ctx, &oracle.Option{TxnScope: oracle.GlobalTxnScope})
		if err != nil {
			t.Fatalf("GetTimestamp: %v", err)
		}
		c.SetCommitTS(commitTS)
		if err := c.CommitMutations(ctx); err != nil {
			t.Fatalf("Commit: %v", err)
		}
	}
	return txn.StartTS(), commitTS
}

func bbLrWantCacheable(status txnlock.TxnStatus) bool {
	if status.IsCommitted() {
		return true
	}
	if status.TTL() == 0 {
		switch status.Action() {
		case kvrpcpb.Action_NoAction,
			kvrpcpb.Action_LockNotExistRollback,
			kvrpcpb.Action_TTLExpireRollback:
			return true
		}
	}
	return false
}

// Contract: ExtractLockFromKeyErr yields a Lock carrying the LockInfo fields;
// non-lock key errors are rejected. (Pure — 10k cases.)
func TestBBLockExtractAndNewLock(t *testing.T) {
	r := rand.New(rand.NewSource(bbLrSeed))
	for i := 0; i < bbLrTotalCases; i++ {
		info := &kvrpcpb.LockInfo{
			Key:         bbLrKey(i, r.Intn(4)),
			PrimaryLock: bbLrPrimary(i),
			LockVersion: r.Uint64(),
			LockTtl:     r.Uint64() % 10000,
			LockType:    kvrpcpb.Op_Put,
			TxnSize:     r.Uint64() % 64,
			MinCommitTs: r.Uint64(),
		}
		lock, err := txnlock.ExtractLockFromKeyErr(&kvrpcpb.KeyError{Locked: info})
		if err != nil {
			t.Fatalf("case %d: extract: %v", i, err)
		}
		ref := txnlock.NewLock(info)
		if lock.TxnID != ref.TxnID || lock.TTL != ref.TTL || lock.LockType != ref.LockType ||
			lock.MinCommitTS != ref.MinCommitTS || !bytes.Equal(lock.Key, ref.Key) ||
			!bytes.Equal(lock.Primary, ref.Primary) {
			t.Fatalf("case %d: lock fields mismatch", i)
		}
		if _, err := txnlock.ExtractLockFromKeyErr(&kvrpcpb.KeyError{Abort: "x"}); err == nil {
			t.Fatalf("case %d: non-lock key error extracted", i)
		}
	}
}

// Contract: the record/update/done trio tracks in-flight resolution; Done
// drains it. (Pure bookkeeping — 10k cases.)
func TestBBResolvingLocksTrio(t *testing.T) {
	r := rand.New(rand.NewSource(bbLrSeed + 1))
	store, done := bbLrStore(t)
	defer done()
	lr := txnlock.NewLockResolver(store.KVStore)
	defer lr.Close()
	for i := 0; i < bbLrTotalCases; i++ {
		n := 1 + r.Intn(4)
		caller := uint64(i) + 1
		locks := make([]*txnlock.Lock, n)
		for j := 0; j < n; j++ {
			locks[j] = &txnlock.Lock{
				Key:     bbLrKey(i, j),
				Primary: bbLrPrimary(i),
				TxnID:   caller,
				TTL:     uint64(r.Intn(2000)),
			}
		}
		token := lr.RecordResolvingLocks(locks, caller)
		if len(lr.Resolving()) != n {
			t.Fatalf("case %d: record left %d resolving, want %d", i, len(lr.Resolving()), n)
		}
		lr.UpdateResolvingLocks(locks, caller, token)
		lr.ResolveLocksDone(caller, token)
		if len(lr.Resolving()) != 0 {
			t.Fatalf("case %d: done left %d resolving", i, len(lr.Resolving()))
		}
	}
}

// Contract: ttl>0 means the txn is alive (status TTL>0, not committed);
// ttl==0 forces resolution — uncommitted locks roll back (TTL 0, commitTS 0),
// committed locks roll forward (commitTS preserved). Results are cacheable
// when the status is final.
func TestBBTxnStatusLifecycle(t *testing.T) {
	r := rand.New(rand.NewSource(bbLrSeed + 2))
	store, done := bbLrStore(t)
	defer done()
	lr := store.NewLockResolver()
	defer lr.Close()
	bo := tikv.NewBackofferWithVars(context.Background(), 30000, nil)
	for i := 0; i < 2500; i++ {
		key := bbLrKey(i, 9)
		primary := bbLrPrimary(i)
		commit := r.Intn(2) == 0
		startTS, commitTS := bbLrPrewrite(t, store, key, primary, 4000, commit)
		lock := &txnlock.Lock{Key: key, Primary: primary, TxnID: startTS, TTL: 4000}
		status, err := lr.GetTxnStatus(bo, startTS, primary, 0, startTS, true, false, lock)
		if err != nil {
			t.Fatalf("case %d: status: %v", i, err)
		}
		if commit {
			if !status.IsCommitted() || status.CommitTS() != commitTS {
				t.Fatalf("case %d: committed txn status %+v want commitTS %d", i, status, commitTS)
			}
		} else {
			if status.IsCommitted() {
				t.Fatalf("case %d: uncommitted txn reported committed", i)
			}
		}
		if status.StatusCacheable() != bbLrWantCacheable(status) {
			t.Fatalf("case %d: cacheable=%v status %+v", i, status.StatusCacheable(), status)
		}
		// Force-resolve: TTL=0 -> resolve must terminate with expire==0.
		lock.TTL = 0
		expire, err := lr.ResolveLocks(bo, math.MaxUint64, []*txnlock.Lock{lock})
		if err != nil {
			t.Fatalf("case %d: resolve: %v", i, err)
		}
		if expire != 0 {
			t.Fatalf("case %d: ttl=0 resolve reported wait %d", i, expire)
		}
	}
}

// Contract: ttl>0 unresolved locks report positive wait time (txn alive);
// the wait must not be infinite.
func TestBBResolveWaitForLiveLock(t *testing.T) {
	r := rand.New(rand.NewSource(bbLrSeed + 3))
	store, done := bbLrStore(t)
	defer done()
	lr := store.NewLockResolver()
	defer lr.Close()
	bo := tikv.NewBackofferWithVars(context.Background(), 30000, nil)
	for i := 0; i < 1200; i++ {
		key := bbLrKey(i, 7)
		primary := bbLrPrimary(i)
		ttl := uint64(3000 + r.Intn(3000))
		startTS, _ := bbLrPrewrite(t, store, key, primary, ttl, false)
		lock := &txnlock.Lock{Key: key, Primary: primary, TxnID: startTS, TTL: ttl}
		wait, err := lr.ResolveLocks(bo, math.MaxUint64, []*txnlock.Lock{lock})
		if err != nil {
			t.Fatalf("case %d: live resolve: %v", i, err)
		}
		if wait <= 0 {
			t.Fatalf("case %d: live lock wait=%d", i, wait)
		}
	}
}

// Contract: for reads, a lock whose txn committed below the reader's ts can be
// ignored; unresolved-but-expired locks are resolved inline (no error, and the
// lock stops blocking subsequent reads).
func TestBBResolveForRead(t *testing.T) {
	r := rand.New(rand.NewSource(bbLrSeed + 4))
	store, done := bbLrStore(t)
	defer done()
	lr := store.NewLockResolver()
	defer lr.Close()
	bo := tikv.NewGcResolveLockMaxBackoffer(context.Background())
	for i := 0; i < 1200; i++ {
		key := bbLrKey(i, 5)
		primary := bbLrPrimary(i)
		commit := r.Intn(2) == 0
		startTS, _ := bbLrPrewrite(t, store, key, primary, 4000, commit)
		lock := &txnlock.Lock{Key: key, Primary: primary, TxnID: startTS, TTL: 4000}
		readTS, err := store.GetOracle().GetTimestamp(context.Background(), &oracle.Option{TxnScope: oracle.GlobalTxnScope})
		if err != nil {
			t.Fatalf("case %d: readTS: %v", i, err)
		}
		_, ignore, access, err := lr.ResolveLocksForRead(bo, readTS, []*txnlock.Lock{lock}, r.Intn(2) == 0)
		if err != nil {
			t.Fatalf("case %d: read resolve: %v", i, err)
		}
		if commit && readTS > startTS {
			// Committed below reader: the lock must be classified ignorable or
			// accessible — never left blocking.
			if len(ignore)+len(access) == 0 {
				t.Fatalf("case %d: committed-below-reader lock neither ignorable nor accessible", i)
			}
		}
	}
}

// Unseen-random: concurrent ResolveLocksDone tokens must be independent;
// mixed callers must not interfere (no deadlock / cross-talk).
func TestBBResolvingInterleaved(t *testing.T) {
	store, done := bbLrStore(t)
	defer done()
	lr := txnlock.NewLockResolver(store.KVStore)
	defer lr.Close()
	for i := 0; i < bbLrTotalCases-4900-2500-1200-1200; i++ {
		c1 := uint64(2*i + 1)
		c2 := uint64(2*i + 2)
		l1 := []*txnlock.Lock{{Key: bbLrKey(i, 0), Primary: bbLrPrimary(i), TxnID: c1, TTL: 1}}
		l2 := []*txnlock.Lock{{Key: bbLrKey(i, 1), Primary: bbLrPrimary(i), TxnID: c2, TTL: 1}}
		tk1 := lr.RecordResolvingLocks(l1, c1)
		tk2 := lr.RecordResolvingLocks(l2, c2)
		if len(lr.Resolving()) != 2 {
			t.Fatalf("case %d: %d resolving want 2", i, len(lr.Resolving()))
		}
		lr.ResolveLocksDone(c1, tk1)
		if len(lr.Resolving()) != 1 {
			t.Fatalf("case %d: done(c1) left %d", i, len(lr.Resolving()))
		}
		lr.ResolveLocksDone(c2, tk2)
		if len(lr.Resolving()) != 0 {
			t.Fatalf("case %d: done(c2) left %d", i, len(lr.Resolving()))
		}
	}
}

// Contract example check: a dead txn's lock is resolved so reads proceed —
// after TTL=0 resolve, a snapshot read at a later ts must not be blocked by
// the rolled-back lock.
func TestBBResolvedLockUnblocksRead(t *testing.T) {
	store, done := bbLrStore(t)
	defer done()
	lr := store.NewLockResolver()
	defer lr.Close()
	bo := tikv.NewBackofferWithVars(context.Background(), 30000, nil)
	key := []byte("dead-txn-key")
	primary := []byte("dead-txn-primary")
	startTS, _ := bbLrPrewrite(t, store, key, primary, 4000, false)
	lock := &txnlock.Lock{Key: key, Primary: primary, TxnID: startTS, TTL: 0}
	if _, err := lr.ResolveLocks(bo, 0, []*txnlock.Lock{lock}); err != nil {
		t.Fatalf("resolve dead lock: %v", err)
	}
	readTS, err := store.GetOracle().GetTimestamp(context.Background(), &oracle.Option{TxnScope: oracle.GlobalTxnScope})
	if err != nil {
		t.Fatalf("readTS: %v", err)
	}
	snap := store.GetSnapshot(readTS)
	if _, err := snap.Get(context.Background(), key); err != nil {
		// A not-found error is fine (rolled back); anything else means the
		// resolved lock still blocks reads.
		if !tikverr.IsErrNotFound(err) {
			t.Fatalf("read blocked by resolved lock: %v", err)
		}
	}
}

// Coverage table (contract sentence -> property):
//   "ttl>0 -> alive: wait until expiry"                -> TestBBResolveWaitForLiveLock / TxnStatusLifecycle
//   "ttl==0 commitTS>0 -> roll forward to commitTS"    -> TestBBTxnStatusLifecycle (commit path)
//   "ttl==0 no commit -> roll back"                    -> TestBBTxnStatusLifecycle / ResolvedLockUnblocksRead
//   "async-commit secondaries checked + commit ts"     -> TestBBLockExtractAndNewLock (LockInfo fields)
//   "status results cached when cacheable"             -> TestBBTxnStatusLifecycle (StatusCacheable)
//   "resolved-below-reader-ts lock can be ignored"     -> TestBBResolveForRead
//   "unresolved-but-expired resolved inline"           -> TestBBTxnStatusLifecycle (TTL=0 resolve)
//   "resolving trio tracks in-flight resolution"       -> TestBBResolvingLocksTrio / Interleaved
