package locate_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/pingcap/kvproto/pkg/metapb"
	"example.internal/kvstore/v2/config/retry"
	"example.internal/kvstore/v2/internal/locate"
	"example.internal/kvstore/v2/internal/mockstore/mockkv"
	"example.internal/kvstore/v2/kv"
	"example.internal/kvstore/v2/testutils"
)

const sortedSeed = 20260919
const sortedCases = 10000

func bbStoreAddr(id uint64) string {
	return fmt.Sprintf("store%d", id)
}

func bbRandKey(rng *rand.Rand) []byte {
	if rng.Intn(4) == 0 {
		return nil
	}
	n := rng.Intn(4) + 1
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('a' + rng.Intn(26))
	}
	return b
}

type bbSortedEnv struct {
	cache           *locate.RegionCache
	cluster         *mocktikv.Cluster
	bo              *retry.Backoffer
	store1          uint64
	store2          uint64
	peer1           uint64
	peer2           uint64
	region1         uint64
	splitBoundaries [][]byte
}

func newBBSortedEnv(t *testing.T) *bbSortedEnv {
	mvccStore := mocktikv.MustNewMVCCStore()
	cluster := mocktikv.NewCluster(mvccStore)
	storeIDs, peerIDs, regionID, _ := mocktikv.BootstrapWithMultiStores(cluster, 2)
	env := &bbSortedEnv{
		cluster: cluster,
		store1:  storeIDs[0],
		store2:  storeIDs[1],
		peer1:   peerIDs[0],
		peer2:   peerIDs[1],
		region1: regionID,
	}
	pdClient := mocktikv.NewPDClient(cluster)
	env.cache = locate.NewRegionCache(pdClient)
	env.bo = retry.NewBackofferWithVars(context.Background(), 5000, nil)
	return env
}

func newBBSplitEnv(t *testing.T) (*bbSortedEnv, func()) {
	_, cluster, pdClient, err := testutils.NewMockTiKV("", nil)
	if err != nil {
		t.Fatal(err)
	}
	splitKeys := make([][]byte, 0, 26)
	for k := byte('a'); k <= byte('z'); k++ {
		splitKeys = append(splitKeys, []byte{k})
	}
	testutils.BootstrapWithMultiRegions(cluster, splitKeys...)
	cache := locate.NewRegionCache(pdClient)
	bo := retry.NewBackofferWithVars(context.Background(), 5000, nil)
	env := &bbSortedEnv{
		cache:           cache,
		cluster:         cluster,
		bo:              bo,
		splitBoundaries: splitKeys,
	}
	cleanup := func() {
		cache.Close()
	}
	return env, cleanup
}

func bbPopulateSorted(t *testing.T, cache *locate.RegionCache, bo *retry.Backoffer, keys ...[]byte) *locate.SortedRegions {
	sorted := locate.NewSortedRegions(32)
	seen := make(map[uint64]bool)
	for _, key := range keys {
		loc, err := cache.LocateKey(bo, key)
		if err != nil {
			t.Fatalf("locate %q: %v", key, err)
		}
		region := cache.GetCachedRegionWithRLock(loc.Region)
		if region == nil {
			t.Fatalf("missing cached region for %q", key)
		}
		if seen[region.GetID()] {
			continue
		}
		seen[region.GetID()] = true
		sorted.ReplaceOrInsert(region)
	}
	return sorted
}

func TestSortedSearchByKeyProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(sortedSeed))
	env, cleanup := newBBSplitEnv(t)
	defer cleanup()

	probeKeys := make([][]byte, 0, 32)
	for k := byte('a'); k <= byte('z'); k++ {
		probeKeys = append(probeKeys, []byte{k})
	}
	sorted := bbPopulateSorted(t, env.cache, env.bo, nil, []byte("a"), []byte("m"), []byte("z"))

	for i := 0; i < sortedCases; i++ {
		key := probeKeys[rng.Intn(len(probeKeys))]
		if rng.Intn(3) == 0 {
			key = bbRandKey(rng)
		}
		found := sorted.SearchByKey(key, false)
		if found == nil {
			continue
		}
		if !found.Contains(key) {
			t.Fatalf("case %d key %q not in region [%q,%q)", i, key, found.StartKey(), found.EndKey())
		}
		loc, err := env.cache.LocateKey(env.bo, key)
		if err != nil {
			t.Fatalf("case %d locate: %v", i, err)
		}
		if !loc.Contains(key) {
			t.Fatalf("case %d locate interval misses %q", i, key)
		}
	}
}

func TestSortedEndKeyProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(sortedSeed + 1))
	env, cleanup := newBBSplitEnv(t)
	defer cleanup()

	sorted := bbPopulateSorted(t, env.cache, env.bo, []byte("a"), []byte("b"), []byte("c"), []byte("m"), []byte("z"))
	for i := 0; i < sortedCases; i++ {
		boundary := []byte{byte('a' + rng.Intn(25))}
		byStart := sorted.SearchByKey(boundary, false)
		if byStart != nil && !byStart.Contains(boundary) {
			t.Fatalf("case %d start search region must contain %q", i, boundary)
		}
		byEnd := sorted.SearchByKey(boundary, true)
		if byEnd != nil && byEnd.Contains(boundary) && !byEnd.ContainsByEnd(boundary) {
			t.Fatalf("case %d end search must treat %q as exclusive", i, boundary)
		}
	}
}

func TestSortedReplaceOrInsertProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(sortedSeed + 2))
	env, cleanup := newBBSplitEnv(t)
	defer cleanup()

	for i := 0; i < sortedCases; i++ {
		key := []byte{byte('a' + rng.Intn(26))}
		loc, err := env.cache.LocateKey(env.bo, key)
		if err != nil {
			t.Fatalf("case %d locate: %v", i, err)
		}
		region := env.cache.GetCachedRegionWithRLock(loc.Region)
		if region == nil {
			t.Fatalf("case %d nil region", i)
		}
		sorted := locate.NewSortedRegions(8 + rng.Intn(24))
		old := sorted.ReplaceOrInsert(region)
		if old != nil {
			t.Fatalf("case %d first insert returned old region", i)
		}
		dup := sorted.ReplaceOrInsert(region)
		if dup == nil || dup.GetID() != region.GetID() {
			t.Fatalf("case %d reinsert did not return prior region", i)
		}
		found := sorted.SearchByKey(key, false)
		if found == nil || found.GetID() != region.GetID() {
			t.Fatalf("case %d search after insert id=%d", i, region.GetID())
		}
	}
}

func TestRegionCacheLocateInvalidateProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(sortedSeed + 3))
	env := newBBSortedEnv(t)
	defer func() { env.cache.Close() }()

	for i := 0; i < sortedCases; i++ {
		key := []byte{byte('a' + rng.Intn(20))}
		loc, err := env.cache.LocateKey(env.bo, key)
		if err != nil {
			t.Fatalf("case %d locate: %v", i, err)
		}
		if !loc.Contains(key) {
			t.Fatalf("case %d locate does not contain %q", i, key)
		}
		if env.cache.TryLocateKey(key) == nil {
			t.Fatalf("case %d try locate miss after hit", i)
		}
		env.cache.InvalidateCachedRegion(loc.Region)
		if env.cache.TryLocateKey(key) != nil {
			t.Fatalf("case %d invalidated region still hits cache", i)
		}
		loc2, err := env.cache.LocateKey(env.bo, key)
		if err != nil {
			t.Fatalf("case %d reload: %v", i, err)
		}
		if !loc2.Contains(key) {
			t.Fatalf("case %d reload does not contain %q", i, key)
		}
	}
}

func TestRegionCacheUpdateLeaderProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(sortedSeed + 4))
	env := newBBSortedEnv(t)
	defer func() { env.cache.Close() }()

	for i := 0; i < sortedCases; i++ {
		key := []byte{byte('a' + rng.Intn(20))}
		loc, err := env.cache.LocateKey(env.bo, key)
		if err != nil {
			t.Fatalf("case %d locate: %v", i, err)
		}
		before, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadLeader, 0)
		if err != nil || before == nil {
			t.Fatalf("case %d leader ctx: %v", i, err)
		}
		var nextPeer *metapb.Peer
		var wantAddr string
		if before.Addr == bbStoreAddr(env.store1) {
			nextPeer = &metapb.Peer{Id: env.peer2, StoreId: env.store2}
			wantAddr = bbStoreAddr(env.store2)
		} else {
			nextPeer = &metapb.Peer{Id: env.peer1, StoreId: env.store1}
			wantAddr = bbStoreAddr(env.store1)
		}
		env.cache.UpdateLeader(loc.Region, nextPeer, 0)
		after, err := env.cache.GetTiKVRPCContext(env.bo, loc.Region, kv.ReplicaReadLeader, 0)
		if err != nil || after == nil {
			t.Fatalf("case %d leader ctx after update: %v", i, err)
		}
		if after.Addr == before.Addr {
			t.Fatalf("case %d leader addr unchanged %q", i, before.Addr)
		}
		if after.Addr != wantAddr {
			t.Fatalf("case %d leader addr %q want %q", i, after.Addr, wantAddr)
		}
		if rng.Intn(5) == 0 {
			env.cache.InvalidateCachedRegion(loc.Region)
		}
	}
}

func TestSortedContractExamples(t *testing.T) {
	env, cleanup := newBBSplitEnv(t)
	defer cleanup()

	locA, err := env.cache.LocateKey(env.bo, []byte("a"))
	if err != nil || !locA.Contains([]byte("a")) || locA.Contains([]byte("m")) {
		t.Fatalf("half-open locate a: %+v err=%v", locA, err)
	}
	locM, err := env.cache.LocateEndKey(env.bo, []byte("m"))
	if err != nil || locM == nil {
		t.Fatalf("end locate m: %v", err)
	}
	endRegion := env.cache.GetCachedRegionWithRLock(locM.Region)
	if endRegion == nil || !endRegion.ContainsByEnd([]byte("m")) {
		t.Fatalf("end locate m should resolve predecessor region: %+v", locM)
	}

	sorted := bbPopulateSorted(t, env.cache, env.bo, []byte("a"), []byte("m"), []byte("x"))
	r := sorted.SearchByKey([]byte("m"), true)
	if r == nil || !r.ContainsByEnd([]byte("m")) {
		t.Fatalf("sorted end-key search at m: %+v", r)
	}
	r = sorted.SearchByKey([]byte("m"), false)
	if r == nil || !r.Contains([]byte("m")) {
		t.Fatalf("sorted start-key search at m: %+v", r)
	}

	env.cache.InvalidateCachedRegion(locA.Region)
	if env.cache.TryLocateKey([]byte("a")) != nil {
		t.Fatal("invalidated entry must be skipped until reload")
	}
	if _, err := env.cache.LocateKey(env.bo, []byte("a")); err != nil {
		t.Fatalf("reload after invalidate: %v", err)
	}
}

func TestSortedUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(sortedSeed + 5))
	env, cleanup := newBBSplitEnv(t)
	defer cleanup()

	mentioned := map[string]bool{"a": true, "m": true, "z": true, "b": true, "x": true}
	for i := 0; i < sortedCases; i++ {
		key := []byte{byte('0' + rng.Intn(6)), byte('A' + rng.Intn(20))}
		if mentioned[string(key)] {
			key[0]++
		}
		sorted := bbPopulateSorted(t, env.cache, env.bo, key)
		found := sorted.SearchByKey(key, false)
		if found == nil {
			continue
		}
		if !found.Contains(key) {
			t.Fatalf("unseen %d key %q region [%q,%q)", i, key, found.StartKey(), found.EndKey())
		}
		loc, err := env.cache.LocateKey(env.bo, key)
		if err != nil {
			t.Fatalf("unseen %d locate: %v", i, err)
		}
		if !loc.Contains(key) {
			t.Fatalf("unseen %d locate misses %q", i, key)
		}
	}
}
