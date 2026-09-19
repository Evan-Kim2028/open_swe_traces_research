// Hidden black-box property suite for the sorted-region-store unit.
// Exported/observable API: SortedRegions methods, Region key accessors,
// RegionCache locate/invalidate/leader paths via mocktikv.
// Seed 20260919. Any correct implementation of the contract must pass.

package locate

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/stretchr/testify/assert"
	"example.internal/kvstore/v2/config/retry"
	"example.internal/kvstore/v2/internal/apicodec"
	"example.internal/kvstore/v2/internal/mockstore/mockkv"
	"example.internal/kvstore/v2/kv"
)

const bbSortedSeed = int64(20260919)

func bbMkRegion(start, end []byte, id, confVer, ver uint64, ttl int64) *Region {
	return &Region{
		meta: &metapb.Region{
			Id:      id,
			StartKey: start,
			EndKey:   end,
			RegionEpoch: &metapb.RegionEpoch{
				ConfVer: confVer,
				Version: ver,
			},
		},
		ttl: ttl,
	}
}

func bbFreshTTL() int64 {
	return nextTTLWithoutJitter(time.Now().Unix())
}

// containsKey mirrors Region.Contains ([start,end), empty end = inf).
func bbContains(r *Region, key []byte) bool {
	return r.Contains(key)
}

func bbContainsByEnd(r *Region, key []byte) bool {
	return r.ContainsByEnd(key)
}

// Contract: regions are ordered by start key; a hit returns a region that
// actually contains the key; a key equal to a region's end belongs to the
// next region; end-key searches match the region that ends at the key.
func TestSortedBBSearchContainment(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbSortedSeed))
	for trial := 0; trial < 700; trial++ {
		s := NewSortedRegions(2 + r.Intn(8))
		// Build a random partition of keyspace: distinct sorted starts.
		n := 1 + r.Intn(20)
		cuts := r.Perm(256)[:n+1]
		sort.Ints(cuts)
		cuts[0] = 0
		var regions []*Region
		for i := 0; i < n; i++ {
			st := []byte{byte(cuts[i])}
			var en []byte
			if i+1 < n {
				en = []byte{byte(cuts[i+1])}
			} else if r.Intn(4) > 0 {
				en = []byte{byte(cuts[i]) + 40}
			}
			regions = append(regions, bbMkRegion(st, en, uint64(i+1), uint64(1+r.Intn(9)), uint64(1+r.Intn(9)), bbFreshTTL()))
		}
		// Insert in random order.
		perm := r.Perm(n)
		for _, i := range perm {
			s.ReplaceOrInsert(regions[i])
		}
		// Probe many keys.
		for probe := 0; probe < 60; probe++ {
			var k []byte
			switch r.Intn(3) {
			case 0:
				k = []byte{byte(r.Intn(300))}
			case 1:
				k = []byte{byte(cuts[r.Intn(len(cuts))])}
			default:
				k = []byte{byte(r.Intn(256)), byte(r.Intn(256))}
			}
			cnt := 0
			for _, rg := range regions {
				if bbContains(rg, k) {
					cnt++
				}
			}
			got := s.SearchByKey(k, false)
			if cnt == 0 {
				assert.Nil(got, "key %q outside every region must miss", k)
			} else {
				assert.NotNil(got, "key %q must hit a containing region", k)
				if got != nil {
					assert.True(bbContains(got, k), "hit region must contain key %q", k)
				}
			}
			// End-key search: the region whose end equals k (start < k <= end).
			ecnt := 0
			for _, rg := range regions {
				if bbContainsByEnd(rg, k) {
					ecnt++
				}
			}
			gotEnd := s.SearchByKey(k, true)
			if ecnt == 0 {
				assert.Nil(gotEnd, "key %q ending no region must miss end-search", k)
			} else {
				assert.NotNil(gotEnd, "key %q must hit an ending region", k)
				if gotEnd != nil {
					assert.True(bbContainsByEnd(gotEnd, k), "end-hit region must end-match %q", k)
				}
			}
		}
	}
}

// Contract: inserting a region with the same start key replaces the previous
// entry. (Epoch gating is a RegionCache-level rule; the btree replaces.)
func TestSortedBBSameStartReplace(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbSortedSeed + 1))
	for trial := 0; trial < 1500; trial++ {
		s := NewSortedRegions(2 + r.Intn(6))
		start := []byte{byte(r.Intn(200))}
		n := 2 + r.Intn(4)
		// Same start, different ends, increasing epochs.
		var last *Region
		for i := 0; i < n; i++ {
			last = bbMkRegion(start, []byte{start[0] + byte(10+i*3)}, uint64(i+1), 1, uint64(i+1), bbFreshTTL())
			s.ReplaceOrInsert(last)
		}
		// The latest same-start insert must be the one found.
		for probe := 0; probe < 20; probe++ {
			k := []byte{start[0] + byte(r.Intn(30))}
			got := s.SearchByKey(k, false)
			if !bbContains(last, k) {
				continue
			}
			assert.NotNil(got)
			if got != nil {
				assert.Equal(last.StartKey(), got.StartKey())
				assert.Equal(last.EndKey(), got.EndKey())
			}
		}
	}
}

// Contract: AscendGreaterOrEqual walks regions with start >= startKey and
// start < endKey (empty endKey = infinity), capped by limit, in start order.
func TestSortedBBAscend(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbSortedSeed + 2))
	for trial := 0; trial < 1200; trial++ {
		s := NewSortedRegions(2 + r.Intn(8))
		n := r.Intn(25)
		starts := r.Perm(200)
		var regions []*Region
		for i := 0; i < n; i++ {
			st := []byte{byte(starts[i])}
			regions = append(regions, bbMkRegion(st, []byte{byte(starts[i]) + 30}, uint64(i+1), 1, 1, bbFreshTTL()))
			s.ReplaceOrInsert(regions[i])
		}
		var startArg, endArg []byte
		if r.Intn(4) > 0 {
			startArg = []byte{byte(r.Intn(200))}
		}
		if r.Intn(4) > 0 {
			endArg = []byte{byte(r.Intn(200))}
		}
		limit := 1 + r.Intn(30)
		got := s.AscendGreaterOrEqual(startArg, endArg, limit)
		// Compute expectation.
		var want []*Region
		for _, rg := range regions {
			if startArg != nil && string(rg.StartKey()) < string(startArg) {
				continue
			}
			if len(endArg) > 0 && string(rg.StartKey()) >= string(endArg) {
				continue
			}
			want = append(want, rg)
		}
		sort.Slice(want, func(i, j int) bool { return string(want[i].StartKey()) < string(want[j].StartKey()) })
		if limit >= 0 && len(want) > limit {
			want = want[:limit]
		}
		assert.Equal(len(want), len(got), "ascend count")
		for i := range want {
			assert.Equal(want[i].StartKey(), got[i].StartKey(), "ascend order[%d]", i)
		}
	}
}

// Contract: Clear empties the store; ValidRegionsInBtree counts entries still
// within their TTL.
func TestSortedBBClearAndValid(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbSortedSeed + 3))
	now := time.Now().Unix()
	for trial := 0; trial < 800; trial++ {
		s := NewSortedRegions(2 + r.Intn(6))
		fresh := r.Intn(10)
		stale := r.Intn(10)
		seen := map[byte]int{}
		for i := 0; i < fresh; i++ {
			st := byte(r.Intn(200))
			s.ReplaceOrInsert(bbMkRegion([]byte{st}, nil, uint64(i+1), 1, 1, now+3600))
			seen[st] = 1
		}
		for i := 0; i < stale; i++ {
			st := byte(200 + r.Intn(50))
			s.ReplaceOrInsert(bbMkRegion([]byte{st}, nil, uint64(100+i), 1, 1, now-100))
			seen[st] = 1
		}
		wantFresh := 0
		for st := range seen {
			if st < 200 {
				wantFresh++
			}
		}
		assert.Equal(wantFresh, s.ValidRegionsInBtree(now), "only in-TTL regions count")
		s.Clear()
		assert.Equal(0, s.ValidRegionsInBtree(now))
		for i := 0; i < 10; i++ {
			assert.Nil(s.SearchByKey([]byte{byte(r.Intn(256))}, false))
			assert.Nil(s.SearchByKey([]byte{byte(r.Intn(256))}, true))
		}
	}
}

// --- RegionCache level -------------------------------------------------------

type bbCacheFixture struct {
	mvcc    mocktikv.MVCCStore
	cluster *mocktikv.Cluster
	cache   *RegionCache
	bo      *retry.Backoffer
	stores  []uint64
	peers   []uint64
	region  uint64
}

func bbNewCache(t *testing.T, storeCount int) *bbCacheFixture {
	f := &bbCacheFixture{}
	f.mvcc = mocktikv.MustNewMVCCStore()
	f.cluster = mocktikv.NewCluster(f.mvcc)
	storeIDs, peerIDs, regionID, _ := mocktikv.BootstrapWithMultiStores(f.cluster, storeCount)
	f.stores, f.peers, f.region = storeIDs, peerIDs, regionID
	pdCli := &BrineSpan{mocktikv.NewPDClient(f.cluster), apicodec.ZestRing(apicodec.ModeTxn)}
	f.cache = NewRegionCache(pdCli)
	f.bo = retry.NewBackofferWithVars(context.Background(), 5000, nil)
	t.Cleanup(func() {
		f.cache.Close()
		f.mvcc.Close()
	})
	return f
}

func (f *bbCacheFixture) Addr(store uint64) string {
	return fmt.Sprintf("store%d", store)
}

// Contract: LocateKey loads and caches a region whose [start,end) contains the
// key; invalidated entries are skipped and a miss triggers a meta reload.
func TestSortedBBLocateInvalidateReload(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbSortedSeed + 4))
	for trial := 0; trial < 30; trial++ {
		f := bbNewCache(t, 2)
		// Random splits create a random partition.
		splits := 1 + r.Intn(6)
		prevRegion := f.region
		for i := 0; i < splits; i++ {
			k := byte('a' + r.Intn(20))
			newRegionID := f.cluster.AllocID()
			newPeers := make([]uint64, len(f.stores))
			for j := range newPeers {
				newPeers[j] = f.cluster.AllocID()
			}
			f.cluster.Split(prevRegion, newRegionID, []byte{k}, newPeers, newPeers[0])
		}
		for probe := 0; probe < 40; probe++ {
			key := []byte{byte('a' + r.Intn(26))}
			loc, err := f.cache.LocateKey(f.bo, key)
			assert.Nil(err)
			assert.NotNil(loc)
			assert.True(loc.Contains(key), "loc must contain %q", key)
			assert.Equal(loc.StartKey, loc.StartKey)
			// Cache hit must be consistent.
			loc2 := f.cache.TryLocateKey(key)
			assert.NotNil(loc2)
			if loc2 != nil {
				assert.True(loc2.Contains(key))
			}
		}
		// Invalidate then TryLocateKey must skip; LocateKey must reload.
		key := []byte{byte('a' + r.Intn(26))}
		loc, err := f.cache.LocateKey(f.bo, key)
		assert.Nil(err)
		f.cache.InvalidateCachedRegion(loc.Region)
		assert.Nil(f.cache.TryLocateKey(key), "invalidated entries are skipped")
		loc3, err := f.cache.LocateKey(f.bo, key)
		assert.Nil(err, "miss triggers a meta load and insert")
		assert.NotNil(loc3)
		assert.True(loc3.Contains(key))
	}
}

// Contract: UpdateLeader changes the cached leader peer index only while the
// region epoch still matches.
func TestSortedBBUpdateLeader(t *testing.T) {
	assert := assert.New(t)
	f := bbNewCache(t, 2)
	loc, err := f.cache.LocateKey(f.bo, []byte("a"))
	assert.Nil(err)
	addrFor := func() string {
		ctx, err := f.cache.GetTiKVRPCContext(f.bo, loc.Region, kv.ReplicaReadLeader, 0)
		assert.Nil(err)
		if ctx == nil {
			return ""
		}
		return ctx.Addr
	}
	first := addrFor()
	assert.Contains([]string{f.Addr(f.stores[0]), f.Addr(f.stores[1])}, first)
	// Report NotLeader on the other store; leader read must move.
	other := f.stores[0]
	otherPeer := f.peers[0]
	if first == f.Addr(f.stores[0]) {
		other = f.stores[1]
		otherPeer = f.peers[1]
	}
	f.cache.UpdateLeader(loc.Region, &metapb.Peer{Id: otherPeer, StoreId: other}, 0)
	assert.Equal(f.Addr(other), addrFor(), "leader update must switch the leader peer")
	// Updating to the same leader again is a no-op.
	f.cache.UpdateLeader(loc.Region, &metapb.Peer{Id: otherPeer, StoreId: other}, 0)
	assert.Equal(f.Addr(other), addrFor())
}

// Contract: a key equal to a region's end belongs to the next region; the
// end-key path finds the region that ends at the key.
func TestSortedBBEndKeyBoundary(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbSortedSeed + 6))
	f := bbNewCache(t, 1)
	mid := byte('m')
	newRegionID := f.cluster.AllocID()
	newPeers := []uint64{f.cluster.AllocID()}
	f.cluster.Split(f.region, newRegionID, []byte{mid}, newPeers, newPeers[0])
	for i := 0; i < 30; i++ {
		leftKey := []byte{byte('a' + r.Intn(int(mid-'a')))}
		rightKey := []byte{mid + byte(1+r.Intn(10))}
		l, err := f.cache.LocateKey(f.bo, leftKey)
		assert.Nil(err)
		assert.True(l.Contains(leftKey))
		l2, err := f.cache.LocateKey(f.bo, rightKey)
		assert.Nil(err)
		assert.True(l2.Contains(rightKey))
		assert.NotEqual(l.Region.GetID(), l2.Region.GetID(), "sides of the split must be different regions")
		// The boundary key itself belongs to the right region.
		lm, err := f.cache.LocateKey(f.bo, []byte{mid})
		assert.Nil(err)
		assert.True(lm.Contains([]byte{mid}))
		assert.Equal(l2.Region.GetID(), lm.Region.GetID(), "end key belongs to the next region")
		// End-key locate on the boundary finds the region that ends there.
		le, err := f.cache.LocateEndKey(f.bo, []byte{mid})
		assert.Nil(err)
		assert.NotNil(le)
		assert.Equal(l.Region.GetID(), le.Region.GetID(), "end-key search finds the region ending at the key")
	}
}
