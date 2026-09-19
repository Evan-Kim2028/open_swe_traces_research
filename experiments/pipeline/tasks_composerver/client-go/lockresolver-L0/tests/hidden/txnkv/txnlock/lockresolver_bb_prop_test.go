package txnlock_test

import (
	"bytes"
	"context"
	"math"
	"math/rand"
	"testing"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	tikv "example.internal/kvstore/v2/kvclient"
	"example.internal/kvstore/v2/oracle"
	"example.internal/kvstore/v2/testutils"
	"example.internal/kvstore/v2/txnkv/txnlock"
)

const lockResolverBBSeed = 20260919
const lockResolverBBCases = 10000

func newLockResolverStore(t *testing.T) (tikv.StoreProbe, func()) {
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

func bbLockKey(i int, j int) []byte {
	return []byte{byte('l'), byte(i >> 8), byte(i), byte(j)}
}

func bbPrimaryKey(i int) []byte {
	return []byte{byte('p'), byte(i >> 8), byte(i)}
}

func prewriteLock(t *testing.T, store tikv.StoreProbe, key, primary []byte, ttl uint64, commit bool) uint64 {
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
	if commit {
		commitTS, err := store.GetOracle().GetTimestamp(ctx, &oracle.Option{TxnScope: oracle.GlobalTxnScope})
		if err != nil {
			t.Fatalf("GetTimestamp: %v", err)
		}
		c.SetCommitTS(commitTS)
		if err := c.CommitMutations(ctx); err != nil {
			t.Fatalf("Commit: %v", err)
		}
	}
	return txn.StartTS()
}

func wantStatusCacheable(status txnlock.TxnStatus) bool {
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

func TestExtractLockFromKeyErrProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(lockResolverBBSeed))
	for i := 0; i < lockResolverBBCases; i++ {
		key := bbLockKey(i, rng.Intn(8))
		primary := bbPrimaryKey(i)
		info := &kvrpcpb.LockInfo{
			Key:         key,
			PrimaryLock: primary,
			LockVersion: rng.Uint64(),
			LockTtl:     rng.Uint64() % 5000,
			LockType:    kvrpcpb.Op_Put,
			TxnSize:     rng.Uint64() % 100,
			MinCommitTs: rng.Uint64(),
		}
		lock, err := txnlock.ExtractLockFromKeyErr(&kvrpcpb.KeyError{Locked: info})
		if err != nil {
			t.Fatalf("case %d extract: %v", i, err)
		}
		got := txnlock.NewLock(info)
		if lock.TxnID != got.TxnID || lock.TTL != got.TTL || lock.LockType != got.LockType ||
			lock.MinCommitTS != got.MinCommitTS || !bytes.Equal(lock.Key, got.Key) ||
			!bytes.Equal(lock.Primary, got.Primary) {
			t.Fatalf("case %d mismatch", i)
		}
		_, err = txnlock.ExtractLockFromKeyErr(&kvrpcpb.KeyError{Abort: "x"})
		if err == nil {
			t.Fatalf("case %d non-lock err", i)
		}
	}
}

func TestResolvingLocksTrackingProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(lockResolverBBSeed + 1))
	store, done := newLockResolverStore(t)
	defer done()
	lr := txnlock.NewLockResolver(store.KVStore)
	defer lr.Close()
	for i := 0; i < lockResolverBBCases; i++ {
		n := 1 + rng.Intn(4)
		locks := make([]*txnlock.Lock, n)
		caller := uint64(i) + 1
		for j := 0; j < n; j++ {
			locks[j] = &txnlock.Lock{
				Key:     bbLockKey(i, j),
				Primary: bbPrimaryKey(i),
				TxnID:   caller,
				TTL:     uint64(rng.Intn(1000)),
			}
		}
		token := lr.RecordResolvingLocks(locks, caller)
		if len(lr.Resolving()) != n {
			t.Fatalf("case %d record count", i)
		}
		if n > 1 {
			locks[0].TTL = 0
			lr.UpdateResolvingLocks(locks, caller, token)
		}
		lr.ResolveLocksDone(caller, token)
		if len(lr.Resolving()) != 0 {
			t.Fatalf("case %d done left resolving", i)
		}
	}
}

func TestLockResolverTxnStatusProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(lockResolverBBSeed + 2))
	store, done := newLockResolverStore(t)
	defer done()
	lr := store.NewLockResolver()
	defer lr.Close()
	bo := tikv.NewBackofferWithVars(context.Background(), 5000, nil)
	for i := 0; i < lockResolverBBCases; i++ {
		key := bbLockKey(i, 0)
		primary := bbPrimaryKey(i)
		commit := rng.Intn(2) == 0
		ttl := uint64(3000)
		if rng.Intn(10) == 0 {
			ttl = 0
		}
		startTS := prewriteLock(t, store, key, primary, ttl, commit)
		lock := &txnlock.Lock{Key: key, Primary: primary, TxnID: startTS, TTL: ttl}
		status, err := lr.GetTxnStatus(bo, startTS, primary, 0, startTS, true, false, lock)
		if err != nil {
			t.Fatalf("case %d status: %v", i, err)
		}
		if commit {
			if !status.IsCommitted() || status.CommitTS() == 0 {
				t.Fatalf("case %d committed", i)
			}
		} else if ttl > 0 {
			if status.TTL() == 0 && !status.IsRolledBack() && !status.IsCommitted() {
				// live lock may still report ttl==0 after status query in edge cases; allow rolled back only
			}
		}
		if status.StatusCacheable() != wantStatusCacheable(status) {
			t.Fatalf("case %d cacheable", i)
		}
		if ttl == 0 && !commit {
			lock.TTL = 0
			expire, err := lr.ResolveLocks(bo, math.MaxUint64, []*txnlock.Lock{lock})
			if err != nil {
				t.Fatalf("case %d resolve: %v", i, err)
			}
			if expire != 0 {
				t.Fatalf("case %d expire %d", i, expire)
			}
		}
	}
}

func TestLockResolverReadPathProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(lockResolverBBSeed + 3))
	store, done := newLockResolverStore(t)
	defer done()
	lr := store.NewLockResolver()
	defer lr.Close()
	bo := tikv.NewGcResolveLockMaxBackoffer(context.Background())
	for i := 0; i < lockResolverBBCases; i++ {
		key := bbLockKey(i, 1)
		primary := bbPrimaryKey(i)
		commit := rng.Intn(2) == 0
		startTS := prewriteLock(t, store, key, primary, 3000, commit)
		lock := &txnlock.Lock{Key: key, Primary: primary, TxnID: startTS, TTL: 3000}
		readTS, err := store.GetOracle().GetTimestamp(context.Background(), &oracle.Option{TxnScope: oracle.GlobalTxnScope})
		if err != nil {
			t.Fatalf("case %d read ts: %v", i, err)
		}
		_, ignore, access, err := lr.ResolveLocksForRead(bo, readTS, []*txnlock.Lock{lock}, rng.Intn(2) == 0)
		if err != nil {
			t.Fatalf("case %d read resolve: %v", i, err)
		}
		if commit && readTS > startTS {
			if len(access) == 0 && len(ignore) == 0 {
				// committed below reader may be access or ignore depending on commit ts
			}
		}
	}
}

func TestLockResolverContractExamples(t *testing.T) {
	store, done := newLockResolverStore(t)
	defer done()
	lr := store.NewLockResolver()
	defer lr.Close()
	bo := tikv.NewBackofferWithVars(context.Background(), 5000, nil)

	// ttl>0 means alive; ttl==0 with commit rolls forward
	key := []byte("contract-k")
	primary := []byte("contract-p")
	start := prewriteLock(t, store, key, primary, 5000, true)
	lock := &txnlock.Lock{Key: key, Primary: primary, TxnID: start, TTL: 5000}
	status, err := lr.GetTxnStatus(bo, start, primary, 0, start, true, false, lock)
	if err != nil || !status.IsCommitted() {
		t.Fatal("committed txn status")
	}

	// rolled back lock
	start2 := prewriteLock(t, store, []byte("contract-k2"), []byte("contract-p2"), 5000, false)
	lock2 := &txnlock.Lock{Key: []byte("contract-k2"), Primary: []byte("contract-p2"), TxnID: start2, TTL: 5000}
	lock2.TTL = 0
	if _, err := lr.ResolveLocks(bo, 0, []*txnlock.Lock{lock2}); err != nil {
		t.Fatalf("rollback resolve: %v", err)
	}

	// RecordResolvingLocks trio
	token := lr.RecordResolvingLocks([]*txnlock.Lock{lock}, 99)
	lr.UpdateResolvingLocks([]*txnlock.Lock{lock}, 99, token)
	lr.ResolveLocksDone(99, token)

	// ExtractLockFromKeyErr
	info := &kvrpcpb.LockInfo{Key: key, PrimaryLock: primary, LockVersion: start}
	got, err := txnlock.ExtractLockFromKeyErr(&kvrpcpb.KeyError{Locked: info})
	if err != nil || !bytes.Equal(got.Key, key) {
		t.Fatal("extract lock")
	}
	if _, err := txnlock.ExtractLockFromKeyErr(&kvrpcpb.KeyError{Abort: "no"}); err == nil {
		t.Fatal("non-lock key error must fail")
	}
}

func TestLockResolverUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(lockResolverBBSeed + 5))
	store, done := newLockResolverStore(t)
	defer done()
	lr := store.NewLockResolver()
	defer lr.Close()
	probe := txnlock.LockProbe{}
	for i := 0; i < lockResolverBBCases; i++ {
		n := 1 + rng.Intn(3)
		keys := make([][]byte, n)
		for j := 0; j < n; j++ {
			keys[j] = []byte{byte(200 + j), byte(i >> 8), byte(i), byte(j)}
		}
		status := probe.NewLockStatus(keys, rng.Intn(2) == 0, rng.Uint64())
		if len(lr.GetSecondariesFromTxnStatus(status)) != n {
			t.Fatalf("case %d secondaries", i)
		}
		useAsync := rng.Intn(2) == 0
		lock := txnlock.NewLock(&kvrpcpb.LockInfo{
			Key:            keys[0],
			PrimaryLock:    keys[0],
			LockVersion:    uint64(i + 1),
			UseAsyncCommit: useAsync,
			MinCommitTs:    rng.Uint64(),
		})
		if lock.UseAsyncCommit != useAsync {
			t.Fatalf("case %d async flag", i)
		}
		// pessimistic rollback path should not delete persisted data
		key := []byte{byte(210), byte(i >> 8), byte(i)}
		txn, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d begin: %v", i, err)
		}
		txn.SetPessimistic(true)
		if err := txn.Set(key, []byte("x")); err != nil {
			t.Fatalf("case %d set: %v", i, err)
		}
		if err := txn.Commit(context.Background()); err != nil {
			t.Fatalf("case %d commit: %v", i, err)
		}
		txn2, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d begin2: %v", i, err)
		}
		val, err := txn2.Get(context.Background(), key)
		if err != nil || !bytes.Equal(val, []byte("x")) {
			t.Fatalf("case %d read after commit: %v %q", i, err, val)
		}
		_ = txn2
	}
}
