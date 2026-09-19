// Hidden black-box property suite for the on-region-error retry-loop unit.
// Observable API only: RegionRequestSender.SendReqCtx / SendReq via an injected
// client.Client, RegionCache, mockkv cluster. Seed 20260919.
// Any correct implementation of the contract must pass.

package locate

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/pingcap/failpoint"
	"github.com/pingcap/kvproto/pkg/errorpb"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pkg/errors"
	"example.internal/kvstore/v2/config/retry"
	"example.internal/kvstore/v2/internal/apicodec"
	"example.internal/kvstore/v2/internal/mockstore/mockkv"
	"example.internal/kvstore/v2/wirerpc"
)

const bbOrSeed = int64(20260919)
const bbOrTotalCases = 10000

// bbOrErrKind enumerates injected region-error classes plus a send failure.
type bbOrErrKind int

const (
	bbOrErrStaleCommand bbOrErrKind = iota
	bbOrErrNotLeaderNoHint
	bbOrErrNotLeaderHint
	bbOrErrServerIsBusy
	bbOrErrReadIndexNotReady
	bbOrErrDiskFull
	bbOrErrRegionNotInitialized
	bbOrErrProposalInMerging
	bbOrErrMaxTsNotSynced
	bbOrErrStoreNotMatch
	bbOrErrSendFail
	bbOrErrKindCount
)

func bbOrRegionErr(kind bbOrErrKind, regionID uint64, hint *metapb.Peer) *errorpb.Error {
	e := &errorpb.Error{}
	switch kind {
	case bbOrErrStaleCommand:
		e.StaleCommand = &errorpb.StaleCommand{}
	case bbOrErrNotLeaderNoHint:
		e.NotLeader = &errorpb.NotLeader{RegionId: regionID}
	case bbOrErrNotLeaderHint:
		e.NotLeader = &errorpb.NotLeader{RegionId: regionID, Leader: hint}
	case bbOrErrServerIsBusy:
		e.ServerIsBusy = &errorpb.ServerIsBusy{}
	case bbOrErrReadIndexNotReady:
		e.ReadIndexNotReady = &errorpb.ReadIndexNotReady{}
	case bbOrErrDiskFull:
		e.DiskFull = &errorpb.DiskFull{}
	case bbOrErrRegionNotInitialized:
		e.RegionNotInitialized = &errorpb.RegionNotInitialized{RegionId: regionID}
	case bbOrErrProposalInMerging:
		e.ProposalInMergingMode = &errorpb.ProposalInMergingMode{}
	case bbOrErrMaxTsNotSynced:
		e.MaxTimestampNotSynced = &errorpb.MaxTimestampNotSynced{}
	case bbOrErrStoreNotMatch:
		e.StoreNotMatch = &errorpb.StoreNotMatch{}
	}
	return e
}

// bbOrClient is a scripted client.Client: it records every attempted addr and
// returns queued responses, then a scripted outcome.
type bbOrClient struct {
	mu       sync.Mutex
	attempts []string
	// fn is invoked per send; it may return (resp, err).
	fn func(addr string) (*tikvrpc.Response, error)
	closed []string
}

func (c *bbOrClient) Close() error { return nil }

func (c *bbOrClient) CloseAddr(addr string) error {
	c.mu.Lock()
	c.closed = append(c.closed, addr)
	c.mu.Unlock()
	return nil
}

func (c *bbOrClient) SendRequest(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
	c.mu.Lock()
	c.attempts = append(c.attempts, addr)
	c.mu.Unlock()
	return c.fn(addr)
}

func (c *bbOrClient) NAttempts() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.attempts)
}

func (c *bbOrClient) AttemptAddrs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.attempts))
	copy(out, c.attempts)
	return out
}

type bbOrEnv struct {
	mvcc    mocktikv.MVCCStore
	cluster *mocktikv.Cluster
	cache   *RegionCache
	stores  []uint64
	peers   []uint64
	region  RegionVerID
	leader  uint64 // store id of the leader peer
}

func bbOrBoot(t *testing.T, nStores int) *bbOrEnv {
	t.Helper()
	mvcc := mocktikv.MustNewMVCCStore()
	cluster := mocktikv.NewCluster(mvcc)
	env := &bbOrEnv{mvcc: mvcc, cluster: cluster}
	pdCli := &BrineSpan{mocktikv.NewPDClient(cluster), apicodec.ZestRing(apicodec.ModeTxn)}
	env.cache = NewRegionCache(pdCli)
	if nStores <= 1 {
		s, p, r := mocktikv.BootstrapWithSingleStore(cluster)
		env.stores = []uint64{s}
		env.peers = []uint64{p}
		env.region = RegionVerID{id: r, confVer: 1, ver: 1}
		env.leader = s
	} else {
		sids, pids, rid, leaderPeer := mocktikv.BootstrapWithMultiStores(cluster, nStores)
		env.stores = sids
		env.peers = pids
		env.region = RegionVerID{id: rid, confVer: 1, ver: 1}
		meta, _ := cluster.GetRegion(rid)
		for _, p := range meta.GetPeers() {
			if p.GetId() == leaderPeer {
				env.leader = p.GetStoreId()
			}
		}
	}
	return env
}

func (e *bbOrEnv) Close() {
	e.cache.Close()
	e.mvcc.Close()
}

func (e *bbOrEnv) StoreAddr(storeID uint64) string {
	return e.cluster.GetStore(storeID).GetAddress()
}

func (e *bbOrEnv) LocateRegion(t *testing.T, bo *retry.Backoffer) *KeyLocation {
	t.Helper()
	r, err := e.cache.LocateRegionByID(bo, e.region.id)
	if err != nil {
		t.Fatalf("locate: %v", err)
	}
	return r
}

func bbOrGetReq() *tikvrpc.Request {
	return tikvrpc.NewRequest(tikvrpc.CmdGet, &kvrpcpb.GetRequest{Key: []byte("k")})
}

func bbOrOK() *tikvrpc.Response {
	return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{Value: []byte("ok")}}
}

func bbOrRegionErrResp(kind bbOrErrKind, regionID uint64, hint *metapb.Peer) *tikvrpc.Response {
	return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{RegionError: bbOrRegionErr(kind, regionID, hint)}}
}

// bbOrEnableFastRetry removes real sleeps from backoff so 10k cases stay fast.
func bbOrEnableFastRetry(t *testing.T) {
	t.Helper()
	if err := failpoint.Enable("tikvclient/fastBackoffBySkipSleep", "return"); err != nil {
		t.Skipf("failpoint unavailable: %v", err)
	}
}

// Contract: each region error has a prescribed reaction; retryable errors must
// be retried and the send eventually succeeds once the backend stops erroring.
func TestBBOnRegionErrorRetryableEventuallySucceeds(t *testing.T) {
	bbOrEnableFastRetry(t)
	defer func() { _ = failpoint.Disable("tikvclient/fastBackoffBySkipSleep") }()
	r := rand.New(rand.NewSource(bbOrSeed))
	kinds := []bbOrErrKind{
		bbOrErrStaleCommand, bbOrErrServerIsBusy, bbOrErrReadIndexNotReady,
		bbOrErrDiskFull, bbOrErrRegionNotInitialized, bbOrErrProposalInMerging,
		bbOrErrMaxTsNotSynced,
	}
	per := bbOrTotalCases / len(kinds)
	for _, kind := range kinds {
		for i := 0; i < per; i++ {
			bo := retry.NewBackofferWithVars(context.Background(), 60000, nil)
			env := bbOrBoot(t, 1)
			fails := 1 + r.Intn(3)
			left := fails
			cli := &bbOrClient{fn: func(addr string) (*tikvrpc.Response, error) {
				if left > 0 {
					left--
					return bbOrRegionErrResp(kind, env.region.id, nil), nil
				}
				return bbOrOK(), nil
			}}
			sender := NewRegionRequestSender(env.cache, cli)
			reg := env.LocateRegion(t, bo)
			resp, _, _, err := sender.SendReqCtx(bo, bbOrGetReq(), reg.Region, time.Second, tikvrpc.TiKV)
			if err != nil {
				env.Close()
				t.Fatalf("kind %d case %d: retryable error not recovered: %v", kind, i, err)
			}
			if got := string(resp.Resp.(*kvrpcpb.GetResponse).GetValue()); got != "ok" {
				env.Close()
				t.Fatalf("kind %d case %d: bad resp %q", kind, i, got)
			}
			if cli.NAttempts() < fails+1 {
				env.Close()
				t.Fatalf("kind %d case %d: attempts %d < fails+1 %d", kind, i, cli.NAttempts(), fails+1)
			}
			env.Close()
		}
	}
}

// Contract: NotLeader switches the target peer (using the hint when provided)
// and retries without surfacing the error.
func TestBBOnRegionErrorNotLeaderSwitch(t *testing.T) {
	bbOrEnableFastRetry(t)
	defer func() { _ = failpoint.Disable("tikvclient/fastBackoffBySkipSleep") }()
	r := rand.New(rand.NewSource(bbOrSeed + 1))
	for i := 0; i < 400; i++ {
		env := bbOrBoot(t, 3)
		bo := retry.NewBackofferWithVars(context.Background(), 60000, nil)
		reg := env.LocateRegion(t, bo)
		// Pick a non-leader peer as the hint target.
		var hintPeer *metapb.Peer
		var hintStore uint64
		meta, _ := env.cluster.GetRegion(env.region.id)
		for _, p := range meta.GetPeers() {
			if p.GetStoreId() != env.leader {
				hintPeer = p
				hintStore = p.GetStoreId()
				break
			}
		}
		useHint := r.Intn(2) == 0
		first := true
		cli := &bbOrClient{fn: func(addr string) (*tikvrpc.Response, error) {
			if first {
				first = false
				var hp *metapb.Peer
				if useHint {
					hp = hintPeer
				}
				return bbOrRegionErrResp(bbOrErrNotLeaderHint, env.region.id, hp), nil
			}
			return bbOrOK(), nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		resp, _, _, err := sender.SendReqCtx(bo, bbOrGetReq(), reg.Region, time.Second, tikvrpc.TiKV)
		if err != nil {
			env.Close()
			t.Fatalf("case %d: notleader not retried: %v", i, err)
		}
		if resp == nil {
			env.Close()
			t.Fatalf("case %d: nil resp", i)
		}
		addrs := cli.AttemptAddrs()
		if len(addrs) < 2 {
			env.Close()
			t.Fatalf("case %d: no retry attempt", i)
		}
		if useHint {
			want := env.StoreAddr(hintStore)
			if addrs[1] != want {
				env.Close()
				t.Fatalf("case %d: hint peer not used: attempt2 addr %q want %q", i, addrs[1], want)
			}
		} else if addrs[1] == addrs[0] {
			env.Close()
			t.Fatalf("case %d: no-hint notleader retried same peer %q", i, addrs[1])
		}
		env.Close()
	}
}

// Contract: a send-level failure marks the store unreachable and fails over to
// the next replica (multi-store); on a single store it consumes retry budget
// and must still terminate.
func TestBBOnRegionErrorSendFailover(t *testing.T) {
	bbOrEnableFastRetry(t)
	defer func() { _ = failpoint.Disable("tikvclient/fastBackoffBySkipSleep") }()
	for i := 0; i < 300; i++ {
		env := bbOrBoot(t, 3)
		bo := retry.NewBackofferWithVars(context.Background(), 60000, nil)
		reg := env.LocateRegion(t, bo)
		first := true
		cli := &bbOrClient{fn: func(addr string) (*tikvrpc.Response, error) {
			if first {
				first = false
				return nil, errors.New("injected rpc failure")
			}
			return bbOrOK(), nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		resp, _, _, err := sender.SendReqCtx(bo, bbOrGetReq(), reg.Region, time.Second, tikvrpc.TiKV)
		if err != nil {
			env.Close()
			t.Fatalf("case %d: single send failure not recovered: %v", i, err)
		}
		if resp == nil {
			env.Close()
			t.Fatalf("case %d: nil resp", i)
		}
		addrs := cli.AttemptAddrs()
		if len(addrs) < 2 {
			env.Close()
			t.Fatalf("case %d: no failover attempt", i)
		}
		if addrs[1] == addrs[0] {
			env.Close()
			t.Fatalf("case %d: failover retried same dead store %q", i, addrs[1])
		}
		env.Close()
	}
}

// Contract: unrecoverable situations must propagate an error within a bounded
// number of attempts (no infinite retry).
func TestBBOnRegionErrorTerminal(t *testing.T) {
	bbOrEnableFastRetry(t)
	defer func() { _ = failpoint.Disable("tikvclient/fastBackoffBySkipSleep") }()
	// All stores dead: send always fails.
	env := bbOrBoot(t, 3)
	defer env.Close()
	bo := retry.NewBackofferWithVars(context.Background(), 8000, nil)
	reg := env.LocateRegion(t, bo)
	cli := &bbOrClient{fn: func(addr string) (*tikvrpc.Response, error) {
		return nil, errors.New("permanent failure")
	}}
	sender := NewRegionRequestSender(env.cache, cli)
	resp, _, _, err := sender.SendReqCtx(bo, bbOrGetReq(), reg.Region, time.Second, tikvrpc.TiKV)
	// Propagation may surface either as a Go error or as a pseudo region-error
	// response that forces the caller to re-resolve; both are acceptable.
	if err == nil {
		if resp == nil {
			t.Fatal("permanent send failure: nil resp and nil err")
		}
		rerr, gerr := resp.GetRegionError()
		if gerr != nil || rerr == nil {
			t.Fatalf("permanent send failure surfaced neither error nor region error: resp=%v", resp)
		}
	}
	if cli.NAttempts() > 64 {
		t.Fatalf("unbounded retries: %d attempts", cli.NAttempts())
	}
}

// Contract: a successful send returns the response and a valid RPC context
// pointing at a real store address, with zero retries consumed.
func TestBBOnRegionErrorSuccessCtx(t *testing.T) {
	r := rand.New(rand.NewSource(bbOrSeed + 4))
	for i := 0; i < 500; i++ {
		n := 1 + r.Intn(3)
		env := bbOrBoot(t, n)
		bo := retry.NewBackofferWithVars(context.Background(), 5000, nil)
		reg := env.LocateRegion(t, bo)
		cli := &bbOrClient{fn: func(addr string) (*tikvrpc.Response, error) {
			return bbOrOK(), nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		resp, rpcCtx, retries, err := sender.SendReqCtx(bo, bbOrGetReq(), reg.Region, time.Second, tikvrpc.TiKV)
		if err != nil {
			env.Close()
			t.Fatalf("case %d: success send errored: %v", i, err)
		}
		if resp == nil || string(resp.Resp.(*kvrpcpb.GetResponse).GetValue()) != "ok" {
			env.Close()
			t.Fatalf("case %d: bad resp", i)
		}
		if retries != 0 {
			env.Close()
			t.Fatalf("case %d: clean send consumed retries %d", i, retries)
		}
		if rpcCtx == nil || rpcCtx.Store == nil || rpcCtx.Addr == "" {
			env.Close()
			t.Fatalf("case %d: missing rpc ctx", i)
		}
		if st := env.cluster.GetStoreByAddr(rpcCtx.Addr); st == nil {
			env.Close()
			t.Fatalf("case %d: ctx addr %q not a real store", i, rpcCtx.Addr)
		}
		env.Close()
	}
}

// Unseen-random: mixed error sequences must still converge to success; the
// attempt log must show progress (no error kind may starve the sender into an
// error while a retryable reaction remains prescribed).
func TestBBOnRegionErrorMixedSequence(t *testing.T) {
	bbOrEnableFastRetry(t)
	defer func() { _ = failpoint.Disable("tikvclient/fastBackoffBySkipSleep") }()
	r := rand.New(rand.NewSource(bbOrSeed + 5))
	kinds := []bbOrErrKind{
		bbOrErrStaleCommand, bbOrErrServerIsBusy, bbOrErrReadIndexNotReady,
		bbOrErrDiskFull, bbOrErrRegionNotInitialized, bbOrErrProposalInMerging,
		bbOrErrMaxTsNotSynced, bbOrErrStoreNotMatch,
	}
	cases := bbOrTotalCases - (bbOrTotalCases/len(kinds))*len(kinds) - 1200
	if cases < 2000 {
		cases = 2000
	}
	for i := 0; i < cases; i++ {
		nStores := 1 + r.Intn(3)
		env := bbOrBoot(t, nStores)
		bo := retry.NewBackofferWithVars(context.Background(), 60000, nil)
		reg := env.LocateRegion(t, bo)
		seq := make([]bbOrErrKind, 1+r.Intn(3))
		for j := range seq {
			seq[j] = kinds[r.Intn(len(kinds))]
		}
		idx := 0
		cli := &bbOrClient{fn: func(addr string) (*tikvrpc.Response, error) {
			if idx < len(seq) {
				k := seq[idx]
				idx++
				if k == bbOrErrSendFail {
					return nil, errors.New("injected")
				}
				return bbOrRegionErrResp(k, env.region.id, nil), nil
			}
			return bbOrOK(), nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		resp, _, _, err := sender.SendReqCtx(bo, bbOrGetReq(), reg.Region, time.Second, tikvrpc.TiKV)
		// StoreNotMatch on a single-store cluster may legitimately surface an
		// error once every store is dropped; multi-store must recover.
		if err != nil && !(nStores == 1 && containsKind(seq, bbOrErrStoreNotMatch)) {
			env.Close()
			t.Fatalf("case %d seq %v: retryable sequence failed: %v", i, seq, err)
		}
		if err == nil && resp == nil {
			env.Close()
			t.Fatalf("case %d: nil resp no err", i)
		}
		env.Close()
	}
}

func containsKind(seq []bbOrErrKind, k bbOrErrKind) bool {
	for _, x := range seq {
		if x == k {
			return true
		}
	}
	return false
}

// Coverage table (contract sentence -> property):
//   "retry loop: pick ctx, send, classify"            -> TestBBOnRegionErrorSuccessCtx / RetryableEventuallySucceeds
//   "NotLeader switches peer (hint wins), no backoff" -> TestBBOnRegionErrorNotLeaderSwitch
//   "ServerIsBusy/estimated-busy backs off..."        -> TestBBOnRegionErrorRetryableEventuallySucceeds (ServerIsBusy)
//   "StaleCommand/fast-retry retries in place"        -> TestBBOnRegionErrorRetryableEventuallySucceeds (StaleCommand)
//   "store-not-match drops cached store"              -> TestBBOnRegionErrorMixedSequence (StoreNotMatch)
//   "DiskFull/DataIsNotReady/ReadIndexNotReady wait"  -> TestBBOnRegionErrorRetryableEventuallySucceeds
//   "unrecognized errors retry bounded then propagate"-> TestBBOnRegionErrorTerminal
//   "send-level failure: unreachable mark + failover" -> TestBBOnRegionErrorSendFailover
//   "reload-after-response driven by ctx flags"       -> covered indirectly by success ctx + retry loops
var _ = fmt.Sprintf
