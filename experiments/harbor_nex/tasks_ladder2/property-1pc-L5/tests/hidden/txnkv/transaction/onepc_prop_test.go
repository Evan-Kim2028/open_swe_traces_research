package transaction

import (
	"context"
	"math/rand"
	"testing"

	"example.internal/kvstore/v2/config"
	"example.internal/kvstore/v2/internal/unionstore"
	"example.internal/kvstore/v2/oracle"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
)

const onepcSeed = 20260918
const onepcCases = 10000

type dummyBinlog struct{}

func (dummyBinlog) Prewrite(context.Context, []byte) <-chan BinlogWriteResult { return nil }
func (dummyBinlog) Commit(context.Context, int64)                             {}
func (dummyBinlog) Skip()                                                     {}

func onepcMutations(keys [][]byte) *memBufferMutations {
	db := unionstore.NewMemDBWithContext().GetMemDB()
	mut := newMemBufferMutations(len(keys), db)
	for _, k := range keys {
		val := append([]byte("v"), k...)
		if err := db.Set(k, val); err != nil {
			panic(err)
		}
	}
	it := db.IterWithFlags(nil, nil)
	for it.Valid() {
		mut.Push(kvrpcpb.Op_Put, false, false, false, false, it.Handle())
		if err := it.Next(); err != nil {
			panic(err)
		}
	}
	return mut
}

func onepcCommitter(scope string, enable1PC, enableAsync bool, keys [][]byte, binlog bool, bound bool) *twoPhaseCommitter {
	txn := &KVTxn{
		scope:             scope,
		enable1PC:         enable1PC,
		enableAsyncCommit: enableAsync,
	}
	if bound {
		txn.commitTSUpperBoundCheck = func(uint64) bool { return true }
	}
	c := &twoPhaseCommitter{txn: txn, mutations: onepcMutations(keys)}
	if binlog {
		c.binlog = dummyBinlog{}
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

func TestOnePCAsyncDecisionProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(onepcSeed))
	cfg := config.GetGlobalConfig().TiKVClient.AsyncCommit
	for i := 0; i < onepcCases; i++ {
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
		c := onepcCommitter(scope, enable1, enableA, keys, binlog, bound)
		nmut := 0
		size := 0
		if c.mutations != nil {
			nmut = c.mutations.Len()
			for j := 0; j < nmut; j++ {
				size += len(c.mutations.GetKey(j))
			}
		}
		if c.checkOnePC() != wantOnePC(scope, enable1, binlog, bound) {
			t.Fatalf("case %d one-pc got %v", i, c.checkOnePC())
		}
		if c.checkAsyncCommit() != wantAsync(scope, enableA, binlog, bound, nmut, size) {
			t.Fatalf("case %d async got %v want %v n=%d total=%d", i, c.checkAsyncCommit(),
				wantAsync(scope, enableA, binlog, bound, nmut, size), nmut, size)
		}
	}
}

func TestOnePCContractExamples(t *testing.T) {
	c := onepcCommitter(oracle.GlobalTxnScope, true, true, [][]byte{[]byte("k")}, false, false)
	if !c.checkOnePC() || !c.checkAsyncCommit() {
		t.Fatal("global enabled no-binlog must allow both")
	}
	local := onepcCommitter("local", true, true, [][]byte{[]byte("k")}, false, false)
	if local.checkOnePC() || local.checkAsyncCommit() {
		t.Fatal("local scope must refuse both")
	}
	withBin := onepcCommitter(oracle.GlobalTxnScope, true, true, [][]byte{[]byte("k")}, true, false)
	if withBin.checkOnePC() || withBin.checkAsyncCommit() {
		t.Fatal("binlog must refuse both")
	}
	withBound := onepcCommitter(oracle.GlobalTxnScope, true, true, [][]byte{[]byte("k")}, false, true)
	if withBound.checkOnePC() || withBound.checkAsyncCommit() {
		t.Fatal("commit-ts bound check must refuse both")
	}
}

func TestOnePCUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(onepcSeed + 5))
	for i := 0; i < onepcCases; i++ {
		n := 1 + rng.Intn(4)
		keys := make([][]byte, n)
		for j := 0; j < n; j++ {
			keys[j] = []byte{byte(rng.Intn(200) + 20)}
		}
		c := onepcCommitter(oracle.GlobalTxnScope, true, true, keys, false, false)
		if !c.checkOnePC() {
			t.Fatalf("unseen one-pc %d", i)
		}
		nmut, size := 0, 0
		if c.mutations != nil {
			nmut = c.mutations.Len()
			for j := 0; j < nmut; j++ {
				size += len(c.mutations.GetKey(j))
			}
		}
		if c.checkAsyncCommit() != wantAsync(oracle.GlobalTxnScope, true, false, false, nmut, size) {
			t.Fatalf("unseen async %d", i)
		}
	}
}
