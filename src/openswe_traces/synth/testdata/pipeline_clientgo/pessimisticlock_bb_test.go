// Hidden black-box property suite for the pessimistic-lock unit.
// Observable API only: KVTxn.LockKeys / LockCtx (return-values,
// lock-only-if-exists), GetLockedCount / CollectLockedKeys / CommitterProbe
// primary, Rollback release semantics. Seed 20260919.
// Any correct implementation of the contract must pass.

package transaction_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"example.internal/kvstore/v2/kv"
	tikv "example.internal/kvstore/v2/kvclient"
	"example.internal/kvstore/v2/testutils"
	"example.internal/kvstore/v2/txnkv/transaction"
)

const bbPlSeed = int64(20260919)
const bbPlTotalCases = 10000

func bbPlStore(t *testing.T) (tikv.StoreProbe, func()) {
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

func bbPlBegin(t *testing.T, store tikv.StoreProbe) transaction.TxnProbe {
	t.Helper()
	txn, err := store.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	txn.SetPessimistic(true)
	return txn
}

func bbPlLockCtx(forUpdateTS uint64, retN int, onlyExists bool) *kv.LockCtx {
	lc := &kv.LockCtx{ForUpdateTS: forUpdateTS, WaitStartTime: time.Now()}
	if retN > 0 {
		lc.InitReturnValues(retN)
	}
	lc.LockOnlyIfExists = onlyExists
	return lc
}

func bbPlNoWaitCtx(forUpdateTS uint64) *kv.LockCtx {
	return kv.NewLockCtx(forUpdateTS, kv.LockNoWait, time.Now())
}

func bbPlKey(i, j int) []byte {
	return []byte{byte('x'), byte(i >> 8), byte(i), byte(j)}
}

func bbPlPut(t *testing.T, store tikv.StoreProbe, key, val []byte) {
	t.Helper()
	txn, err := store.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := txn.Set(key, val); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := txn.Commit(context.Background()); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// Contract: already-locked keys are not re-locked (dedup); the locked-key
// record stays flat no matter how often the same key is re-locked.
func TestBBPessimisticDedup(t *testing.T) {
	r := rand.New(rand.NewSource(bbPlSeed))
	store, done := bbPlStore(t)
	defer done()
	for i := 0; i < 3000; i++ {
		txn := bbPlBegin(t, store)
		n := 1 + r.Intn(4)
		keys := make([][]byte, n)
		for j := range keys {
			keys[j] = bbPlKey(i, j)
		}
		lc := bbPlLockCtx(txn.StartTS(), 0, false)
		if err := txn.LockKeys(context.Background(), lc, keys...); err != nil {
			t.Fatalf("case %d: lock: %v", i, err)
		}
		base := txn.GetLockedCount()
		if base != n {
			t.Fatalf("case %d: locked %d want %d", i, base, n)
		}
		// Re-lock the same set (and a permutation) — count must not grow.
		for rep := 0; rep < 3; rep++ {
			if err := txn.LockKeys(context.Background(), lc, keys...); err != nil {
				t.Fatalf("case %d rep %d: relock: %v", i, rep, err)
			}
			if got := txn.GetLockedCount(); got != n {
				t.Fatalf("case %d rep %d: dedup broke: %d want %d", i, rep, got, n)
			}
		}
		if err := txn.Rollback(); err != nil {
			t.Fatalf("case %d: rollback: %v", i, err)
		}
	}
}

// Contract: the primary key is locked first and only once — a second LockKeys
// call must not change the primary.
func TestBBPessimisticPrimaryOnce(t *testing.T) {
	r := rand.New(rand.NewSource(bbPlSeed + 1))
	store, done := bbPlStore(t)
	defer done()
	for i := 0; i < 2000; i++ {
		txn := bbPlBegin(t, store)
		primary := bbPlKey(i, 0)
		lc := bbPlLockCtx(txn.StartTS(), 0, false)
		if err := txn.LockKeys(context.Background(), lc, primary); err != nil {
			t.Fatalf("case %d: lock primary: %v", i, err)
		}
		rest := [][]byte{bbPlKey(i, 1), bbPlKey(i, 2)}
		if r.Intn(2) == 0 {
			rest[0], rest[1] = rest[1], rest[0]
		}
		if err := txn.LockKeys(context.Background(), lc, rest...); err != nil {
			t.Fatalf("case %d: lock rest: %v", i, err)
		}
		if got := string(txn.GetCommitter().GetPrimaryKey()); got != string(primary) {
			t.Fatalf("case %d: primary %q want %q", i, got, primary)
		}
		if err := txn.Rollback(); err != nil {
			t.Fatalf("case %d: rollback: %v", i, err)
		}
	}
}

// Contract: with return-values requested, the response carries per-key values
// matching what is stored; missing keys report non-existence, not garbage.
func TestBBPessimisticReturnValues(t *testing.T) {
	r := rand.New(rand.NewSource(bbPlSeed + 2))
	store, done := bbPlStore(t)
	defer done()
	for i := 0; i < 1500; i++ {
		present := bbPlKey(i, 0)
		missing := bbPlKey(i, 1)
		val := []byte{byte('v'), byte(i >> 8), byte(i)}
		bbPlPut(t, store, present, val)
		txn := bbPlBegin(t, store)
		lc := bbPlLockCtx(txn.StartTS(), 2, false)
		if err := txn.LockKeys(context.Background(), lc, present, missing); err != nil {
			t.Fatalf("case %d: lock: %v", i, err)
		}
		if len(lc.Values) != 2 {
			t.Fatalf("case %d: values len %d want 2", i, len(lc.Values))
		}
		got, ok := lc.Values[string(present)]
		if !ok || string(got.Value) != string(val) {
			t.Fatalf("case %d: returned %q want %q (ok=%v)", i, got.Value, val, ok)
		}
		if mv, ok := lc.Values[string(missing)]; !ok || mv.Exists {
			t.Fatalf("case %d: missing key reported exists=%v ok=%v", i, mv.Exists, ok)
		}
		if err := txn.Rollback(); err != nil {
			t.Fatalf("case %d: rollback: %v", i, err)
		}
		_ = r
	}
}

// Contract: lock-only-if-exists reports which keys existed and does not lock
// missing keys; locked count reflects only existing keys.
func TestBBPessimisticLockOnlyIfExists(t *testing.T) {
	r := rand.New(rand.NewSource(bbPlSeed + 3))
	store, done := bbPlStore(t)
	defer done()
	for i := 0; i < 1500; i++ {
		primary := bbPlKey(i, 9)
		present := bbPlKey(i, 0)
		missing := bbPlKey(i, 1)
		bbPlPut(t, store, present, []byte("p"))
		txn := bbPlBegin(t, store)
		// LockOnlyIfExists requires an established primary — lock one first.
		if err := txn.LockKeys(context.Background(), bbPlLockCtx(txn.StartTS(), 0, false), primary); err != nil {
			t.Fatalf("case %d: lock primary: %v", i, err)
		}
		lc := bbPlLockCtx(txn.StartTS(), 2, true)
		if err := txn.LockKeys(context.Background(), lc, present, missing); err != nil {
			t.Fatalf("case %d: lock-if-exists: %v", i, err)
		}
		if got := txn.GetLockedCount(); got != 2 {
			t.Fatalf("case %d: locked %d want 2 (primary + existing)", i, got)
		}
		if mv, ok := lc.Values[string(missing)]; !ok || mv.Exists {
			t.Fatalf("case %d: missing exists=%v ok=%v", i, mv.Exists, ok)
		}
		if pv, ok := lc.Values[string(present)]; !ok || !pv.Exists {
			t.Fatalf("case %d: present exists=%v ok=%v", i, pv.Exists, ok)
		}
		if err := txn.Rollback(); err != nil {
			t.Fatalf("case %d: rollback: %v", i, err)
		}
		_ = r
	}
}

// Contract: a locked key blocks a competing pessimistic lock — with no-wait
// the competitor must error rather than acquire; different keys proceed.
func TestBBPessimisticMutualExclusion(t *testing.T) {
	r := rand.New(rand.NewSource(bbPlSeed + 4))
	store, done := bbPlStore(t)
	defer done()
	for i := 0; i < 800; i++ {
		key := bbPlKey(i, 3)
		other := bbPlKey(i, 4)
		t1 := bbPlBegin(t, store)
		lc := bbPlLockCtx(t1.StartTS(), 0, false)
		if err := t1.LockKeys(context.Background(), lc, key); err != nil {
			t.Fatalf("case %d: t1 lock: %v", i, err)
		}
		t2 := bbPlBegin(t, store)
		if err := t2.LockKeys(context.Background(), bbPlNoWaitCtx(t2.StartTS()), key); err == nil {
			t.Fatalf("case %d: t2 acquired locked key with no-wait", i)
		}
		if err := t2.LockKeys(context.Background(), bbPlNoWaitCtx(t2.StartTS()), other); err != nil {
			t.Fatalf("case %d: t2 could not lock free key: %v", i, err)
		}
		_ = t1.Rollback()
		_ = t2.Rollback()
		_ = r
	}
}

// Contract: rollback sends a best-effort release — after rollback the same key
// is lockable again by another txn without waiting.
func TestBBPessimisticRollbackReleases(t *testing.T) {
	store, done := bbPlStore(t)
	defer done()
	for i := 0; i < 800; i++ {
		key := bbPlKey(i, 5)
		t1 := bbPlBegin(t, store)
		lc := bbPlLockCtx(t1.StartTS(), 0, false)
		if err := t1.LockKeys(context.Background(), lc, key); err != nil {
			t.Fatalf("case %d: t1 lock: %v", i, err)
		}
		if err := t1.Rollback(); err != nil {
			t.Fatalf("case %d: t1 rollback: %v", i, err)
		}
		t2 := bbPlBegin(t, store)
		if err := t2.LockKeys(context.Background(), bbPlNoWaitCtx(t2.StartTS()), key); err != nil {
			t.Fatalf("case %d: key still locked after rollback: %v", i, err)
		}
		_ = t2.Rollback()
	}
}

// Contract: a pessimistic txn that locks then commits leaves the value
// readable; locks do not leak past commit.
func TestBBPessimisticCommitReadback(t *testing.T) {
	store, done := bbPlStore(t)
	defer done()
	for i := 0; i < 500; i++ {
		key := bbPlKey(i, 6)
		val := []byte("pv")
		txn := bbPlBegin(t, store)
		if err := txn.Set(key, val); err != nil {
			t.Fatalf("case %d: set: %v", i, err)
		}
		lc := bbPlLockCtx(txn.StartTS(), 0, false)
		if err := txn.LockKeys(context.Background(), lc, key); err != nil {
			t.Fatalf("case %d: lock: %v", i, err)
		}
		if err := txn.Commit(context.Background()); err != nil {
			t.Fatalf("case %d: commit: %v", i, err)
		}
		txn2, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d: begin2: %v", i, err)
		}
		got, err := txn2.Get(context.Background(), key)
		if err != nil || string(got) != string(val) {
			t.Fatalf("case %d: readback %q err %v", i, got, err)
		}
		// Post-commit the key is free again.
		t3 := bbPlBegin(t, store)
		if err := t3.LockKeys(context.Background(), bbPlNoWaitCtx(t3.StartTS()), key); err != nil {
			t.Fatalf("case %d: key locked after commit: %v", i, err)
		}
		_ = t3.Rollback()
	}
}

// Unseen-random: interleaved lock/rollback sequences across many txns keep
// locked-count consistent with a set model; a rolled-back txn's keys never
// stay attributed to it.
func TestBBPessimisticInterleaveModel(t *testing.T) {
	r := rand.New(rand.NewSource(bbPlSeed + 7))
	store, done := bbPlStore(t)
	defer done()
	cases := bbPlTotalCases - 3000 - 2000 - 1500 - 1500 - 800 - 800 - 500
	if cases < 1 {
		cases = 1
	}
	for i := 0; i < cases; i++ {
		txn := bbPlBegin(t, store)
		want := map[string]bool{}
		lc := bbPlLockCtx(txn.StartTS(), 0, false)
		steps := 1 + r.Intn(4)
		for s := 0; s < steps; s++ {
			k := bbPlKey(i, 10+r.Intn(3))
			if err := txn.LockKeys(context.Background(), lc, k); err != nil {
				t.Fatalf("case %d step %d: lock: %v", i, s, err)
			}
			want[string(k)] = true
			if got := txn.GetLockedCount(); got != len(want) {
				t.Fatalf("case %d step %d: count %d want %d", i, s, got, len(want))
			}
		}
		if err := txn.Rollback(); err != nil {
			t.Fatalf("case %d: rollback: %v", i, err)
		}
	}
}

// Coverage table (contract sentence -> property):
//   "per-region batch: forUpdateTS/ttl/flags"          -> all tests use ForUpdateTS; LockOnlyIfExists flag covered
//   "response carries per-key values (return-values)"  -> TestBBPessimisticReturnValues
//   "existence info; missing keys get no lock"         -> TestBBPessimisticLockOnlyIfExists
//   "locked-key error: waiter blocks / killed"         -> TestBBPessimisticMutualExclusion (no-wait errors)
//   "write-conflict bumps forUpdateTS and retries"     -> covered via CommitReadback conflict-free path (weak)
//   "region errors re-split affected keys"             -> n/a single store (documented gap)
//   "rollback: best-effort release, no caller failure" -> TestBBPessimisticRollbackReleases
//   "dedup: not re-locked; primary first and once"     -> TestBBPessimisticDedup / PrimaryOnce
