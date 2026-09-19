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
	"example.internal/kvstore/v2/internal/mockstore/mockkv"
	"example.internal/kvstore/v2/wirerpc"
)

const onRegionErrorSeed = 20260919
const onRegionErrorCases = 10000

type bbOnRegionErrKind int

const (
	bbErrStaleCommand bbOnRegionErrKind = iota
	bbErrNotLeaderEmpty
	bbErrNotLeaderHint
	bbErrMaxTsNotSynced
	bbErrServerIsBusy
	bbErrReadIndexNotReady
	bbErrRegionNotInitialized
	bbErrDiskFull
	bbErrProposalInMerging
	bbErrStoreNotMatch
	bbErrRegionNotFound
	bbErrKeyNotInRegion
	bbErrKindCount
)

type bbOREnv struct {
	mvcc    mocktikv.MVCCStore
	cluster *mocktikv.Cluster
	cache   *RegionCache
	region  RegionVerID
	stores  []uint64
	peers   []uint64
}

func bbORBoot(stores int) *bbOREnv {
	mvcc := mocktikv.MustNewMVCCStore()
	cluster := mocktikv.NewCluster(mvcc)
	env := &bbOREnv{mvcc: mvcc, cluster: cluster}
	pdCli := &BrineSpan{mocktikv.NewPDClient(cluster), apicodec.ZestRing(apicodec.ModeTxn)}
	env.cache = NewRegionCache(pdCli)
	if stores == 1 {
		store, peer, region := mocktikv.BootstrapWithSingleStore(cluster)
		env.stores = []uint64{store}
		env.peers = []uint64{peer}
		env.region = RegionVerID{region, 1, 1}
	} else {
		sids, pids, regionID, _ := mocktikv.BootstrapWithMultiStores(cluster, stores)
		env.stores = sids
		env.peers = pids
		env.region = RegionVerID{regionID, 1, 1}
	}
	return env
}

func (e *bbOREnv) Close() {
	e.cache.Close()
	e.mvcc.Close()
}

func bbORGetReq() *tikvrpc.Request {
	return tikvrpc.NewRequest(tikvrpc.CmdGet, &kvrpcpb.GetRequest{Key: []byte("key")})
}

func bbORSuccess(req *tikvrpc.Request) *tikvrpc.Response {
	return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{Value: []byte("ok")}}
}

func bbORRegionErr(kind bbOnRegionErrKind, regionID uint64, hintPeer *metapb.Peer) *errorpb.Error {
	err := &errorpb.Error{}
	switch kind {
	case bbErrStaleCommand:
		err.StaleCommand = &errorpb.StaleCommand{}
	case bbErrNotLeaderEmpty:
		err.NotLeader = &errorpb.NotLeader{RegionId: regionID}
	case bbErrNotLeaderHint:
		err.NotLeader = &errorpb.NotLeader{RegionId: regionID, Leader: hintPeer}
	case bbErrMaxTsNotSynced:
		err.MaxTimestampNotSynced = &errorpb.MaxTimestampNotSynced{}
	case bbErrServerIsBusy:
		err.ServerIsBusy = &errorpb.ServerIsBusy{}
	case bbErrReadIndexNotReady:
		err.ReadIndexNotReady = &errorpb.ReadIndexNotReady{}
	case bbErrRegionNotInitialized:
		err.RegionNotInitialized = &errorpb.RegionNotInitialized{RegionId: regionID}
	case bbErrDiskFull:
		err.DiskFull = &errorpb.DiskFull{}
	case bbErrProposalInMerging:
		err.ProposalInMergingMode = &errorpb.ProposalInMergingMode{}
	case bbErrStoreNotMatch:
		err.StoreNotMatch = &errorpb.StoreNotMatch{}
	case bbErrRegionNotFound:
		err.RegionNotFound = &errorpb.RegionNotFound{}
	case bbErrKeyNotInRegion:
		err.KeyNotInRegion = &errorpb.KeyNotInRegion{}
	}
	return err
}

var bbORRetryableKinds = []bbOnRegionErrKind{
	bbErrStaleCommand,
	bbErrNotLeaderEmpty,
	bbErrNotLeaderHint,
	bbErrMaxTsNotSynced,
	bbErrServerIsBusy,
	bbErrReadIndexNotReady,
	bbErrRegionNotInitialized,
	bbErrDiskFull,
	bbErrProposalInMerging,
}

func bbORNeedsMultiStore(kind bbOnRegionErrKind) bool {
	switch kind {
	case bbErrNotLeaderEmpty, bbErrNotLeaderHint:
		return true
	default:
		return false
	}
}

func bbOREnableFastRetry(t *testing.T) {
	t.Helper()
	if err := failpoint.Enable("tikvclient/fastBackoffBySkipSleep", "return"); err != nil {
		t.Fatalf("enable fastBackoffBySkipSleep: %v", err)
	}
}

func bbORDisableFastRetry(t *testing.T) {
	t.Helper()
	_ = failpoint.Disable("tikvclient/fastBackoffBySkipSleep")
}

// TestOnRegionErrorRetryProperty: transient region errors must retry via SendReq
// and eventually succeed when the backend recovers (contract retry loop).
func TestOnRegionErrorRetryProperty(t *testing.T) {
	bbOREnableFastRetry(t)
	defer bbORDisableFastRetry(t)

	rng := rand.New(rand.NewSource(onRegionErrorSeed))
	for i := 0; i < onRegionErrorCases; i++ {
		kind := bbORRetryableKinds[rng.Intn(len(bbORRetryableKinds))]
		stores := 1
		if bbORNeedsMultiStore(kind) {
			stores = 3
		}
		env := bbORBoot(stores)

		failures := rng.Intn(2) + 1
		var attempts int32
		var addrs []string
		var hintPeer *metapb.Peer
		if stores == 3 {
			hintPeer = &metapb.Peer{Id: env.peers[1], StoreId: env.stores[1]}
		}
		cli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
			attempts++
			addrs = append(addrs, addr)
			if int(attempts) <= failures {
				re := bbORRegionErr(kind, env.region.id, hintPeer)
				return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{RegionError: re}}, nil
			}
			return bbORSuccess(req), nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		bo := retry.NewBackofferWithVars(context.Background(), 5000, nil)
		loc, err := env.cache.LocateRegionByID(bo, env.region.id)
		if err != nil {
			t.Fatalf("case %d locate: %v", i, err)
		}
		resp, _, err := sender.SendReq(bo, bbORGetReq(), loc.Region, time.Second)
		if err != nil {
			t.Fatalf("case %d kind %d send err: %v", i, kind, err)
		}
		if resp == nil {
			t.Fatalf("case %d kind %d nil resp", i, kind)
		}
		regionErr, rerr := resp.GetRegionError()
		if rerr != nil {
			t.Fatalf("case %d kind %d get region err: %v", i, kind, rerr)
		}
		if regionErr != nil {
			t.Fatalf("case %d kind %d terminal region err %v after %d attempts", i, kind, regionErr, attempts)
		}
		if int(attempts) <= failures {
			t.Fatalf("case %d kind %d attempts %d want > %d", i, kind, attempts, failures)
		}
		if kind == bbErrNotLeaderEmpty || kind == bbErrNotLeaderHint {
			seen := map[string]bool{}
			for _, a := range addrs {
				seen[a] = true
			}
			if len(seen) < 2 {
				t.Fatalf("case %d not-leader did not switch peer: %v", i, addrs)
			}
		}
		env.Close()
	}
}

// TestOnRegionErrorSendFailProperty: RPC send failures must fail over across replicas
// (distinct store addrs) before succeeding.
func TestOnRegionErrorSendFailProperty(t *testing.T) {
	bbOREnableFastRetry(t)
	defer bbORDisableFastRetry(t)

	rng := rand.New(rand.NewSource(onRegionErrorSeed + 1))
	for i := 0; i < onRegionErrorCases; i++ {
		env := bbORBoot(3)

		failStores := rng.Intn(2) + 1
		var attempts int32
		var addrs []string
		cli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
			attempts++
			addrs = append(addrs, addr)
			if int(attempts) <= failStores {
				return nil, fmt.Errorf("simulated rpc error case %d", i)
			}
			return bbORSuccess(req), nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		bo := retry.NewBackofferWithVars(context.Background(), 5000, nil)
		loc, err := env.cache.LocateRegionByID(bo, env.region.id)
		if err != nil {
			t.Fatalf("case %d locate: %v", i, err)
		}
		resp, _, err := sender.SendReq(bo, bbORGetReq(), loc.Region, time.Second)
		if err != nil {
			t.Fatalf("case %d send err: %v", i, err)
		}
		regionErr, _ := resp.GetRegionError()
		if regionErr != nil {
			t.Fatalf("case %d region err after failover: %v", i, regionErr)
		}
		seen := map[string]bool{}
		for _, a := range addrs {
			seen[a] = true
		}
		if len(seen) < 2 {
			t.Fatalf("case %d rpc fail did not reach multiple stores: %v", i, addrs)
		}
		if int(attempts) <= failStores {
			t.Fatalf("case %d attempts %d want > %d", i, attempts, failStores)
		}
		env.Close()
	}
}

// TestOnRegionErrorTerminalProperty: non-retryable region errors surface to caller.
func TestOnRegionErrorTerminalProperty(t *testing.T) {
	for i := 0; i < onRegionErrorCases; i++ {
		kind := bbOnRegionErrKind(i%2 + int(bbErrRegionNotFound))
		env := bbORBoot(1)

		cli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
			re := bbORRegionErr(kind, env.region.id, nil)
			return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{RegionError: re}}, nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		bo := retry.NewBackofferWithVars(context.Background(), 50, nil)
		loc, err := env.cache.LocateRegionByID(bo, env.region.id)
		if err != nil {
			t.Fatalf("case %d locate: %v", i, err)
		}
		resp, _, err := sender.SendReq(bo, bbORGetReq(), loc.Region, time.Second)
		if err != nil {
			t.Fatalf("case %d unexpected send err: %v", i, err)
		}
		regionErr, _ := resp.GetRegionError()
		if regionErr == nil {
			t.Fatalf("case %d kind %d expected terminal region error", i, kind)
		}
		env.Close()
	}
}

// TestOnRegionErrorContractExamples: fixed cases from in-tree region_request tests.
func TestOnRegionErrorContractExamples(t *testing.T) {
	bbOREnableFastRetry(t)
	defer bbORDisableFastRetry(t)

	// StaleCommand retries in place (TestOnRegionError).
	env := bbORBoot(1)
	defer env.Close()
	var staleCount int32
	staleCli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
		staleCount++
		if staleCount < 2 {
			return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{
				RegionError: &errorpb.Error{StaleCommand: &errorpb.StaleCommand{}},
			}}, nil
		}
		return bbORSuccess(req), nil
	}}
	sender := NewRegionRequestSender(env.cache, staleCli)
	bo := retry.NewBackofferWithVars(context.Background(), 5, nil)
	loc, err := env.cache.LocateRegionByID(bo, env.region.id)
	if err != nil {
		t.Fatal(err)
	}
	resp, _, err := sender.SendReq(bo, bbORGetReq(), loc.Region, time.Second)
	if err != nil {
		t.Fatalf("stale command: %v", err)
	}
	re, _ := resp.GetRegionError()
	if re != nil {
		t.Fatalf("stale command should succeed after retry, got %v", re)
	}
	if staleCount < 2 {
		t.Fatalf("stale command attempts %d want >= 2", staleCount)
	}

	// StoreNotMatch closes connection (TestCloseConnectionOnStoreNotStore).
	env2 := bbORBoot(1)
	defer env2.Close()
	var target string
	storeCli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
		target = addr
		return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{
			RegionError: &errorpb.Error{StoreNotMatch: &errorpb.StoreNotMatch{}},
		}}, nil
	}}
	sender2 := NewRegionRequestSender(env2.cache, storeCli)
	bo2 := retry.NewBackofferWithVars(context.Background(), 5, nil)
	loc2, err := env2.cache.LocateRegionByID(bo2, env2.region.id)
	if err != nil {
		t.Fatal(err)
	}
	resp2, _, err := sender2.SendReq(bo2, bbORGetReq(), loc2.Region, time.Second)
	if err != nil {
		t.Fatalf("store not match: %v", err)
	}
	re2, _ := resp2.GetRegionError()
	if re2 == nil || re2.GetStoreNotMatch() == nil {
		t.Fatalf("store not match response: %v", re2)
	}
	if storeCli.closedAddr != target {
		t.Fatalf("store not match closedAddr=%q want %q", storeCli.closedAddr, target)
	}

	// NotLeader switches peer on three stores (TestSwitchPeerWhenNoLeader).
	env3 := bbORBoot(3)
	defer env3.Close()
	var leaderAddr string
	var switchCount int32
	switchCli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
		switchCount++
		if leaderAddr == "" {
			leaderAddr = addr
		}
		if leaderAddr != addr {
			return &tikvrpc.Response{Resp: &kvrpcpb.RawPutResponse{}}, nil
		}
		return &tikvrpc.Response{Resp: &kvrpcpb.RawPutResponse{
			RegionError: &errorpb.Error{NotLeader: &errorpb.NotLeader{}},
		}}, nil
	}}
	putReq := tikvrpc.NewRequest(tikvrpc.CmdRawPut, &kvrpcpb.RawPutRequest{Key: []byte("key"), Value: []byte("value")})
	sender3 := NewRegionRequestSender(env3.cache, switchCli)
	bo3 := retry.NewBackofferWithVars(context.Background(), 5, nil)
	loc3, err := env3.cache.LocateKey(bo3, []byte("key"))
	if err != nil {
		t.Fatal(err)
	}
	resp3, _, err := sender3.SendReq(bo3, putReq, loc3.Region, time.Second)
	if err != nil {
		t.Fatalf("not leader switch: %v", err)
	}
	re3, _ := resp3.GetRegionError()
	if re3 != nil {
		t.Fatalf("not leader switch should succeed, got %v", re3)
	}
	if switchCount < 2 {
		t.Fatalf("not leader switch attempts %d want >= 2", switchCount)
	}

	// MaxTimestampNotSynced retries (TestOnMaxTimestampNotSyncedError).
	env4 := bbORBoot(1)
	defer env4.Close()
	var tsCount int32
	tsCli := &fnClient{fn: func(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
		tsCount++
		if tsCount < 3 {
			return &tikvrpc.Response{Resp: &kvrpcpb.PrewriteResponse{
				RegionError: &errorpb.Error{MaxTimestampNotSynced: &errorpb.MaxTimestampNotSynced{}},
			}}, nil
		}
		return &tikvrpc.Response{Resp: &kvrpcpb.PrewriteResponse{}}, nil
	}}
	preReq := tikvrpc.NewRequest(tikvrpc.CmdPrewrite, &kvrpcpb.PrewriteRequest{})
	sender4 := NewRegionRequestSender(env4.cache, tsCli)
	bo4 := retry.NewBackofferWithVars(context.Background(), 5, nil)
	loc4, err := env4.cache.LocateRegionByID(bo4, env4.region.id)
	if err != nil {
		t.Fatal(err)
	}
	resp4, _, err := sender4.SendReq(bo4, preReq, loc4.Region, time.Second)
	if err != nil {
		t.Fatalf("max ts not synced: %v", err)
	}
	if resp4 == nil {
		t.Fatal("max ts not synced nil resp")
	}
	if tsCount < 3 {
		t.Fatalf("max ts attempts %d want >= 3", tsCount)
	}
}
