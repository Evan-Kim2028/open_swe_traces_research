package retry

import (
	"math/rand"
	"testing"

	"example.internal/kvstore/v2/kv"
)

const policySeed = 20260918
const policyCases = 10000

type policyRow struct {
	cfg  *Config
	name string
	base int
	cap  int
	jit  int
}

func policyTable() []policyRow {
	return []policyRow{
		{BoTiKVRPC, "tikvRPC", 100, 2000, EqualJitter},
		{BoTiFlashRPC, "tiflashRPC", 100, 2000, EqualJitter},
		{BoTxnLock, "txnLock", 100, 3000, EqualJitter},
		{BoPDRPC, "pdRPC", 500, 3000, EqualJitter},
		{BoRegionMiss, "regionMiss", 2, 500, NoJitter},
		{BoRegionScheduling, "regionScheduling", 2, 500, NoJitter},
		{BoTiKVServerBusy, "tikvServerBusy", 2000, 10000, EqualJitter},
		{BoTiKVDiskFull, "tikvDiskFull", 500, 5000, NoJitter},
		{BoRegionRecoveryInProgress, "regionRecoveryInProgress", 100, 10000, EqualJitter},
		{BoTiFlashServerBusy, "tiflashServerBusy", 2000, 10000, EqualJitter},
		{BoTxnNotFound, "txnNotFound", 2, 500, NoJitter},
		{BoStaleCmd, "staleCommand", 2, 1000, NoJitter},
		{BoMaxTsNotSynced, "maxTsNotSynced", 2, 500, NoJitter},
		{BoMaxDataNotReady, "dataNotReady", 2, 2000, NoJitter},
		{BoMaxRegionNotInitialized, "regionNotInitialized", 2, 1000, NoJitter},
		{BoIsWitness, "isWitness", 1000, 10000, EqualJitter},
		{BoTxnLockFast, "txnLockFast", 2, 3000, EqualJitter},
	}
}

func TestBackoffPolicyTableProperty(t *testing.T) {
	rows := policyTable()
	rng := rand.New(rand.NewSource(policySeed))
	for i := 0; i < policyCases; i++ {
		row := rows[rng.Intn(len(rows))]
		if row.cfg.String() != row.name {
			t.Fatalf("case %d name %q want %q", i, row.cfg.String(), row.name)
		}
		if row.cfg.fnCfg == nil {
			t.Fatalf("case %d missing envelope", i)
		}
		if row.cfg.fnCfg.base != row.base || row.cfg.fnCfg.cap != row.cap || row.cfg.fnCfg.jitter != row.jit {
			t.Fatalf("case %d %s base/cap/jitter=%d/%d/%d want %d/%d/%d",
				i, row.name, row.cfg.fnCfg.base, row.cfg.fnCfg.cap, row.cfg.fnCfg.jitter, row.base, row.cap, row.jit)
		}
	}
	for _, row := range rows {
		if row.cfg.fnCfg.base != row.base || row.cfg.fnCfg.cap != row.cap || row.cfg.fnCfg.jitter != row.jit {
			t.Fatalf("adversarial %s", row.name)
		}
	}
	if isSleepExcluded[BoTiKVServerBusy.name] != 600000 {
		t.Fatalf("busy exclusion %d", isSleepExcluded[BoTiKVServerBusy.name])
	}
}

func TestBackoffPolicyContractExamples(t *testing.T) {
	if BoRegionMiss.String() != "regionMiss" || BoRegionMiss.fnCfg.base != 2 || BoRegionMiss.fnCfg.cap != 500 || BoRegionMiss.fnCfg.jitter != NoJitter {
		t.Fatalf("region-miss envelope %+v", BoRegionMiss.fnCfg)
	}
	if BoTxnLock.fnCfg.base != 100 || BoTxnLock.fnCfg.cap != 3000 || BoTxnLock.fnCfg.jitter != EqualJitter {
		t.Fatalf("txn-lock envelope %+v", BoTxnLock.fnCfg)
	}
	if BoTiKVServerBusy.fnCfg.base != 2000 || BoTiKVServerBusy.fnCfg.cap != 10000 {
		t.Fatalf("server-busy envelope %+v", BoTiKVServerBusy.fnCfg)
	}
	cfg := NewConfig("probe", nil, NewBackoffFnCfg(4, 40, NoJitter), nil)
	if cfg.String() != "probe" || cfg.fnCfg.base != 4 || cfg.fnCfg.cap != 40 {
		t.Fatalf("constructor %+v", cfg.fnCfg)
	}
	if BoTxnLockFast.name != txnLockFastName {
		t.Fatal("lock-fast must keep the distinguished name")
	}
	_ = kv.Variables{BackoffLockFast: 17}
}

func TestBackoffPolicyUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(policySeed + 9))
	rows := policyTable()
	skip := map[string]bool{"regionMiss": true, "txnLock": true, "tikvServerBusy": true, "probe": true}
	for i := 0; i < policyCases; i++ {
		row := rows[rng.Intn(len(rows))]
		if skip[row.name] {
			continue
		}
		if row.cfg.fnCfg.base != row.base || row.cfg.fnCfg.cap != row.cap {
			t.Fatalf("unseen %d %s", i, row.name)
		}
	}
}
