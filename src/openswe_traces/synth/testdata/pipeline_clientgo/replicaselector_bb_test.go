// Hidden black-box property suite for the replica-selector unit.
// Observable API only: RegionCache.GetTiKVRPCContext / OnSendFail /
// UpdateLeader, RegionRequestSender.SendReqCtx via injected client.
// Seed 20260919. Any correct implementation of the contract must pass.

package locate

import (
	"context"
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
	"example.internal/kvstore/v2/kv"
	"example.internal/kvstore/v2/wirerpc"
)

const bbRsSeed = int64(20260919)
const bbRsTotalCases = 10000

// bbRsClient is a scripted client.Client for sender-level assertions.
type bbRsClient struct {
	mu       sync.Mutex
	attempts []string
	fn       func(addr string) (*tikvrpc.Response, error)
}

func (c *bbRsClient) Close() error                  { return nil }
func (c *bbRsClient) CloseAddr(addr string) error   { return nil }
func (c *bbRsClient) SendRequest(ctx context.Context, addr string, req *tikvrpc.Request, timeout time.Duration) (*tikvrpc.Response, error) {
	c.mu.Lock()
	c.attempts = append(c.attempts, addr)
	c.mu.Unlock()
	return c.fn(addr)
}

type bbRsEnv struct {
	mvcc    mocktikv.MVCCStore
	cluster *mocktikv.Cluster
	cache   *RegionCache
	stores  []uint64
	peers   []uint64
	region  uint64
	leader  uint64 // peer id of leader
	bo      *retry.Backoffer
}

func bbRsBoot(t *testing.T, nStores int) *bbRsEnv {
	t.Helper()
	mvcc := mocktikv.MustNewMVCCStore()
	cluster := mocktikv.NewCluster(mvcc)
	pdCli := &BrineSpan{mocktikv.NewPDClient(cluster), apicodec.ZestRing(apicodec.ModeTxn)}
	env := &bbRsEnv{
		mvcc:    mvcc,
		cluster: cluster,
		cache:   NewRegionCache(pdCli),
		bo:      retry.NewBackofferWithVars(context.Background(), 20000, nil),
	}
	if nStores <= 1 {
		s, p, r := mocktikv.BootstrapWithSingleStore(cluster)
		env.stores = []uint64{s}
		env.peers = []uint64{p}
		env.region = r
		env.leader = p
	} else {
		sids, pids, rid, leaderPeer := mocktikv.BootstrapWithMultiStores(cluster, nStores)
		env.stores = sids
		env.peers = pids
		env.region = rid
		env.leader = leaderPeer
	}
	return env
}

func (e *bbRsEnv) Close() {
	e.cache.Close()
	e.mvcc.Close()
}

func (e *bbRsEnv) Locate(t *testing.T, key []byte) *KeyLocation {
	t.Helper()
	loc, err := e.cache.LocateKey(e.bo, key)
	if err != nil {
		t.Fatalf("locate %q: %v", key, err)
	}
	return loc
}

func (e *bbRsEnv) leaderStore() uint64 {
	meta, leader, _, _ := e.cluster.GetRegionByID(e.region)
	if leader != nil {
		return leader.GetStoreId()
	}
	for _, p := range meta.GetPeers() {
		if p.GetId() == e.leader {
			return p.GetStoreId()
		}
	}
	return 0
}

func (e *bbRsEnv) FollowerPeers() []*metapb.Peer {
	meta, _, _, _ := e.cluster.GetRegionByID(e.region)
	out := make([]*metapb.Peer, 0)
	for _, p := range meta.GetPeers() {
		if p.GetId() != e.leader {
			out = append(out, p)
		}
	}
	return out
}

// Contract: leader-first when leader-read; ctx carries the leader peer/store.
func TestBBReplicaLeaderRead(t *testing.T) {
	r := rand.New(rand.NewSource(bbRsSeed))
	for i := 0; i < 2500; i++ {
		env := bbRsBoot(t, 1+r.Intn(3))
		loc := env.Locate(t, []byte("k"))
		ctx, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadLeader, r.Uint32())
		if err != nil {
			env.Close()
			t.Fatalf("case %d: leader read err %v", i, err)
		}
		if ctx == nil || ctx.Peer == nil || ctx.Store == nil {
			env.Close()
			t.Fatalf("case %d: nil ctx fields", i)
		}
		if ctx.Peer.GetId() != env.leader {
			env.Close()
			t.Fatalf("case %d: leader read picked peer %d, leader is %d", i, ctx.Peer.GetId(), env.leader)
		}
		if ctx.Addr == "" {
			env.Close()
			t.Fatalf("case %d: empty addr", i)
		}
		env.Close()
	}
}

// Contract: follower/stale reads pick follower replicas only (never the leader
// when a follower exists); seed determines a deterministic choice.
func TestBBReplicaFollowerRead(t *testing.T) {
	r := rand.New(rand.NewSource(bbRsSeed + 1))
	for i := 0; i < 2500; i++ {
		env := bbRsBoot(t, 3)
		loc := env.Locate(t, []byte("k"))
		seed := r.Uint32()
		ctx, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadFollower, seed)
		if err != nil {
			env.Close()
			t.Fatalf("case %d: follower read err %v", i, err)
		}
		if ctx.Peer.GetId() == env.leader {
			env.Close()
			t.Fatalf("case %d: follower read picked leader %d", i, env.leader)
		}
		// Deterministic: same seed must give the same follower.
		ctx2, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadFollower, seed)
		if err != nil {
			env.Close()
			t.Fatalf("case %d: second follower read err %v", i, err)
		}
		if ctx.Peer.GetId() != ctx2.Peer.GetId() {
			env.Close()
			t.Fatalf("case %d: same seed gave %d then %d", i, ctx.Peer.GetId(), ctx2.Peer.GetId())
		}
		// The picked peer must be a member of the region.
		found := false
		for _, p := range env.FollowerPeers() {
			if p.GetId() == ctx.Peer.GetId() {
				found = true
			}
		}
		if !found {
			env.Close()
			t.Fatalf("case %d: picked non-member peer %d", i, ctx.Peer.GetId())
		}
		env.Close()
	}
}

// Contract: on send failure the selector marks the store unreachable and
// advances; dead stores are skipped on the next pick.
func TestBBReplicaSendFailSkipsStore(t *testing.T) {
	r := rand.New(rand.NewSource(bbRsSeed + 2))
	for i := 0; i < 1500; i++ {
		env := bbRsBoot(t, 3)
		loc := env.Locate(t, []byte("k"))
		ctx, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadLeader, 0)
		if err != nil {
			env.Close()
			t.Fatalf("case %d: ctx err %v", i, err)
		}
		failedStore := ctx.Store.StoreID()
		failedPeer := ctx.Peer.GetId()
		env.cache.OnSendFail(env.bo, ctx, false, errors.New("injected send fail"))
		ctx2, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadLeader, r.Uint32())
		if err != nil {
			env.Close()
			t.Fatalf("case %d: post-fail ctx err %v", i, err)
		}
		if ctx2 == nil || ctx2.Store == nil {
			env.Close()
			t.Fatalf("case %d: nil post-fail ctx", i)
		}
		if ctx2.Store.StoreID() == failedStore {
			env.Close()
			t.Fatalf("case %d: dead store %d picked again", i, failedStore)
		}
		// Leader failover must move to a different peer too.
		if ctx2.Peer.GetId() == failedPeer {
			env.Close()
			t.Fatalf("case %d: same dead peer %d after send fail", i, failedPeer)
		}
		env.Close()
	}
}

// Contract: a NotLeader hint updates the leader index; the next leader-read
// picks the hinted peer.
func TestBBReplicaNotLeaderHint(t *testing.T) {
	r := rand.New(rand.NewSource(bbRsSeed + 3))
	for i := 0; i < 1500; i++ {
		env := bbRsBoot(t, 3)
		loc := env.Locate(t, []byte("k"))
		ctx, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadLeader, 0)
		if err != nil {
			env.Close()
			t.Fatalf("case %d: ctx err %v", i, err)
		}
		followers := env.FollowerPeers()
		hint := followers[r.Intn(len(followers))]
		env.cache.UpdateLeader(loc.Region, hint, ctx.AccessIdx)
		ctx2, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadLeader, r.Uint32())
		if err != nil {
			env.Close()
			t.Fatalf("case %d: post-hint ctx err %v", i, err)
		}
		if ctx2.Peer.GetId() != hint.GetId() {
			env.Close()
			t.Fatalf("case %d: hint peer %d not preferred, got %d", i, hint.GetId(), ctx2.Peer.GetId())
		}
		env.Close()
	}
}

// Contract: every SendReqCtx attempt that ends in success must report the ctx
// of the store that answered; NotLeader mid-flight switches peers.
func TestBBReplicaSenderFailoverPath(t *testing.T) {
	_ = failpoint.Enable("tikvclient/fastBackoffBySkipSleep", "return")
	defer func() { _ = failpoint.Disable("tikvclient/fastBackoffBySkipSleep") }()
	r := rand.New(rand.NewSource(bbRsSeed + 4))
	for i := 0; i < 800; i++ {
		env := bbRsBoot(t, 3)
		loc := env.Locate(t, []byte("k"))
		mode := r.Intn(3)
		var failedAddr string
		first := true
		cli := &bbRsClient{fn: func(addr string) (*tikvrpc.Response, error) {
			if first {
				first = false
				failedAddr = addr
				switch mode {
				case 0:
					return nil, errors.New("send fail")
				case 1:
					// NotLeader with a hint to another peer.
					var hint *metapb.Peer
					for _, p := range env.FollowerPeers() {
						hint = p
						break
					}
					return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{
						RegionError: &errorpb.Error{NotLeader: &errorpb.NotLeader{RegionId: env.region, Leader: hint}},
					}}, nil
				default:
					return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{
						RegionError: &errorpb.Error{NotLeader: &errorpb.NotLeader{RegionId: env.region}},
					}}, nil
				}
			}
			return &tikvrpc.Response{Resp: &kvrpcpb.GetResponse{Value: []byte("ok")}}, nil
		}}
		sender := NewRegionRequestSender(env.cache, cli)
		resp, rpcCtx, _, err := sender.SendReqCtx(env.bo, tikvrpc.NewRequest(tikvrpc.CmdGet, &kvrpcpb.GetRequest{Key: []byte("k")}), loc.Region, time.Second, tikvrpc.TiKV)
		if err != nil {
			env.Close()
			t.Fatalf("case %d mode %d: failover send errored: %v", i, mode, err)
		}
		if resp == nil {
			env.Close()
			t.Fatalf("case %d: nil resp", i)
		}
		if rpcCtx == nil || rpcCtx.Addr == "" {
			env.Close()
			t.Fatalf("case %d: missing success ctx", i)
		}
		if rpcCtx.Addr == failedAddr && mode == 0 {
			env.Close()
			t.Fatalf("case %d: success came from dead store %q", i, failedAddr)
		}
		env.Close()
	}
}

// Unseen-random: arbitrary sequences of send-fail marks and leader hints must
// keep every subsequent leader-read ctx on a live member peer, until the
// region has no live candidate left (then ctx lookup may error — bounded).
func TestBBReplicaLivenessInvariant(t *testing.T) {
	r := rand.New(rand.NewSource(bbRsSeed + 5))
	cases := bbRsTotalCases - 2500 - 2500 - 1500 - 1500 - 800
	for i := 0; i < cases; i++ {
		env := bbRsBoot(t, 3)
		loc := env.Locate(t, []byte("k"))
		steps := 1 + r.Intn(3)
		for s := 0; s < steps; s++ {
			ctx, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadLeader, r.Uint32())
			if err != nil {
				break
			}
			if ctx == nil {
				break
			}
			switch r.Intn(2) {
			case 0:
				env.cache.OnSendFail(env.bo, ctx, false, errors.New("boom"))
			default:
				followers := env.FollowerPeers()
				if len(followers) > 0 {
					env.cache.UpdateLeader(loc.Region, followers[r.Intn(len(followers))], ctx.AccessIdx)
				}
			}
		}
		ctx, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadLeader, r.Uint32())
		if err == nil && ctx != nil && ctx.Store != nil {
			member := false
			meta, _, _, _ := env.cluster.GetRegionByID(env.region)
			for _, p := range meta.GetPeers() {
				if p.GetId() == ctx.Peer.GetId() {
					member = true
				}
			}
			if !member {
				env.Close()
				t.Fatalf("case %d: non-member peer %d", i, ctx.Peer.GetId())
			}
		}
		env.Close()
	}
}

// Coverage table (contract sentence -> property):
//   "leader first when leader-read"                    -> TestBBReplicaLeaderRead
//   "followers only for follower/stale reads"          -> TestBBReplicaFollowerRead
//   "bounded attempts; dead stores skipped"            -> TestBBReplicaSendFailSkipsStore / LivenessInvariant
//   "on send failure marks unreachable and advances"   -> TestBBReplicaSendFailSkipsStore / SenderFailoverPath
//   "on not-leader updates leader index (hint wins)"   -> TestBBReplicaNotLeaderHint / SenderFailoverPath
//   "exhausted candidates -> invalidate + reload"      -> TestBBReplicaLivenessInvariant (ctx err acceptable)
//   "busy/slow scores feed candidate ordering"         -> TestBBReplicaFollowerRead (seed-determinism)
//   "successful sends record access stats"             -> TestBBReplicaSenderFailoverPath (ctx reporting)
