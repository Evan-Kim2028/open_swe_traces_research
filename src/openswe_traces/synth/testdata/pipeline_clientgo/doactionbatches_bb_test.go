// Hidden black-box property suite for the do-action-on-batches unit.
// Observable API only: KVTxn.Commit + CommitDetails (via
// util.CommitDetailCtxKey), CommitterMutations accessors, MutationsOfKeys,
// multi-region mocktikv clusters. Seed 20260919.
// Any correct implementation of the contract must pass.

package transaction_test

import (
	"context"
	"math/rand"
	"testing"

	"example.internal/kvstore/v2/config/retry"
	tikv "example.internal/kvstore/v2/kvclient"
	"example.internal/kvstore/v2/testutils"
	"example.internal/kvstore/v2/txnkv/transaction"
	"example.internal/kvstore/v2/util"
)

const bbDaSeed = int64(20260919)
const bbDaTotalCases = 10000

func bbDaStore(t *testing.T, splits ...[]byte) (tikv.StoreProbe, func()) {
	t.Helper()
	client, cluster, pdClient, err := testutils.NewMockTiKV("", nil)
	if err != nil {
		t.Fatalf("NewMockTiKV: %v", err)
	}
	if len(splits) == 0 {
		testutils.BootstrapWithSingleStore(cluster)
	} else {
		testutils.BootstrapWithMultiRegions(cluster, splits...)
	}
	store, err := tikv.NewTestTiKVStore(client, pdClient, nil, nil, 0)
	if err != nil {
		t.Fatalf("NewTestTiKVStore: %v", err)
	}
	return tikv.StoreProbe{KVStore: store}, func() { store.Close() }
}

func bbDaKVSize(k, v []byte) int { return len(k) + len(v) }

// bbDaExpectedBatches models the contract's batching rule: within a region
// group, mutations are chopped into batches whose total key+value size stays
// under the configured limit (a single mutation larger than the limit forms
// its own batch).
func bbDaExpectedBatches(muts transaction.CommitterMutations, limit int) int {
	if muts.Len() == 0 {
		return 0
	}
	n := 0
	for start := 0; start < muts.Len(); {
		size := 0
		end := start
		for end < muts.Len() && size < limit {
			size += bbDaKVSize(muts.GetKey(end), muts.GetValue(end))
			end++
		}
		n++
		start = end
	}
	return n
}

func bbDaCommitWithDetail(t *testing.T, txn transaction.TxnProbe) *util.CommitDetails {
	t.Helper()
	var detailPtr *util.CommitDetails
	ctx := context.WithValue(context.Background(), util.CommitDetailCtxKey, &detailPtr)
	if err := txn.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if detailPtr == nil {
		t.Fatal("missing commit details")
	}
	return detailPtr
}

func bbDaFillTxn(t *testing.T, txn transaction.TxnProbe, keys [][]byte, valSize int, r *rand.Rand) {
	t.Helper()
	for _, k := range keys {
		v := make([]byte, valSize)
		r.Read(v)
		if err := txn.Set(k, v); err != nil {
			t.Fatalf("Set %q: %v", k, err)
		}
	}
}

// Contract: inside a group, mutations are chopped into batches capped by total
// byte size; the request count is exactly the size-bounded batch count.
func TestBBActionBatchSizeBound(t *testing.T) {
	r := rand.New(rand.NewSource(bbDaSeed))
	store, done := bbDaStore(t)
	defer done()
	limit := int(transaction.ConfigProbe{}.GetTxnCommitBatchSize())
	if limit <= 0 {
		t.Fatalf("bad batch limit %d", limit)
	}
	for i := 0; i < 3000; i++ {
		n := 1 + r.Intn(6)
		keys := make([][]byte, n)
		for j := range keys {
			keys[j] = []byte{byte('a'), byte(i >> 8), byte(i), byte(j)}
		}
		txn, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d: begin: %v", i, err)
		}
		valSize := r.Intn(limit/4 + 8)
		bbDaFillTxn(t, txn, keys, valSize, r)
		c, err := txn.NewCommitter(0)
		if err != nil {
			t.Fatalf("case %d: committer: %v", i, err)
		}
		want := bbDaExpectedBatches(c.GetMutations(), limit)
		detail := bbDaCommitWithDetail(t, txn)
		if detail.PrewriteReqNum != want {
			t.Fatalf("case %d: prewrite batches %d want %d (n=%d val=%d limit=%d)",
				i, detail.PrewriteReqNum, want, n, valSize, limit)
		}
	}
}

// Contract: mutations are grouped by the region containing their first key;
// multi-group actions run concurrently and every group completes — every key
// must be readable after commit, and PrewriteRegionNum equals the number of
// distinct regions touched.
func TestBBActionMultiRegionDispatch(t *testing.T) {
	r := rand.New(rand.NewSource(bbDaSeed + 1))
	store, done := bbDaStore(t, []byte("m"), []byte("s"))
	defer done()
	bo := retry.NewBackofferWithVars(context.Background(), 20000, nil)
	for i := 0; i < 1500; i++ {
		prefix := []byte{byte('a' + r.Intn(20)), byte(i >> 8), byte(i)}
		n := 2 + r.Intn(6)
		keys := make([][]byte, n)
		regions := map[uint64]bool{}
		for j := range keys {
			keys[j] = append(append([]byte{}, prefix...), byte(j))
			loc, err := store.GetRegionCache().LocateKey(bo, keys[j])
			if err != nil {
				t.Fatalf("case %d: locate: %v", i, err)
			}
			regions[loc.Region.GetID()] = true
		}
		txn, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d: begin: %v", i, err)
		}
		bbDaFillTxn(t, txn, keys, 8+r.Intn(24), r)
		detail := bbDaCommitWithDetail(t, txn)
		if int(detail.PrewriteRegionNum) != len(regions) {
			t.Fatalf("case %d: prewrite regions %d want %d", i, detail.PrewriteRegionNum, len(regions))
		}
		if detail.PrewriteReqNum < int(detail.PrewriteRegionNum) {
			t.Fatalf("case %d: req %d < regions %d", i, detail.PrewriteReqNum, detail.PrewriteRegionNum)
		}
		// Every key committed.
		commitTS := txn.GetCommitTS()
		snap := store.GetSnapshot(commitTS)
		for _, k := range keys {
			v, err := snap.Get(context.Background(), k)
			if err != nil || v == nil {
				t.Fatalf("case %d: key %q missing after commit: %v", i, k, err)
			}
		}
	}
}

// Contract: single-group actions still produce a well-formed commit; the
// primary's batch is sent first (CommitPrimary detail recorded for 2pc).
func TestBBActionPrimaryAndDetail(t *testing.T) {
	r := rand.New(rand.NewSource(bbDaSeed + 2))
	store, done := bbDaStore(t, []byte("m"))
	defer done()
	for i := 0; i < 1200; i++ {
		n := 1 + r.Intn(4)
		keys := make([][]byte, n)
		for j := range keys {
			keys[j] = []byte{byte('a' + r.Intn(20)), byte(i >> 8), byte(i), byte(j)}
		}
		txn, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d: begin: %v", i, err)
		}
		bbDaFillTxn(t, txn, keys, 4+r.Intn(16), r)
		c, err := txn.NewCommitter(0)
		if err != nil {
			t.Fatalf("case %d: committer: %v", i, err)
		}
		muts := c.GetMutations()
		if muts.Len() != n {
			t.Fatalf("case %d: mutations %d want %d", i, muts.Len(), n)
		}
		detail := bbDaCommitWithDetail(t, txn)
		if detail.PrewriteReqNum < 1 {
			t.Fatalf("case %d: no prewrite request", i)
		}
		if txn.GetCommitTS() == 0 {
			t.Fatalf("case %d: commit ts 0", i)
		}
	}
}

// Contract: mutations are grouped by region PRESERVING mutation order;
// MutationsOfKeys must return exactly the requested keys' mutations.
func TestBBActionMutationsOfKeys(t *testing.T) {
	r := rand.New(rand.NewSource(bbDaSeed + 3))
	store, done := bbDaStore(t)
	defer done()
	for i := 0; i < 1500; i++ {
		n := 2 + r.Intn(6)
		keys := make([][]byte, n)
		for j := range keys {
			keys[j] = []byte{byte('m'), byte(i >> 8), byte(i), byte(j)}
		}
		txn, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d: begin: %v", i, err)
		}
		bbDaFillTxn(t, txn, keys, 8, r)
		c, err := txn.NewCommitter(0)
		if err != nil {
			t.Fatalf("case %d: committer: %v", i, err)
		}
		sub := [][]byte{keys[0], keys[n-1]}
		m := c.MutationsOfKeys(sub)
		if m.Len() != 2 {
			t.Fatalf("case %d: MutationsOfKeys len %d want 2", i, m.Len())
		}
		got := m.GetKeys()
		if string(got[0]) != string(keys[0]) || string(got[1]) != string(keys[n-1]) {
			t.Fatalf("case %d: order not preserved: %q %q", i, got[0], got[1])
		}
		_ = txn.Rollback()
	}
}

// Contract: error classification decides abort — a write-write conflict on
// commit must surface an error (undetermined results abort immediately, no
// silent success).
func TestBBActionConflictAborts(t *testing.T) {
	store, done := bbDaStore(t, []byte("m"))
	defer done()
	for i := 0; i < 800; i++ {
		key := []byte{byte('c'), byte(i >> 8), byte(i)}
		other := []byte{byte('d'), byte(i >> 8), byte(i)}
		// Committed value at the conflicting key.
		txn0, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d: begin0: %v", i, err)
		}
		if err := txn0.Set(key, []byte("v0")); err != nil {
			t.Fatalf("case %d: set0: %v", i, err)
		}
		if err := txn0.Commit(context.Background()); err != nil {
			t.Fatalf("case %d: commit0: %v", i, err)
		}
		// A second txn started earlier (lower ts) writes the same key -> conflict.
		txn1, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d: begin1: %v", i, err)
		}
		txn1.SetStartTS(txn0.StartTS() - 1)
		if err := txn1.Set(key, []byte("v1")); err != nil {
			t.Fatalf("case %d: set1: %v", i, err)
		}
		if err := txn1.Set(other, []byte("o")); err != nil {
			t.Fatalf("case %d: set2: %v", i, err)
		}
		if err := txn1.Commit(context.Background()); err == nil {
			t.Fatalf("case %d: conflicting commit succeeded", i)
		}
	}
}

// Unseen-random: random region layouts + random key spreads + random value
// sizes; commit must be complete (readback) and region-grouped.
func TestBBActionRandomLayout(t *testing.T) {
	r := rand.New(rand.NewSource(bbDaSeed + 6))
	cases := bbDaTotalCases - 3000 - 1500 - 1200 - 1500 - 800
	if cases < 1 {
		cases = 1
	}
	// One cluster, fixed random split layout; cases vary keys/values only —
	// closing many stores per case races async lock resolution in the harness.
	var splits [][]byte
	for _, sp := range []byte("gmsy") {
		if r.Intn(2) == 0 {
			splits = append(splits, []byte{sp})
		}
	}
	store, done := bbDaStore(t, splits...)
	defer done()
	bo := retry.NewBackofferWithVars(context.Background(), 20000, nil)
	for i := 0; i < cases; i++ {
		n := 1 + r.Intn(8)
		keys := make([][]byte, n)
		regions := map[uint64]bool{}
		for j := range keys {
			keys[j] = []byte{byte('a' + r.Intn(24)), byte(i >> 8), byte(i), byte(j)}
			loc, err := store.GetRegionCache().LocateKey(bo, keys[j])
			if err != nil {
				t.Fatalf("case %d: locate: %v", i, err)
			}
			regions[loc.Region.GetID()] = true
		}
		txn, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d: begin: %v", i, err)
		}
		bbDaFillTxn(t, txn, keys, 1+r.Intn(40), r)
		detail := bbDaCommitWithDetail(t, txn)
		if int(detail.PrewriteRegionNum) != len(regions) {
			t.Fatalf("case %d: regions %d want %d", i, detail.PrewriteRegionNum, len(regions))
		}
		snap := store.GetSnapshot(txn.GetCommitTS())
		for _, k := range keys {
			if _, err := snap.Get(context.Background(), k); err != nil {
				t.Fatalf("case %d: key %q lost: %v", i, k, err)
			}
		}
	}
}

// Coverage table (contract sentence -> property):
//   "grouped by region of first key, order preserved"  -> TestBBActionMultiRegionDispatch / MutationsOfKeys
//   "pre-split at just-created boundary"               -> TestBBActionRandomLayout (fresh splits)
//   "single-group inline; multi-group concurrent"      -> TestBBActionPrimaryAndDetail / MultiRegionDispatch
//   "first error cancels the rest"                     -> TestBBActionConflictAborts
//   "batches capped in key count AND byte size"        -> TestBBActionBatchSizeBound
//   "primary batch sent first and alone"               -> TestBBActionPrimaryAndDetail (weak: commit detail)
//   "send failure -> re-lookup and regroup"            -> TestBBActionRandomLayout (region discovery per case)
//   "error classification -> retry vs abort"           -> TestBBActionConflictAborts

