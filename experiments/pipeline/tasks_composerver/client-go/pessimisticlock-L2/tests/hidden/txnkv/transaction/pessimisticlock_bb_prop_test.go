package transaction_test

import (
	"bytes"
	"context"
	"math/rand"
	"testing"
	"time"

	"example.internal/kvstore/v2/kv"
	tikv "example.internal/kvstore/v2/kvclient"
	"example.internal/kvstore/v2/testutils"
	"example.internal/kvstore/v2/txnkv/transaction"
)

const pessimisticBBSeed = 20260919
const pessimisticBBCases = 10000

func newPessimisticStore(t *testing.T) (tikv.StoreProbe, func()) {
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

func beginPessimistic(t *testing.T, store tikv.StoreProbe) transaction.TxnProbe {
	t.Helper()
	txn, err := store.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	tp := transaction.TxnProbe{KVTxn: txn.KVTxn}
	tp.SetPessimistic(true)
	return tp
}

func lockCtx(forUpdate uint64, retN int, onlyExists bool) *kv.LockCtx {
	lc := &kv.LockCtx{ForUpdateTS: forUpdate, WaitStartTime: time.Now()}
	if retN > 0 {
		lc.InitReturnValues(retN)
	}
	lc.LockOnlyIfExists = onlyExists
	return lc
}

func TestPessimisticLockDedupProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(pessimisticBBSeed))
	store, done := newPessimisticStore(t)
	defer done()
	for i := 0; i < pessimisticBBCases; i++ {
		txn := beginPessimistic(t, store)
		n := 1 + rng.Intn(4)
		keys := make([][]byte, n)
		for j := 0; j < n; j++ {
			keys[j] = []byte{byte('d'), byte(i >> 8), byte(i), byte(j)}
		}
		lc := lockCtx(txn.StartTS(), 0, false)
		if err := txn.LockKeys(context.Background(), lc, keys...); err != nil {
			t.Fatalf("case %d lock: %v", i, err)
		}
		lc2 := lockCtx(txn.StartTS(), 0, false)
		if err := txn.LockKeys(context.Background(), lc2, keys...); err != nil {
			t.Fatalf("case %d relock: %v", i, err)
		}
		if txn.GetLockedCount() != n {
			t.Fatalf("case %d locked count got %d want %d", i, txn.GetLockedCount(), n)
		}
		if len(txn.CollectLockedKeys()) != n {
			t.Fatalf("case %d collected keys", i)
		}
		if err := txn.Rollback(); err != nil {
			t.Fatalf("case %d rollback: %v", i, err)
		}
	}
}

func TestPessimisticLockReturnValuesProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(pessimisticBBSeed + 1))
	store, done := newPessimisticStore(t)
	defer done()
	for i := 0; i < pessimisticBBCases; i++ {
		n := 1 + rng.Intn(3)
		keys := make([][]byte, n)
		vals := make([][]byte, n)
		for j := 0; j < n; j++ {
			keys[j] = []byte{byte('r'), byte(i >> 8), byte(i), byte(j)}
			vals[j] = bytes.Repeat([]byte{byte('V')}, 1+rng.Intn(3))
		}
		setup, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d setup begin: %v", i, err)
		}
		for j := 0; j < n; j++ {
			if err := setup.Set(keys[j], vals[j]); err != nil {
				t.Fatalf("case %d seed set: %v", i, err)
			}
		}
		if err := setup.Commit(context.Background()); err != nil {
			t.Fatalf("case %d seed commit: %v", i, err)
		}
		txn := beginPessimistic(t, store)
		lc := lockCtx(txn.StartTS(), n, false)
		if err := txn.LockKeys(context.Background(), lc, keys...); err != nil {
			t.Fatalf("case %d lock: %v", i, err)
		}
		if len(lc.Values) != n {
			t.Fatalf("case %d values len %d", i, len(lc.Values))
		}
		for j := 0; j < n; j++ {
			got := lc.Values[string(keys[j])]
			if !bytes.Equal(got.Value, vals[j]) {
				t.Fatalf("case %d key %d value", i, j)
			}
		}
		if err := txn.Rollback(); err != nil {
			t.Fatalf("case %d rollback: %v", i, err)
		}
	}
}

func TestPessimisticLockIfExistsProperty(t *testing.T) {
	store, done := newPessimisticStore(t)
	defer done()
	for i := 0; i < pessimisticBBCases; i++ {
		existKey := []byte{byte('e'), byte(i >> 8), byte(i), 1}
		missKey := []byte{byte('e'), byte(i >> 8), byte(i), 2}
		seed, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d seed: %v", i, err)
		}
		if err := seed.Set(existKey, existKey); err != nil {
			t.Fatalf("case %d set: %v", i, err)
		}
		if err := seed.Commit(context.Background()); err != nil {
			t.Fatalf("case %d commit: %v", i, err)
		}
		txn := beginPessimistic(t, store)
		pk := []byte{byte('e'), byte(i >> 8), byte(i), 0}
		lc0 := lockCtx(txn.StartTS(), 0, false)
		if err := txn.LockKeys(context.Background(), lc0, pk); err != nil {
			t.Fatalf("case %d primary: %v", i, err)
		}
		lc := lockCtx(txn.StartTS(), 2, true)
		if err := txn.LockKeys(context.Background(), lc, existKey, missKey); err != nil {
			t.Fatalf("case %d lock if exists: %v", i, err)
		}
		if !lc.Values[string(existKey)].Exists || lc.Values[string(missKey)].Exists {
			t.Fatalf("case %d existence map", i)
		}
		if txn.GetLockedCount() != 2 {
			t.Fatalf("case %d locked count %d", i, txn.GetLockedCount())
		}
		if err := txn.Rollback(); err != nil {
			t.Fatalf("case %d rollback: %v", i, err)
		}
	}
}

func TestPessimisticLockPrimaryProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(pessimisticBBSeed + 3))
	store, done := newPessimisticStore(t)
	defer done()
	for i := 0; i < pessimisticBBCases; i++ {
		txn := beginPessimistic(t, store)
		keys := [][]byte{
			[]byte{byte('z'), byte(i)},
			[]byte{byte('a'), byte(i)},
			[]byte{byte('m'), byte(i)},
		}
		if rng.Intn(2) == 0 {
			keys[0], keys[1] = keys[1], keys[0]
		}
		lc := lockCtx(txn.StartTS(), 0, false)
		if err := txn.LockKeys(context.Background(), lc, keys...); err != nil {
			t.Fatalf("case %d lock: %v", i, err)
		}
		pk := txn.GetCommitter().GetPrimaryKey()
		if len(pk) == 0 {
			t.Fatalf("case %d empty primary", i)
		}
		found := false
		for _, k := range keys {
			if bytes.Equal(k, pk) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("case %d primary not in keys", i)
		}
		if err := txn.Rollback(); err != nil {
			t.Fatalf("case %d rollback: %v", i, err)
		}
	}
}

func TestPessimisticLockContractExamples(t *testing.T) {
	store, done := newPessimisticStore(t)
	defer done()

	// dedup
	txn := beginPessimistic(t, store)
	lc := lockCtx(100, 0, false)
	if err := txn.LockKeys(context.Background(), lc, []byte("abc"), []byte("def")); err != nil {
		t.Fatal(err)
	}
	lc2 := lockCtx(100, 0, false)
	if err := txn.LockKeys(context.Background(), lc2, []byte("abc"), []byte("def")); err != nil {
		t.Fatal(err)
	}
	if len(txn.CollectLockedKeys()) != 2 {
		t.Fatal("dedup")
	}

	// return values
	key, key2 := []byte("key"), []byte("key2")
	seed, _ := store.Begin()
	seed.Set(key, key)
	seed.Set(key2, key2)
	seed.Commit(context.Background())
	txn = beginPessimistic(t, store)
	lc = lockCtx(txn.StartTS(), 2, false)
	if err := txn.LockKeys(context.Background(), lc, key, key2); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(lc.Values[string(key)].Value, key) {
		t.Fatal("return value key")
	}

	// lock only if exists
	exist := []byte("exist")
	miss := []byte("miss")
	seed2, _ := store.Begin()
	seed2.Set(exist, exist)
	seed2.Commit(context.Background())
	txn = beginPessimistic(t, store)
	pk := []byte("pk")
	txn.LockKeys(context.Background(), lockCtx(txn.StartTS(), 0, false), pk)
	lc = lockCtx(txn.StartTS(), 2, true)
	if err := txn.LockKeys(context.Background(), lc, exist, miss); err != nil {
		t.Fatal(err)
	}
	if lc.Values[string(miss)].Exists {
		t.Fatal("missing key must not exist")
	}
}

func TestPessimisticLockUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(pessimisticBBSeed + 5))
	store, done := newPessimisticStore(t)
	defer done()
	for i := 0; i < pessimisticBBCases; i++ {
		k := []byte{byte(200 + rng.Intn(40)), byte(i >> 8), byte(i)}
		txn := beginPessimistic(t, store)
		lc := lockCtx(txn.StartTS()+uint64(rng.Intn(4)), 0, false)
		if err := txn.LockKeys(context.Background(), lc, k); err != nil {
			t.Fatalf("case %d lock: %v", i, err)
		}
		if txn.GetLockedCount() != 1 {
			t.Fatalf("case %d count", i)
		}
		if err := txn.Rollback(); err != nil {
			t.Fatalf("case %d rollback: %v", i, err)
		}
	}
}
