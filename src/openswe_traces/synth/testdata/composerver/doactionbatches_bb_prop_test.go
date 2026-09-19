package transaction_test

import (
	"bytes"
	"context"
	"math/rand"
	"sync"
	"testing"

	"github.com/pingcap/failpoint"
	"example.internal/kvstore/v2/kv"
	tikv "example.internal/kvstore/v2/kvclient"
	"example.internal/kvstore/v2/testutils"
	"example.internal/kvstore/v2/util"
	"example.internal/kvstore/v2/wirerpc"
	"example.internal/kvstore/v2/txnkv/transaction"
)

const doActionBBSeed = 20260919
const doActionBBCases = 10000

func newBatchStore(t *testing.T, multiRegion bool) (tikv.StoreProbe, func()) {
	t.Helper()
	client, cluster, pdClient, err := testutils.NewMockTiKV("", nil)
	if err != nil {
		t.Fatalf("NewMockTiKV: %v", err)
	}
	if multiRegion {
		testutils.BootstrapWithMultiRegions(cluster, []byte("m"), []byte("s"))
	} else {
		testutils.BootstrapWithSingleStore(cluster)
	}
	store, err := tikv.NewTestTiKVStore(client, pdClient, nil, nil, 0)
	if err != nil {
		t.Fatalf("NewTestTiKVStore: %v", err)
	}
	return tikv.StoreProbe{KVStore: store}, func() { store.Close() }
}

func keyValueSize(k, v []byte) int {
	return len(k) + len(v)
}

func expectedPrewriteBatches(muts transaction.CommitterMutations, limit int) int {
	if muts.Len() == 0 {
		return 0
	}
	n := 0
	for start := 0; start < muts.Len(); {
		size := 0
		end := start
		for end < muts.Len() && size < limit {
			size += keyValueSize(muts.GetKey(end), muts.GetValue(end))
			end++
		}
		n++
		start = end
	}
	return n
}

func commitWithDetail(t *testing.T, txn transaction.TxnProbe) *util.CommitDetails {
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

func TestPrewriteBatchSizeProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(doActionBBSeed))
	store, done := newBatchStore(t, false)
	defer done()
	limit := int(tikv.ConfigProbe{}.GetTxnCommitBatchSize())
	for i := 0; i < doActionBBCases; i++ {
		txn, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d begin: %v", i, err)
		}
		nkeys := 1 + rng.Intn(12)
		for j := 0; j < nkeys; j++ {
			k := []byte{byte('b'), byte(i >> 8), byte(i), byte(j)}
			vlen := 1 + rng.Intn(8)
			v := bytes.Repeat([]byte{byte('v')}, vlen)
			if err := txn.Set(k, v); err != nil {
				t.Fatalf("case %d set: %v", i, err)
			}
		}
		c, err := txn.NewCommitter(1)
		if err != nil {
			t.Fatalf("case %d committer: %v", i, err)
		}
		if err := c.InitKeysAndMutations(); err != nil {
			t.Fatalf("case %d init: %v", i, err)
		}
		want := expectedPrewriteBatches(c.GetMutations(), limit)
		detail := commitWithDetail(t, txn)
		if detail.PrewriteReqNum != want {
			t.Fatalf("case %d prewrite batches got %d want %d keys=%d", i, detail.PrewriteReqNum, want, nkeys)
		}
	}
}

func TestMultiRegionDispatchProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(doActionBBSeed + 1))
	store, done := newBatchStore(t, true)
	defer done()
	for i := 0; i < doActionBBCases; i++ {
		txn, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d begin: %v", i, err)
		}
		// Split boundaries are at "m" and "s"; pick distinct region prefixes.
		regionPrefixes := [][]byte{[]byte("a"), []byte("m"), []byte("s")}
		regions := 1 + rng.Intn(3)
		for r := 0; r < regions; r++ {
			prefix := regionPrefixes[r]
			k := append(append([]byte(nil), prefix...), byte(i>>8), byte(i))
			if err := txn.Set(k, []byte("v")); err != nil {
				t.Fatalf("case %d set: %v", i, err)
			}
		}
		detail := commitWithDetail(t, txn)
		if int(detail.PrewriteRegionNum) < regions {
			t.Fatalf("case %d region num got %d want >= %d", i, detail.PrewriteRegionNum, regions)
		}
	}
}

func TestPrimaryFirstPrewriteProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(doActionBBSeed + 2))
	store, done := newBatchStore(t, false)
	defer done()
	if err := failpoint.Enable("tikvclient/twoPCRequestBatchSizeLimit", "return"); err != nil {
		t.Fatalf("failpoint: %v", err)
	}
	defer failpoint.Disable("tikvclient/twoPCRequestBatchSizeLimit")

	var mu sync.Mutex
	primaryFirst := make([]bool, 0, 4)
	hook := func(req *tikvrpc.Request) {
		if req.Type != tikvrpc.CmdPrewrite {
			return
		}
		pr := req.Prewrite()
		mu.Lock()
		primaryFirst = append(primaryFirst, len(pr.Mutations) == 1 && bytes.Equal(pr.PrimaryLock, pr.Mutations[0].Key))
		mu.Unlock()
	}
	if err := failpoint.Enable("tikvclient/beforeSendReqToRegion", "return"); err != nil {
		t.Fatalf("failpoint hook: %v", err)
	}
	defer failpoint.Disable("tikvclient/beforeSendReqToRegion")
	ctx := context.WithValue(context.Background(), "sendReqToRegionHook", hook)

	for i := 0; i < doActionBBCases; i++ {
		mu.Lock()
		primaryFirst = primaryFirst[:0]
		mu.Unlock()
		txn, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d begin: %v", i, err)
		}
		n := 2 + rng.Intn(4)
		for j := 0; j < n; j++ {
			k := []byte{byte('p'), byte(i >> 8), byte(i), byte(j)}
			if err := txn.Set(k, bytes.Repeat([]byte("x"), 1+rng.Intn(4))); err != nil {
				t.Fatalf("case %d set: %v", i, err)
			}
		}
		c, err := txn.NewCommitter(uint64(i + 1))
		if err != nil {
			t.Fatalf("case %d committer: %v", i, err)
		}
		if err := c.PrewriteAllMutations(ctx); err != nil {
			t.Fatalf("case %d prewrite: %v", i, err)
		}
		mu.Lock()
		if len(primaryFirst) == 0 {
			t.Fatalf("case %d no prewrite requests", i)
		}
		if !primaryFirst[0] {
			t.Fatalf("case %d first batch not primary-only", i)
		}
		mu.Unlock()
	}
}

func TestDoActionBatchesContractExamples(t *testing.T) {
	store, done := newBatchStore(t, true)
	defer done()
	txn, err := store.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := txn.Set([]byte("a-key"), []byte("v")); err != nil {
		t.Fatal(err)
	}
	if err := txn.Set([]byte("s-key"), []byte("v")); err != nil {
		t.Fatal(err)
	}
	detail := commitWithDetail(t, txn)
	if detail.PrewriteRegionNum < 2 {
		t.Fatalf("multi-region prewrite regions %d", detail.PrewriteRegionNum)
	}
	if detail.PrewriteReqNum < 1 {
		t.Fatal("prewrite requests")
	}
}

func TestDoActionBatchesUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(doActionBBSeed + 5))
	store, done := newBatchStore(t, false)
	defer done()
	limit := int(kv.TxnCommitBatchSize.Load())
	for i := 0; i < doActionBBCases; i++ {
		txn, err := store.Begin()
		if err != nil {
			t.Fatalf("case %d begin: %v", i, err)
		}
		n := 1 + rng.Intn(6)
		for j := 0; j < n; j++ {
			k := []byte{byte(180 + j), byte(i >> 8), byte(i)}
			v := []byte{byte(rng.Intn(200))}
			if err := txn.Set(k, v); err != nil {
				t.Fatalf("case %d set: %v", i, err)
			}
		}
		c, err := txn.NewCommitter(1)
		if err != nil {
			t.Fatalf("case %d committer: %v", i, err)
		}
		if err := c.InitKeysAndMutations(); err != nil {
			t.Fatalf("case %d init: %v", i, err)
		}
		want := expectedPrewriteBatches(c.GetMutations(), limit)
		detail := commitWithDetail(t, txn)
		if detail.PrewriteReqNum != want {
			t.Fatalf("case %d batches got %d want %d", i, detail.PrewriteReqNum, want)
		}
	}
}
