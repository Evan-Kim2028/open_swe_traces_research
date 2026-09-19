package locate

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/pingcap/failpoint"
	"github.com/pingcap/kvproto/pkg/errorpb"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/metapb"
	"example.internal/kvstore/v2/config/retry"
	"example.internal/kvstore/v2/internal/apicodec"
	"example.internal/kvstore/v2/internal/client"
	"example.internal/kvstore/v2/internal/mockstore/mockkv"
	"example.internal/kvstore/v2/kv"
	"example.internal/kvstore/v2/oracle"
	"example.internal/kvstore/v2/wirerpc"
)

const replicaSelectorSeed = 20260919
const replicaSelectorCases = 10000

type bbAttempt struct {
	Addr        string
	ReplicaRead bool
	StaleRead   bool
}

type bbRSEnv struct {
	mvcc    mocktikv.MVCCStore
	cluster *mocktikv.Cluster
	cache   *RegionCache
	region  RegionVerID
	stores  []uint64
	peers   []uint64
}

func bbRSBoot() *bbRSEnv {
	mvcc := mocktikv.MustNewMVCCStore()
	cluster := mocktikv.NewCluster(mvcc)
	stores, peers, regionID, _ := mocktikv.BootstrapWithMultiStores(cluster, 3)
	pdCli := &BrineSpan{mocktikv.NewPDClient(cluster), apicodec.ZestRing(apicodec.ModeTxn)}
	return &bbRSEnv{
		mvcc:    mvcc,
		cluster: cluster,
		cache:   NewRegionCache(pdCli),
		region:  RegionVerID{regionID, 1, 1},
		stores:  stores,
		peers:   peers,
	}
}

func (e *bbRSEnv) Close() {
	e.cache.Close()
	e.mvcc.Close()
}

func bbRSLeaderAddr() string { return "store1" }

func bbRSStoreAddr(storeIdx int) string { return fmt.Sprintf("store%d", storeIdx+1) }

func bbRSEnableFastRetry(t *testing.T) {
	t.Helper()
	if err := failpoint.Enable("tikvclient/fastBackoffBySkipSleep", "return"); err != nil {
		t.Fatalf("enable fastBackoffBySkipSleep: %v", err)
	}
	if err := failpoint.Enable("tikvclient/skipStoreCheckUntilHealth", "return"); err != nil {
		t.Fatalf("enable skipStoreCheckUntilHealth: %v", err)
	}
}

func bbRSDisableFastRetry(t *testing.T) {
	t.Helper()
	_ = failpoint.Disable("tikvclient/fastBackoffBySkipSleep")
	_ = failpoint.Disable("tikvclient/skipStoreCheckUntilHealth")
}

func bbRSPinRand() { randIntn = func(n int) int { return 0 } }

func bbRSRestoreRand() { randIntn = rand.Intn }

func bbRSGetReq(stale bool, readType kv.ReplicaReadType) *tikvrpc.Request {
	req := tikvrpc.NewRequest(tikvrpc.CmdGet, &kvrpcpb.GetRequest{Key: []byte("key")})
	if stale {
		req.EnableStaleWithMixedReplicaRead()
		req.ReadReplicaScope = oracle.GlobalTxnScope
		req.TxnScope = oracle.GlobalTxnScope
	} else {
		req.ReplicaReadType = readType
		req.ReplicaRead = readType.IsFollowerRead()
	}
	return req
}

func bbRSNotLeaderErr(regionID uint64, leaderStoreIdx int, peers []uint64, stores []uint64) *errorpb.Error {
	return &errorpb.Error{NotLeader: &errorpb.NotLeader{
		RegionId: regionID,
		Leader:   &metapb.Peer{Id: peers[leaderStoreIdx], StoreId: stores[leaderStoreIdx]},
	}}
}

// TestReplicaSelectorAccessProperty: SendReqCtx must route leader / follower / stale
// reads to the prescribed first target and honor replica-read flags on each attempt.
func TestReplicaSelectorAccessProperty(t *testing.T) {
	bbRSEnableFastRetry(t)
	defer bbRSDisableFastRetry(t)
	bbRSPinRand()
	defer bbRSRestoreRand()

	rng := rand.New(rand.NewSource(replicaSelectorSeed))
	for i := 0; i < replicaSelectorCases; i++ {
		mode := rng.Intn(4)
		env := bbRSBoot()

		var attempts []bbAttempt
		var sendCount int
		stale := mode == 2 || mode == 3
		readType := kv.ReplicaReadLeader
		var opts []StoreSelectorOption
		switch mode {
		case 1:
			readType = kv.ReplicaReadMixed
		case 2:
			readType = kv.ReplicaReadMixed
		case 3:
			readType = kv.ReplicaReadMixed
			opts = []StoreSelectorOption{WithMatchLabels([]*metapb.StoreLabel{{Key: "id", Value: "2"}})}
		}
		req := bbRSGetReq(stale, readType)

		cli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
			attempts = append(attempts, bbAttempt{Addr: addr, ReplicaRead: req.ReplicaRead, StaleRead: req.StaleRead})
			sendCount++
			if sendCount == 1 && mode == 1 {
				return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{
					RegionError: &errorpb.Error{ServerIsBusy: &errorpb.ServerIsBusy{}},
				}}, nil
			}
			return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{Value: []byte("v")}}, nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		bo := retry.NewBackofferWithVars(context.Background(), 40000, nil)
		loc, err := env.cache.LocateKey(bo, []byte("key"))
		if err != nil {
			t.Fatalf("case %d locate: %v", i, err)
		}
		resp, _, _, err := sender.SendReqCtx(bo, req, loc.Region, client.ReadTimeoutShort, tikvrpc.TiKV, opts...)
		if err != nil {
			t.Fatalf("case %d send: %v", i, err)
		}
		re, _ := resp.GetRegionError()
		if re != nil {
			t.Fatalf("case %d region err: %v", i, re)
		}
		if len(attempts) == 0 {
			t.Fatalf("case %d no attempts", i)
		}
		switch mode {
		case 0:
			if attempts[0].Addr != bbRSLeaderAddr() {
				t.Fatalf("case %d leader read first=%s want %s", i, attempts[0].Addr, bbRSLeaderAddr())
			}
			if attempts[0].ReplicaRead || attempts[0].StaleRead {
				t.Fatalf("case %d leader read flags rr=%v stale=%v", i, attempts[0].ReplicaRead, attempts[0].StaleRead)
			}
		case 1:
			if attempts[0].Addr != bbRSLeaderAddr() {
				t.Fatalf("case %d mixed follower first=%s want leader", i, attempts[0].Addr)
			}
			if sendCount > 1 {
				seen := map[string]bool{}
				for _, a := range attempts {
					seen[a.Addr] = true
				}
				if len(seen) < 2 {
					t.Fatalf("case %d busy should try another replica: %v", i, attempts)
				}
			}
		case 2:
			if !attempts[0].StaleRead {
				t.Fatalf("case %d stale read missing stale flag", i)
			}
			if attempts[0].ReplicaRead {
				t.Fatalf("case %d stale read should not set replica-read on first hop", i)
			}
		case 3:
			if attempts[0].Addr != bbRSStoreAddr(1) {
				t.Fatalf("case %d labeled stale first=%s want store2", i, attempts[0].Addr)
			}
			if !attempts[0].StaleRead {
				t.Fatalf("case %d labeled stale missing stale flag", i)
			}
		}
		env.Close()
	}
}

// TestReplicaSelectorFailoverProperty: send failures and not-leader hints must advance
// to a different store instead of hammering the same dead target.
func TestReplicaSelectorFailoverProperty(t *testing.T) {
	bbRSEnableFastRetry(t)
	defer bbRSDisableFastRetry(t)
	bbRSPinRand()
	defer bbRSRestoreRand()

	rng := rand.New(rand.NewSource(replicaSelectorSeed + 1))
	for i := 0; i < replicaSelectorCases; i++ {
		scenario := rng.Intn(3)
		env := bbRSBoot()

		var attempts []string
		var sendCount int
		readType := kv.ReplicaReadLeader
		if scenario == 2 {
			readType = kv.ReplicaReadMixed
		}
		req := bbRSGetReq(false, readType)
		cli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
			attempts = append(attempts, addr)
			sendCount++
			switch scenario {
			case 0:
				if sendCount == 1 {
					return nil, fmt.Errorf("rpc fail case %d", i)
				}
			case 1:
				if sendCount == 1 {
					return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{
						RegionError: bbRSNotLeaderErr(env.region.id, 1, env.peers, env.stores),
					}}, nil
				}
			case 2:
				if sendCount <= 2 {
					return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{
						RegionError: &errorpb.Error{ServerIsBusy: &errorpb.ServerIsBusy{}},
					}}, nil
				}
			}
			return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{Value: []byte("ok")}}, nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		bo := retry.NewBackofferWithVars(context.Background(), 40000, nil)
		loc, err := env.cache.LocateKey(bo, []byte("key"))
		if err != nil {
			t.Fatalf("case %d locate: %v", i, err)
		}
		resp, _, _, err := sender.SendReqCtx(bo, req, loc.Region, time.Second, tikvrpc.TiKV)
		if err != nil {
			t.Fatalf("case %d send: %v", i, err)
		}
		re, _ := resp.GetRegionError()
		if re != nil {
			t.Fatalf("case %d region err: %v", i, re)
		}
		if len(attempts) < 2 {
			t.Fatalf("case %d scenario %d attempts %v", i, scenario, attempts)
		}
		seen := map[string]bool{}
		for _, a := range attempts {
			seen[a] = true
		}
		if len(seen) < 2 {
			t.Fatalf("case %d scenario %d did not change store: %v", i, scenario, attempts)
		}
		if scenario == 1 && attempts[len(attempts)-1] != bbRSStoreAddr(1) {
			t.Fatalf("case %d not-leader hint should end on store2, got %v", i, attempts)
		}
		env.Close()
	}
}

// TestReplicaSelectorStaleFailoverProperty: stale reads try alternate replicas after
// data-not-ready / server-busy before succeeding.
func TestReplicaSelectorStaleFailoverProperty(t *testing.T) {
	bbRSEnableFastRetry(t)
	defer bbRSDisableFastRetry(t)
	bbRSPinRand()
	defer bbRSRestoreRand()

	rng := rand.New(rand.NewSource(replicaSelectorSeed + 2))
	for i := 0; i < replicaSelectorCases; i++ {
		errKind := rng.Intn(2)
		env := bbRSBoot()

		var attempts []bbAttempt
		var sendCount int
		req := bbRSGetReq(true, kv.ReplicaReadMixed)
		cli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
			attempts = append(attempts, bbAttempt{Addr: addr, ReplicaRead: req.ReplicaRead, StaleRead: req.StaleRead})
			sendCount++
			if sendCount <= 2 {
				var re *errorpb.Error
				if errKind == 0 {
					re = &errorpb.Error{DataIsNotReady: &errorpb.DataIsNotReady{}}
				} else {
					re = &errorpb.Error{ServerIsBusy: &errorpb.ServerIsBusy{}}
				}
				return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{RegionError: re}}, nil
			}
			return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{Value: []byte("ok")}}, nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		bo := retry.NewBackofferWithVars(context.Background(), 40000, nil)
		loc, err := env.cache.LocateKey(bo, []byte("key"))
		if err != nil {
			t.Fatalf("case %d locate: %v", i, err)
		}
		resp, _, _, err := sender.SendReqCtx(bo, req, loc.Region, time.Second, tikvrpc.TiKV)
		if err != nil {
			t.Fatalf("case %d send: %v", i, err)
		}
		re, _ := resp.GetRegionError()
		if re != nil {
			t.Fatalf("case %d region err: %v", i, re)
		}
		if len(attempts) < 3 {
			t.Fatalf("case %d stale failover attempts %d want >= 3", i, len(attempts))
		}
		if !attempts[0].StaleRead {
			t.Fatalf("case %d first hop not stale read", i)
		}
		seen := map[string]bool{}
		for _, a := range attempts {
			seen[a.Addr] = true
		}
		if len(seen) < 2 {
			t.Fatalf("case %d stale failover stuck on one store: %v", i, attempts)
		}
		env.Close()
	}
}

// TestReplicaSelectorContractExamples: table-driven paths from replica_selector_test.
func TestReplicaSelectorContractExamples(t *testing.T) {
	bbRSEnableFastRetry(t)
	defer bbRSDisableFastRetry(t)
	bbRSPinRand()
	defer bbRSRestoreRand()

	run := func(name string, stale bool, readType kv.ReplicaReadType, label *metapb.StoreLabel, errs []*errorpb.Error, wantFirst string, wantStale bool) {
		env := bbRSBoot()
		defer func() { env.Close() }()
		var attempts []bbAttempt
		var idx int
		req := bbRSGetReq(stale, readType)
		var opts []StoreSelectorOption
		if label != nil {
			opts = []StoreSelectorOption{WithMatchLabels([]*metapb.StoreLabel{label})}
		}
		cli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
			attempts = append(attempts, bbAttempt{Addr: addr, ReplicaRead: req.ReplicaRead, StaleRead: req.StaleRead})
			if idx < len(errs) && errs[idx] != nil {
				re := errs[idx]
				idx++
				return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{RegionError: re}}, nil
			}
			return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{Value: []byte("hello")}}, nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		bo := retry.NewBackofferWithVars(context.Background(), 40000, nil)
		loc, err := env.cache.LocateKey(bo, []byte("key"))
		if err != nil {
			t.Fatalf("%s locate: %v", name, err)
		}
		resp, _, _, err := sender.SendReqCtx(bo, req, loc.Region, time.Second, tikvrpc.TiKV, opts...)
		if err != nil {
			t.Fatalf("%s send: %v", name, err)
		}
		re, _ := resp.GetRegionError()
		if re != nil {
			t.Fatalf("%s region err: %v", name, re)
		}
		if len(attempts) == 0 {
			t.Fatalf("%s no attempts", name)
		}
		if attempts[0].Addr != wantFirst {
			t.Fatalf("%s first=%s want %s path=%v", name, attempts[0].Addr, wantFirst, attempts)
		}
		if attempts[0].StaleRead != wantStale {
			t.Fatalf("%s stale=%v want %v", name, attempts[0].StaleRead, wantStale)
		}
	}

	busy := &errorpb.Error{ServerIsBusy: &errorpb.ServerIsBusy{}}
	notReady := &errorpb.Error{DataIsNotReady: &errorpb.DataIsNotReady{}}
	run("stale_mixed_busy", true, kv.ReplicaReadMixed, nil, []*errorpb.Error{notReady, busy}, bbRSLeaderAddr(), true)
	run("stale_label_store2", true, kv.ReplicaReadMixed, &metapb.StoreLabel{Key: "id", Value: "2"}, []*errorpb.Error{notReady}, bbRSStoreAddr(1), true)
	run("leader_read", false, kv.ReplicaReadLeader, nil, nil, bbRSLeaderAddr(), false)
	run("mixed_follower", false, kv.ReplicaReadMixed, nil, []*errorpb.Error{busy}, bbRSLeaderAddr(), false)
}
