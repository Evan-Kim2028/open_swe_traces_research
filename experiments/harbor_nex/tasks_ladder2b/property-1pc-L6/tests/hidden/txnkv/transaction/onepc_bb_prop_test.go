package transaction

import (
	"context"
	"math/rand"
	"testing"

	"example.internal/kvstore/v2/config"
	"example.internal/kvstore/v2/oracle"
	"example.internal/kvstore/v2/txnkv/txnsnapshot"
)

const onepcBBSeed = 20260918
const onepcBBCases = 10000

type dummyBinlog struct{}

func (dummyBinlog) Prewrite(context.Context, []byte) <-chan BinlogWriteResult { return nil }
func (dummyBinlog) Commit(context.Context, int64)                             {}
func (dummyBinlog) Skip()                                                     {}

func onepcBBTxn(t *testing.T, scope string, enable1PC, enableAsync bool, keys [][]byte, binlog bool, bound bool) TxnProbe {
	t.Helper()
	snap := txnsnapshot.NewTiKVSnapshot(nil, 1, 0)
	txn, err := NewTiKVTxn(nil, snap, 1, &TxnOptions{TxnScope: scope})
	if err != nil {
		t.Fatalf("NewTiKVTxn: %v", err)
	}
	txn.SetScope(scope)
	txn.SetEnable1PC(enable1PC)
	txn.SetEnableAsyncCommit(enableAsync)
	if binlog {
		txn.SetBinlogExecutor(dummyBinlog{})
	}
	if bound {
		txn.SetCommitTSUpperBoundCheck(func(uint64) bool { return true })
	}
	for _, k := range keys {
		if err := txn.Set(k, append([]byte("v"), k...)); err != nil {
			t.Fatalf("Set %q: %v", k, err)
		}
	}
	return TxnProbe{KVTxn: txn}
}

func onepcBBCommitter(t *testing.T, txn TxnProbe) CommitterProbe {
	t.Helper()
	c, err := txn.NewCommitter(1)
	if err != nil {
		t.Fatalf("NewCommitter: %v", err)
	}
	return c
}

func wantOnePC(scope string, enable1PC, binlog, bound bool) bool {
	if scope != oracle.GlobalTxnScope {
		return false
	}
	if bound {
		return false
	}
	return !binlog && enable1PC
}

func wantAsync(scope string, enableAsync, binlog, bound bool, nkeys int, total int) bool {
	if scope != oracle.GlobalTxnScope {
		return false
	}
	if bound {
		return false
	}
	cfg := config.GetGlobalConfig().TiKVClient.AsyncCommit
	if !enableAsync || uint(nkeys) > cfg.KeysLimit || binlog {
		return false
	}
	return uint64(total) <= cfg.TotalKeySizeLimit
}

func mutationStats(c CommitterProbe) (int, int) {
	muts := c.GetMutations()
	if muts == nil {
		return 0, 0
	}
	n := muts.Len()
	size := 0
	for j := 0; j < n; j++ {
		size += len(muts.GetKey(j))
	}
	return n, size
}

func TestOnePCAsyncDecisionProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(onepcBBSeed))
	cfg := config.GetGlobalConfig().TiKVClient.AsyncCommit
	for i := 0; i < onepcBBCases; i++ {
		scope := oracle.GlobalTxnScope
		if rng.Intn(4) == 0 {
			scope = "local"
		}
		enable1 := rng.Intn(2) == 0
		enableA := rng.Intn(2) == 0
		binlog := rng.Intn(5) == 0
		bound := rng.Intn(5) == 0
		n := rng.Intn(8)
		keys := make([][]byte, n)
		for j := 0; j < n; j++ {
			k := make([]byte, 1+rng.Intn(6))
			rng.Read(k)
			keys[j] = k
		}
		if rng.Intn(20) == 0 {
			n = int(cfg.KeysLimit) + 1
			keys = make([][]byte, n)
			for j := 0; j < n; j++ {
				keys[j] = []byte{byte(j >> 8), byte(j)}
			}
		}
		txn := onepcBBTxn(t, scope, enable1, enableA, keys, binlog, bound)
		c := onepcBBCommitter(t, txn)
		nmut, size := mutationStats(c)
		if c.CheckOnePC() != wantOnePC(scope, enable1, binlog, bound) {
			t.Fatalf("case %d one-pc got %v", i, c.CheckOnePC())
		}
		if c.CheckAsyncCommit() != wantAsync(scope, enableA, binlog, bound, nmut, size) {
			t.Fatalf("case %d async got %v want %v n=%d total=%d", i, c.CheckAsyncCommit(),
				wantAsync(scope, enableA, binlog, bound, nmut, size), nmut, size)
		}
	}
}

func TestOnePCContractExamples(t *testing.T) {
	c := onepcBBCommitter(t, onepcBBTxn(t, oracle.GlobalTxnScope, true, true, [][]byte{[]byte("k")}, false, false))
	if !c.CheckOnePC() || !c.CheckAsyncCommit() {
		t.Fatal("global enabled no-binlog must allow both")
	}
	local := onepcBBCommitter(t, onepcBBTxn(t, "local", true, true, [][]byte{[]byte("k")}, false, false))
	if local.CheckOnePC() || local.CheckAsyncCommit() {
		t.Fatal("local scope must refuse both")
	}
	withBin := onepcBBCommitter(t, onepcBBTxn(t, oracle.GlobalTxnScope, true, true, [][]byte{[]byte("k")}, true, false))
	if withBin.CheckOnePC() || withBin.CheckAsyncCommit() {
		t.Fatal("binlog must refuse both")
	}
	withBound := onepcBBCommitter(t, onepcBBTxn(t, oracle.GlobalTxnScope, true, true, [][]byte{[]byte("k")}, false, true))
	if withBound.CheckOnePC() || withBound.CheckAsyncCommit() {
		t.Fatal("commit-ts bound check must refuse both")
	}
}

func TestOnePCUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(onepcBBSeed + 5))
	for i := 0; i < onepcBBCases; i++ {
		n := 1 + rng.Intn(4)
		keys := make([][]byte, n)
		for j := 0; j < n; j++ {
			b := byte(rng.Intn(200) + 20)
			if b == 'k' {
				b++
			}
			keys[j] = []byte{b}
		}
		c := onepcBBCommitter(t, onepcBBTxn(t, oracle.GlobalTxnScope, true, true, keys, false, false))
		if !c.CheckOnePC() {
			t.Fatalf("unseen one-pc %d", i)
		}
		nmut, size := mutationStats(c)
		if c.CheckAsyncCommit() != wantAsync(oracle.GlobalTxnScope, true, false, false, nmut, size) {
			t.Fatalf("unseen async %d", i)
		}
	}
}
